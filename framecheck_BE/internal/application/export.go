package application

import (
	"context"
	"fmt"
	"strings"
	"time"

	"framecheck/internal/domain"
)

// Export is the reviewed record of one run, with enough provenance to audit it.
type Export struct {
	GeneratedAt time.Time
	Project     domain.Project
	Comparison  ComparisonDetail
	Run         domain.Run
	Findings    []domain.Finding
	Reviews     map[string][]domain.Review
	Caveats     []string
}

// CurrentDecision returns the latest review of a finding, if any.
func (e Export) CurrentDecision(findingID string) *domain.Review {
	rs := e.Reviews[findingID]
	if len(rs) == 0 {
		return nil
	}
	return &rs[len(rs)-1]
}

func (s *Service) ExportRun(ctx context.Context, runID string) (Export, error) {
	rd, err := s.GetRun(ctx, runID)
	if err != nil {
		return Export{}, err
	}
	if !rd.Run.Status.Finished() {
		return Export{}, domain.Conflict("RUN_NOT_FINISHED", "only completed or needs_review runs can be exported")
	}
	cd, err := s.GetComparison(ctx, rd.Run.ComparisonID)
	if err != nil {
		return Export{}, err
	}
	cd.Runs = nil
	p, err := s.repo.GetProject(ctx, rd.Run.ProjectID)
	if err != nil {
		return Export{}, err
	}

	e := Export{
		GeneratedAt: s.clock.Now(), Project: p, Comparison: cd,
		Run: rd.Run, Findings: rd.Findings, Reviews: rd.Reviews,
	}
	if rd.Run.Origin != domain.OriginMetaAPI {
		e.Caveats = append(e.Caveats, fmt.Sprintf("Origin is %s: these findings are not real model output.", rd.Run.Origin))
	}
	if rd.Run.Status == domain.RunNeedsReview {
		e.Caveats = append(e.Caveats, "Run needs review: provider output was incomplete or some evidence failed validation.")
	}
	unreviewed := 0
	for _, f := range rd.Findings {
		if e.CurrentDecision(f.ID) == nil {
			unreviewed++
		}
	}
	if unreviewed > 0 {
		e.Caveats = append(e.Caveats, fmt.Sprintf("%d of %d findings have no human decision.", unreviewed, len(rd.Findings)))
	}
	if len(rd.Findings) == 0 && rd.Run.Status == domain.RunCompleted {
		e.Caveats = append(e.Caveats, "Zero findings means none were reported in the three MVP categories; it is not a release approval.")
	}
	return e, nil
}

// Markdown renders the export for humans. Provider text is escaped so it
// cannot inject headings, links, or HTML into the report.
func (e Export) Markdown() string {
	var b strings.Builder
	w := func(format string, args ...any) { fmt.Fprintf(&b, format, args...) }
	r := e.Run

	w("# FrameCheck report: %s\n\n", md(e.Project.Name))
	w("Generated %s. This report records findings and human decisions; it does not approve a release.\n\n", e.GeneratedAt.UTC().Format(time.RFC3339))

	w("## Run\n\n")
	w("| Field | Value |\n| --- | --- |\n")
	w("| Run ID | `%s` |\n", r.ID)
	w("| Status | %s |\n", r.Status)
	w("| Origin | **%s** |\n", r.Origin)
	w("| Model / config / prompt | %s / %s / %s |\n", md(r.ModelID), md(r.ConfigVersion), md(r.PromptVersion))
	if r.ProviderRequestID != "" {
		w("| Provider request ID | `%s` |\n", md(r.ProviderRequestID))
	}
	w("| Latency | %d ms |\n", r.LatencyMS)
	if r.InputTokens > 0 || r.OutputTokens > 0 {
		w("| Tokens (input / output) | %d / %d |\n", r.InputTokens, r.OutputTokens)
	}
	w("| Input fingerprint | `%s` |\n", r.InputFingerprint)
	w("| Reference image | `%s` (%d×%d px) |\n", e.Comparison.Reference.SHA256, e.Comparison.Reference.WidthPx, e.Comparison.Reference.HeightPx)
	w("| Actual image | `%s` (%d×%d px) |\n", e.Comparison.Actual.SHA256, e.Comparison.Actual.WidthPx, e.Comparison.Actual.HeightPx)
	c := e.Comparison.Reference.Capture
	w("| Capture | %d×%d CSS px @%gx, state `%s` |\n", c.ViewportWidthCSS, c.ViewportHeightCSS, c.DevicePixelRatio, c.ContentStateKey)
	w("| Requirements | version %d, sha256 `%s` |\n\n", e.Comparison.Requirement.Version, e.Comparison.Requirement.SHA256)

	if len(e.Caveats) > 0 || len(r.Issues) > 0 {
		w("## Caveats\n\n")
		for _, c := range e.Caveats {
			w("- %s\n", c)
		}
		for _, i := range r.Issues {
			w("- %s\n", md(i))
		}
		w("\n")
	}

	w("## Findings (%d)\n\n", len(e.Findings))
	if len(e.Findings) == 0 {
		w("No findings reported.\n")
	}
	for i, f := range e.Findings {
		w("### %d. %s — %s\n\n", i+1, f.Category, md(f.Summary))
		decision := "not reviewed"
		if d := e.CurrentDecision(f.ID); d != nil {
			decision = string(d.Decision)
			if d.Reason != "" {
				decision += " — " + md(d.Reason)
			}
		}
		w("- Decision: **%s**\n", decision)
		w("- Evidence status: %s\n", f.EvidenceStatus)
		if f.CheckStep != "" {
			w("- Check step: %s\n", md(f.CheckStep))
		}
		w("- Reference evidence: %s\n", evidenceMD(f.ReferenceEvidence))
		w("- Actual evidence: %s\n", evidenceMD(f.ActualEvidence))
		for _, issue := range f.Issues {
			w("- Validation issue: %s\n", md(issue))
		}
		w("\n")
	}
	return b.String()
}

func evidenceMD(ev *domain.Evidence) string {
	if ev == nil {
		return "none"
	}
	box := "no box"
	if ev.Box != nil {
		box = fmt.Sprintf("box x=%d y=%d w=%d h=%d px", ev.Box.X, ev.Box.Y, ev.Box.Width, ev.Box.Height)
	}
	if ev.Description == "" {
		return box
	}
	return box + " — " + md(ev.Description)
}

var mdEscaper = strings.NewReplacer(
	"\\", "\\\\", "`", "\\`", "*", "\\*", "_", "\\_", "[", "\\[", "]", "\\]",
	"<", "&lt;", ">", "&gt;", "|", "\\|", "#", "\\#", "\n", " ", "\r", " ",
)

func md(s string) string { return mdEscaper.Replace(s) }
