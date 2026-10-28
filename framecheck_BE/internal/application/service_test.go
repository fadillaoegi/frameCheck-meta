package application

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/color"
	"image/png"
	"strings"
	"testing"
	"time"

	"framecheck/internal/domain"
	"framecheck/internal/ports"
)

type stubProvider struct {
	out ports.AnalysisOutput
	err error
}

func (*stubProvider) Origin() domain.RunOrigin { return domain.OriginFixture }
func (*stubProvider) ModelID() string          { return "stub" }
func (*stubProvider) ConfigVersion() string    { return "1" }
func (*stubProvider) PromptVersion() string    { return "1" }
func (p *stubProvider) Analyze(context.Context, ports.AnalysisInput) (ports.AnalysisOutput, error) {
	return p.out, p.err
}

func pngBytes(t *testing.T, w, h int, c color.Color) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, c)
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

var capture = domain.Capture{ViewportWidthCSS: 100, ViewportHeightCSS: 50, DevicePixelRatio: 2, ContentStateKey: "default"}

type fixture struct {
	svc      *Service
	repo     *memRepo
	provider *stubProvider
	project  domain.Project
	cmp      ComparisonDetail
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	ctx := context.Background()
	repo := newMemRepo()
	prov := &stubProvider{out: ports.AnalysisOutput{Complete: true}}
	svc := NewService(Deps{
		Repo: repo, Store: &memStore{data: map[string][]byte{}}, Provider: prov,
		Clock: &fixedClock{t: time.Date(2026, 10, 27, 0, 0, 0, 0, time.UTC)}, IDs: &seqIDs{},
	})
	p, err := svc.CreateProject(ctx, " Demo page ")
	if err != nil {
		t.Fatal(err)
	}
	ref, err := svc.IngestAsset(ctx, IngestAssetInput{ProjectID: p.ID, Role: domain.RoleReference, Capture: capture,
		Data: pngBytes(t, 200, 100, color.White)})
	if err != nil {
		t.Fatal(err)
	}
	act, err := svc.IngestAsset(ctx, IngestAssetInput{ProjectID: p.ID, Role: domain.RoleActual, Capture: capture,
		Data: pngBytes(t, 200, 100, color.Black)})
	if err != nil {
		t.Fatal(err)
	}
	cmp, err := svc.CreateComparison(ctx, CreateComparisonInput{ProjectID: p.ID,
		ReferenceAssetID: ref.ID, ActualAssetID: act.ID, Requirements: "Show the primary button"})
	if err != nil {
		t.Fatal(err)
	}
	return &fixture{svc: svc, repo: repo, provider: prov, project: p, cmp: cmp}
}

// runOnce starts a run and processes it synchronously.
func (f *fixture) runOnce(t *testing.T, in StartRunInput) RunDetail {
	t.Helper()
	ctx := context.Background()
	if in.ComparisonID == "" {
		in.ComparisonID = f.cmp.Comparison.ID
	}
	run, _, err := f.svc.StartRun(ctx, in)
	if err != nil {
		t.Fatal(err)
	}
	f.svc.process(ctx, <-f.svc.queue)
	rd, err := f.svc.GetRun(ctx, run.ID)
	if err != nil {
		t.Fatal(err)
	}
	return rd
}

func code(err error) string {
	if de, ok := domain.AsError(err); ok {
		return de.Code
	}
	return ""
}

func TestIngestAssetValidatesBytes(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	in := IngestAssetInput{ProjectID: f.project.ID, Role: domain.RoleReference, Capture: capture}

	in.Data = []byte("GIF89a not allowed")
	if _, err := f.svc.IngestAsset(ctx, in); code(err) != "INVALID_IMAGE" {
		t.Errorf("gif: %v", err)
	}
	good := pngBytes(t, 10, 10, color.White)
	in.Data = good[:len(good)-20]
	if _, err := f.svc.IngestAsset(ctx, in); code(err) != "INVALID_IMAGE" {
		t.Errorf("truncated png: %v", err)
	}
	in.Data = make([]byte, domain.MaxImageBytes+1)
	if _, err := f.svc.IngestAsset(ctx, in); code(err) != "IMAGE_TOO_LARGE" {
		t.Errorf("oversize: %v", err)
	}
	in.Data = good
	in.Role = "design"
	if _, err := f.svc.IngestAsset(ctx, in); code(err) != "INVALID_ROLE" {
		t.Errorf("role: %v", err)
	}
}

func TestCreateComparisonRejectsMismatchAndVersionsRequirements(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	other := capture
	other.ContentStateKey = "logged-in"
	act2, err := f.svc.IngestAsset(ctx, IngestAssetInput{ProjectID: f.project.ID, Role: domain.RoleActual,
		Capture: other, Data: pngBytes(t, 200, 100, color.Black)})
	if err != nil {
		t.Fatal(err)
	}
	_, err = f.svc.CreateComparison(ctx, CreateComparisonInput{ProjectID: f.project.ID,
		ReferenceAssetID: f.cmp.Reference.ID, ActualAssetID: act2.ID})
	if code(err) != "CAPTURE_MISMATCH" {
		t.Fatalf("mismatch: %v", err)
	}

	same, err := f.svc.CreateComparison(ctx, CreateComparisonInput{ProjectID: f.project.ID,
		ReferenceAssetID: f.cmp.Reference.ID, ActualAssetID: f.cmp.Actual.ID, Requirements: "Show the primary button"})
	if err != nil {
		t.Fatal(err)
	}
	if same.Requirement.ID != f.cmp.Requirement.ID {
		t.Error("unchanged requirements created a new version")
	}
	edited, err := f.svc.CreateComparison(ctx, CreateComparisonInput{ProjectID: f.project.ID,
		ReferenceAssetID: f.cmp.Reference.ID, ActualAssetID: f.cmp.Actual.ID, Requirements: "Show two buttons"})
	if err != nil {
		t.Fatal(err)
	}
	if edited.Requirement.Version != 2 {
		t.Errorf("edited requirement version = %d", edited.Requirement.Version)
	}
}

func TestRunStoresValidatedFindings(t *testing.T) {
	f := newFixture(t)
	f.provider.out = ports.AnalysisOutput{Complete: true, ProviderRequestID: "req-1", Findings: []domain.RawFinding{{
		Category:  "missing_element",
		Summary:   "Button is absent",
		Reference: &domain.RawEvidence{Box: &domain.NormalizedBox{X0: 100, Y0: 100, X1: 200, Y1: 300}},
	}}}
	rd := f.runOnce(t, StartRunInput{})
	if rd.Run.Status != domain.RunCompleted {
		t.Fatalf("status %s issues %v", rd.Run.Status, rd.Run.Issues)
	}
	if rd.Run.ProviderRequestID != "req-1" || rd.Run.StartedAt == nil || rd.Run.FinishedAt == nil {
		t.Errorf("provenance missing: %+v", rd.Run)
	}
	if len(rd.Findings) != 1 {
		t.Fatalf("findings = %d", len(rd.Findings))
	}
	box := rd.Findings[0].ReferenceEvidence.Box
	if *box != (domain.Box{X: 20, Y: 10, Width: 20, Height: 20}) {
		t.Errorf("box mapped to %+v", *box)
	}
	if rd.Findings[0].ReferenceEvidence.AssetID != f.cmp.Reference.ID {
		t.Error("evidence not bound to the reference asset")
	}
}

func TestRunWithRejectedFindingNeedsReview(t *testing.T) {
	f := newFixture(t)
	f.provider.out = ports.AnalysisOutput{Complete: true, Findings: []domain.RawFinding{
		{Category: "color_mismatch", Summary: "x"},
	}}
	rd := f.runOnce(t, StartRunInput{})
	if rd.Run.Status != domain.RunNeedsReview || len(rd.Findings) != 0 || len(rd.Run.Issues) == 0 {
		t.Fatalf("status %s, findings %d, issues %v", rd.Run.Status, len(rd.Findings), rd.Run.Issues)
	}
}

func TestProviderFailureNeverBecomesCleanPass(t *testing.T) {
	f := newFixture(t)
	f.provider.err = errors.New("upstream said: secret prompt text")
	rd := f.runOnce(t, StartRunInput{})
	if rd.Run.Status != domain.RunFailed || rd.Run.FailureCode != "PROVIDER_ERROR" {
		t.Fatalf("status %s code %s", rd.Run.Status, rd.Run.FailureCode)
	}
	if strings.Contains(rd.Run.FailureMessage, "secret") {
		t.Error("provider error text leaked into stored failure message")
	}
	if _, err := f.svc.ExportRun(context.Background(), rd.Run.ID); code(err) != "RUN_NOT_FINISHED" {
		t.Errorf("export of failed run: %v", err)
	}
}

func TestStartRunGuards(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()

	first, created, err := f.svc.StartRun(ctx, StartRunInput{ComparisonID: f.cmp.Comparison.ID, IdempotencyKey: "k1"})
	if err != nil || !created {
		t.Fatalf("first: %v %v", created, err)
	}
	again, created, err := f.svc.StartRun(ctx, StartRunInput{ComparisonID: f.cmp.Comparison.ID, IdempotencyKey: "k1"})
	if err != nil || created || again.ID != first.ID {
		t.Fatalf("idempotent replay: %v %v %v", again.ID, created, err)
	}
	if _, _, err := f.svc.StartRun(ctx, StartRunInput{ComparisonID: f.cmp.Comparison.ID}); code(err) != "RUN_ACTIVE" {
		t.Fatalf("second active run: %v", err)
	}

	f.svc.process(ctx, <-f.svc.queue)
	if _, _, err := f.svc.StartRun(ctx, StartRunInput{ComparisonID: f.cmp.Comparison.ID}); code(err) != "DUPLICATE_RUN" {
		t.Fatalf("duplicate fingerprint: %v", err)
	}
	if _, created, err := f.svc.StartRun(ctx, StartRunInput{ComparisonID: f.cmp.Comparison.ID, AllowDuplicate: true}); err != nil || !created {
		t.Fatalf("deliberate re-run: %v", err)
	}
}

func TestRecoverInterruptedFailsStaleRuns(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	run, _, err := f.svc.StartRun(ctx, StartRunInput{ComparisonID: f.cmp.Comparison.ID})
	if err != nil {
		t.Fatal(err)
	}
	if err := f.svc.RecoverInterrupted(ctx); err != nil {
		t.Fatal(err)
	}
	got, _ := f.repo.GetRun(ctx, run.ID)
	if got.Status != domain.RunFailed || got.FailureCode != "INTERRUPTED" {
		t.Fatalf("status %s code %s", got.Status, got.FailureCode)
	}
	// The worker must not resurrect it.
	f.svc.process(ctx, <-f.svc.queue)
	got, _ = f.repo.GetRun(ctx, run.ID)
	if got.Status != domain.RunFailed {
		t.Fatalf("status after stale process = %s", got.Status)
	}
}

func TestReviewAndExport(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	f.provider.out = ports.AnalysisOutput{Complete: true, Findings: []domain.RawFinding{{
		Category:  "layout_mismatch",
		Summary:   "Header | shifted <script>",
		Reference: &domain.RawEvidence{Box: &domain.NormalizedBox{X0: 0, Y0: 0, X1: 500, Y1: 500}},
		Actual:    &domain.RawEvidence{Box: &domain.NormalizedBox{X0: 10, Y0: 0, X1: 510, Y1: 500}},
	}}}
	rd := f.runOnce(t, StartRunInput{})
	fid := rd.Findings[0].ID

	if _, err := f.svc.ReviewFinding(ctx, fid, domain.DecisionRejected, ""); code(err) != "REASON_REQUIRED" {
		t.Fatalf("reasonless reject: %v", err)
	}
	if _, err := f.svc.ReviewFinding(ctx, fid, domain.DecisionRejected, "intended"); err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.ReviewFinding(ctx, fid, domain.DecisionAccepted, "re-checked"); err != nil {
		t.Fatal(err)
	}

	e, err := f.svc.ExportRun(ctx, rd.Run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if d := e.CurrentDecision(fid); d == nil || d.Decision != domain.DecisionAccepted {
		t.Fatalf("current decision = %+v", d)
	}
	if len(e.Reviews[fid]) != 2 {
		t.Error("review history not kept")
	}
	md := e.Markdown()
	for _, want := range []string{"Origin | **fixture**", "accepted — re-checked", "Header \\| shifted &lt;script&gt;", "not real model output"} {
		if !strings.Contains(md, want) {
			t.Errorf("markdown missing %q", want)
		}
	}
	if strings.Contains(md, "<script>") {
		t.Error("provider text injected raw HTML into markdown")
	}
}

func TestSafeProviderErrorIsStored(t *testing.T) {
	f := newFixture(t)
	f.provider.err = &ports.ProviderError{Code: "NO_FIXTURE", Message: "unknown pair"}
	rd := f.runOnce(t, StartRunInput{})
	if rd.Run.FailureCode != "NO_FIXTURE" || rd.Run.FailureMessage != "unknown pair" {
		t.Fatalf("failure = %s %q", rd.Run.FailureCode, rd.Run.FailureMessage)
	}
}
