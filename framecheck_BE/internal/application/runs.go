package application

import (
	"context"
	"errors"
	"strconv"
	"time"

	"framecheck/internal/domain"
	"framecheck/internal/ports"
)

// RunTimeout bounds a single provider analysis. There is no automatic retry.
const RunTimeout = 120 * time.Second

type StartRunInput struct {
	ComparisonID   string
	IdempotencyKey string
	// AllowDuplicate starts a new run even when a finished run already has the
	// same input fingerprint. Without it, duplicates are refused so a paid
	// provider is not called twice by accident.
	AllowDuplicate bool
}

// StartRun freezes the comparison inputs into a queued run. Created is false
// when an existing run is returned for a repeated Idempotency-Key.
func (s *Service) StartRun(ctx context.Context, in StartRunInput) (run domain.Run, created bool, err error) {
	if err := domain.ValidateIdempotencyKey(in.IdempotencyKey); err != nil {
		return domain.Run{}, false, err
	}
	cd, err := s.GetComparison(ctx, in.ComparisonID)
	if err != nil {
		return domain.Run{}, false, err
	}

	if in.IdempotencyKey != "" {
		existing, err := s.repo.FindRunByIdempotencyKey(ctx, in.ComparisonID, in.IdempotencyKey)
		if err != nil {
			return domain.Run{}, false, err
		}
		if existing != nil {
			return *existing, false, nil
		}
	}

	maxSide := 0
	if l, ok := s.provider.(ports.ImageSizeLimiter); ok {
		maxSide = l.MaxAnalysisSide()
	}
	transform := domain.NewTransformSpec(
		domain.FitTransform(cd.Reference.WidthPx, cd.Reference.HeightPx, maxSide),
		domain.FitTransform(cd.Actual.WidthPx, cd.Actual.HeightPx, maxSide),
	)
	fp := domain.FingerprintInput{
		ReferenceSHA256:   cd.Reference.SHA256,
		ActualSHA256:      cd.Actual.SHA256,
		RequirementSHA256: cd.Requirement.SHA256,
		Origin:            s.provider.Origin(),
		ModelID:           s.provider.ModelID(),
		ConfigVersion:     s.provider.ConfigVersion(),
		PromptVersion:     s.provider.PromptVersion(),
		Transform:         transform,
	}.Fingerprint()

	if !in.AllowDuplicate {
		dup, err := s.repo.FindReusableRun(ctx, fp)
		if err != nil {
			return domain.Run{}, false, err
		}
		if dup != nil {
			return domain.Run{}, false, domain.Conflict("DUPLICATE_RUN",
				"run %s already analysed identical inputs; open it or request a deliberate re-run", dup.ID)
		}
	}

	run = domain.Run{
		ID:               s.ids.NewID(),
		ProjectID:        cd.Comparison.ProjectID,
		ComparisonID:     cd.Comparison.ID,
		Status:           domain.RunQueued,
		Origin:           s.provider.Origin(),
		InputFingerprint: fp,
		IdempotencyKey:   in.IdempotencyKey,
		ModelID:          s.provider.ModelID(),
		ConfigVersion:    s.provider.ConfigVersion(),
		PromptVersion:    s.provider.PromptVersion(),
		Transform:        transform,
		CreatedAt:        s.clock.Now(),
	}
	if err := s.repo.CreateRun(ctx, run); err != nil {
		return domain.Run{}, false, err
	}

	select {
	case s.queue <- run.ID:
	default:
		s.finishFailed(context.WithoutCancel(ctx), run, "QUEUE_FULL", "analysis queue is full; start the run again later")
		return domain.Run{}, false, domain.Unavailable("QUEUE_FULL", "analysis queue is full; try again later")
	}
	return run, true, nil
}

// RecoverInterrupted fails runs a previous process left unfinished. Call it
// once at startup before Work.
func (s *Service) RecoverInterrupted(ctx context.Context) error {
	n, err := s.repo.FailInterruptedRuns(ctx, s.clock.Now())
	if err != nil {
		return err
	}
	if n > 0 {
		s.log.Warn("failed interrupted runs from previous process", "count", n)
	}
	return nil
}

// Work processes queued runs sequentially until ctx is cancelled.
func (s *Service) Work(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case id := <-s.queue:
			s.process(ctx, id)
		}
	}
}

func (s *Service) process(ctx context.Context, runID string) {
	run, err := s.repo.GetRun(ctx, runID)
	if err != nil {
		s.log.Error("load queued run", "run_id", runID, "err", err)
		return
	}
	started := s.clock.Now()
	if err := s.repo.MarkRunRunning(ctx, run.ID, started); err != nil {
		s.log.Error("mark run running", "run_id", run.ID, "err", err)
		return
	}
	run.Status = domain.RunRunning
	run.StartedAt = &started

	cd, err := s.GetComparison(ctx, run.ComparisonID)
	if err != nil {
		s.finishFailed(ctx, run, "INPUT_UNAVAILABLE", "comparison inputs could not be loaded")
		return
	}
	refData, err := s.loadAsset(ctx, cd.Reference)
	if err != nil {
		s.finishFailed(ctx, run, "INPUT_UNAVAILABLE", "reference image could not be loaded")
		return
	}
	actData, err := s.loadAsset(ctx, cd.Actual)
	if err != nil {
		s.finishFailed(ctx, run, "INPUT_UNAVAILABLE", "actual image could not be loaded")
		return
	}

	actx, cancel := context.WithTimeout(ctx, RunTimeout)
	out, err := s.provider.Analyze(actx, ports.AnalysisInput{
		Reference: ports.AnalysisImage{AssetID: cd.Reference.ID, Role: domain.RoleReference,
			MIME: cd.Reference.MIME, Data: refData, Transform: run.Transform.Reference},
		Actual: ports.AnalysisImage{AssetID: cd.Actual.ID, Role: domain.RoleActual,
			MIME: cd.Actual.MIME, Data: actData, Transform: run.Transform.Actual},
		Requirements: cd.Requirement.Text,
	})
	cancel()
	run.LatencyMS = s.clock.Now().Sub(started).Milliseconds()
	if err != nil {
		code, msg := "PROVIDER_ERROR", "the analysis provider failed; no result was recorded"
		var pe *ports.ProviderError
		switch {
		case errors.As(err, &pe):
			code, msg = pe.Code, pe.Message
		case errors.Is(err, context.DeadlineExceeded):
			code, msg = "PROVIDER_TIMEOUT", "the analysis provider did not answer in time"
		}
		// Provider errors may echo request content; log the error, persist only
		// the safe message.
		s.log.Warn("provider analysis failed", "run_id", run.ID, "err", err)
		run.ProviderRequestID = out.ProviderRequestID
		s.finishFailed(ctx, run, code, msg)
		return
	}
	run.ProviderRequestID = out.ProviderRequestID
	run.InputTokens, run.OutputTokens = out.InputTokens, out.OutputTokens

	ec := domain.EvidenceContext{
		Reference: cd.Reference, Actual: cd.Actual,
		ReferenceTransform: run.Transform.Reference, ActualTransform: run.Transform.Actual,
	}
	now := s.clock.Now()
	findings := make([]domain.Finding, 0, len(out.Findings))
	rejected := 0
	for i, raw := range out.Findings {
		f, err := domain.BuildFinding(raw, ec)
		if err != nil {
			rejected++
			run.Issues = append(run.Issues, "provider finding "+strconv.Itoa(i+1)+" rejected: "+err.Error())
			continue
		}
		f.ID = s.ids.NewID()
		f.RunID = run.ID
		f.CreatedAt = now
		findings = append(findings, f)
	}
	if !out.Complete {
		run.Issues = append(run.Issues, "provider did not report a complete analysis")
	}
	run.Status = domain.ResolveRunStatus(out.Complete, rejected, findings)
	run.FinishedAt = &now
	if err := s.repo.FinishRun(ctx, run, findings); err != nil {
		s.log.Error("store run result", "run_id", run.ID, "err", err)
		s.finishFailed(ctx, run, "STORE_FAILED", "the analysis result could not be stored")
	}
}

func (s *Service) finishFailed(ctx context.Context, run domain.Run, code, msg string) {
	now := s.clock.Now()
	run.Status = domain.RunFailed
	run.FailureCode = code
	run.FailureMessage = msg
	run.FinishedAt = &now
	if err := s.repo.FinishRun(ctx, run, nil); err != nil {
		s.log.Error("store failed run", "run_id", run.ID, "err", err)
	}
}

type RunDetail struct {
	Run      domain.Run
	Findings []domain.Finding
	// Reviews maps finding ID to its review history, oldest first.
	Reviews map[string][]domain.Review
}

func (s *Service) GetRun(ctx context.Context, id string) (RunDetail, error) {
	run, err := s.repo.GetRun(ctx, id)
	if err != nil {
		return RunDetail{}, err
	}
	findings, err := s.repo.ListFindings(ctx, id)
	if err != nil {
		return RunDetail{}, err
	}
	reviews, err := s.repo.ListReviewsForRun(ctx, id)
	if err != nil {
		return RunDetail{}, err
	}
	byFinding := make(map[string][]domain.Review, len(findings))
	for _, r := range reviews {
		byFinding[r.FindingID] = append(byFinding[r.FindingID], r)
	}
	return RunDetail{Run: run, Findings: findings, Reviews: byFinding}, nil
}

func (s *Service) ReviewFinding(ctx context.Context, findingID string, decision domain.Decision, reason string) (domain.Review, error) {
	reason, err := domain.ValidateReview(decision, reason)
	if err != nil {
		return domain.Review{}, err
	}
	f, err := s.repo.GetFinding(ctx, findingID)
	if err != nil {
		return domain.Review{}, err
	}
	run, err := s.repo.GetRun(ctx, f.RunID)
	if err != nil {
		return domain.Review{}, err
	}
	if !run.Status.Finished() {
		return domain.Review{}, domain.Conflict("RUN_NOT_FINISHED", "findings can be reviewed only after the run finishes")
	}
	r := domain.Review{
		ID: s.ids.NewID(), FindingID: f.ID, Decision: decision,
		Reason: reason, ReviewedAt: s.clock.Now(),
	}
	if err := s.repo.CreateReview(ctx, r); err != nil {
		return domain.Review{}, err
	}
	return r, nil
}
