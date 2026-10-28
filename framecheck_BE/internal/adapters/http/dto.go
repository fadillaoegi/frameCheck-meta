package http

import (
	"time"

	"framecheck/internal/application"
	"framecheck/internal/domain"
)

// DTOs keep the wire format independent of domain structs.

type projectDTO struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

type assetDTO struct {
	ID                string    `json:"id"`
	ProjectID         string    `json:"project_id"`
	Role              string    `json:"role"`
	SHA256            string    `json:"sha256"`
	MIME              string    `json:"mime"`
	Bytes             int64     `json:"bytes"`
	WidthPx           int       `json:"width_px"`
	HeightPx          int       `json:"height_px"`
	ViewportWidthCSS  int       `json:"viewport_width_css"`
	ViewportHeightCSS int       `json:"viewport_height_css"`
	DevicePixelRatio  float64   `json:"device_pixel_ratio"`
	ContentStateKey   string    `json:"content_state_key"`
	ContentURL        string    `json:"content_url"`
	CreatedAt         time.Time `json:"created_at"`
}

type requirementDTO struct {
	ID        string    `json:"id"`
	Version   int       `json:"version"`
	Text      string    `json:"text"`
	SHA256    string    `json:"sha256"`
	CreatedAt time.Time `json:"created_at"`
}

type comparisonDTO struct {
	ID                   string    `json:"id"`
	ProjectID            string    `json:"project_id"`
	ReferenceAssetID     string    `json:"reference_asset_id"`
	ActualAssetID        string    `json:"actual_asset_id"`
	RequirementVersionID string    `json:"requirement_version_id"`
	CreatedAt            time.Time `json:"created_at"`
}

type comparisonDetailDTO struct {
	comparisonDTO
	Reference   assetDTO       `json:"reference"`
	Actual      assetDTO       `json:"actual"`
	Requirement requirementDTO `json:"requirement"`
	Runs        []runDTO       `json:"runs"`
}

type projectDetailDTO struct {
	projectDTO
	Assets      []assetDTO      `json:"assets"`
	Comparisons []comparisonDTO `json:"comparisons"`
}

type runDTO struct {
	ID                string               `json:"id"`
	ProjectID         string               `json:"project_id"`
	ComparisonID      string               `json:"comparison_id"`
	Status            string               `json:"status"`
	Origin            string               `json:"origin"`
	InputFingerprint  string               `json:"input_fingerprint"`
	ModelID           string               `json:"model_id"`
	ConfigVersion     string               `json:"config_version"`
	PromptVersion     string               `json:"prompt_version"`
	Transform         domain.TransformSpec `json:"transform"`
	ProviderRequestID string               `json:"provider_request_id,omitempty"`
	LatencyMS         int64                `json:"latency_ms"`
	InputTokens       int64                `json:"input_tokens"`
	OutputTokens      int64                `json:"output_tokens"`
	FailureCode       string               `json:"failure_code,omitempty"`
	FailureMessage    string               `json:"failure_message,omitempty"`
	Issues            []string             `json:"issues"`
	CreatedAt         time.Time            `json:"created_at"`
	StartedAt         *time.Time           `json:"started_at"`
	FinishedAt        *time.Time           `json:"finished_at"`
}

type boxDTO struct {
	X      int `json:"x"`
	Y      int `json:"y"`
	Width  int `json:"width"`
	Height int `json:"height"`
}

type evidenceDTO struct {
	AssetID     string  `json:"asset_id"`
	ImageRole   string  `json:"image_role"`
	Box         *boxDTO `json:"box_px"`
	Description string  `json:"description"`
}

type reviewDTO struct {
	ID         string    `json:"id"`
	FindingID  string    `json:"finding_id"`
	Decision   string    `json:"decision"`
	Reason     string    `json:"reason"`
	ReviewedAt time.Time `json:"reviewed_at"`
}

type findingDTO struct {
	ID                string       `json:"id"`
	RunID             string       `json:"run_id"`
	Category          string       `json:"category"`
	Summary           string       `json:"summary"`
	CheckStep         string       `json:"check_step"`
	ReferenceEvidence *evidenceDTO `json:"reference_evidence"`
	ActualEvidence    *evidenceDTO `json:"actual_evidence"`
	EvidenceStatus    string       `json:"evidence_status"`
	Issues            []string     `json:"issues"`
	CurrentReview     *reviewDTO   `json:"current_review"`
	ReviewHistory     []reviewDTO  `json:"review_history"`
}

type runDetailDTO struct {
	runDTO
	Findings []findingDTO `json:"findings"`
}

type exportDTO struct {
	GeneratedAt time.Time           `json:"generated_at"`
	Project     projectDTO          `json:"project"`
	Comparison  comparisonDetailDTO `json:"comparison"`
	Run         runDTO              `json:"run"`
	Findings    []findingDTO        `json:"findings"`
	Caveats     []string            `json:"caveats"`
}

func toProject(p domain.Project) projectDTO {
	return projectDTO{ID: p.ID, Name: p.Name, CreatedAt: p.CreatedAt}
}

func toAsset(a domain.Asset) assetDTO {
	return assetDTO{
		ID: a.ID, ProjectID: a.ProjectID, Role: string(a.Role), SHA256: a.SHA256, MIME: a.MIME,
		Bytes: a.Bytes, WidthPx: a.WidthPx, HeightPx: a.HeightPx,
		ViewportWidthCSS: a.Capture.ViewportWidthCSS, ViewportHeightCSS: a.Capture.ViewportHeightCSS,
		DevicePixelRatio: a.Capture.DevicePixelRatio, ContentStateKey: a.Capture.ContentStateKey,
		ContentURL: "/v1/assets/" + a.ID + "/content", CreatedAt: a.CreatedAt,
	}
}

func toComparison(c domain.Comparison) comparisonDTO {
	return comparisonDTO{
		ID: c.ID, ProjectID: c.ProjectID, ReferenceAssetID: c.ReferenceAssetID,
		ActualAssetID: c.ActualAssetID, RequirementVersionID: c.RequirementVersionID, CreatedAt: c.CreatedAt,
	}
}

func toComparisonDetail(cd application.ComparisonDetail) comparisonDetailDTO {
	runs := make([]runDTO, 0, len(cd.Runs))
	for _, r := range cd.Runs {
		runs = append(runs, toRun(r))
	}
	q := cd.Requirement
	return comparisonDetailDTO{
		comparisonDTO: toComparison(cd.Comparison),
		Reference:     toAsset(cd.Reference),
		Actual:        toAsset(cd.Actual),
		Requirement:   requirementDTO{ID: q.ID, Version: q.Version, Text: q.Text, SHA256: q.SHA256, CreatedAt: q.CreatedAt},
		Runs:          runs,
	}
}

func toRun(r domain.Run) runDTO {
	issues := r.Issues
	if issues == nil {
		issues = []string{}
	}
	return runDTO{
		ID: r.ID, ProjectID: r.ProjectID, ComparisonID: r.ComparisonID, Status: string(r.Status),
		Origin: string(r.Origin), InputFingerprint: r.InputFingerprint, ModelID: r.ModelID,
		ConfigVersion: r.ConfigVersion, PromptVersion: r.PromptVersion, Transform: r.Transform,
		ProviderRequestID: r.ProviderRequestID, LatencyMS: r.LatencyMS,
		InputTokens: r.InputTokens, OutputTokens: r.OutputTokens,
		FailureCode: r.FailureCode, FailureMessage: r.FailureMessage, Issues: issues,
		CreatedAt: r.CreatedAt, StartedAt: r.StartedAt, FinishedAt: r.FinishedAt,
	}
}

func toEvidence(ev *domain.Evidence) *evidenceDTO {
	if ev == nil {
		return nil
	}
	out := &evidenceDTO{AssetID: ev.AssetID, ImageRole: string(ev.ImageRole), Description: ev.Description}
	if ev.Box != nil {
		out.Box = &boxDTO{X: ev.Box.X, Y: ev.Box.Y, Width: ev.Box.Width, Height: ev.Box.Height}
	}
	return out
}

func toReview(r domain.Review) reviewDTO {
	return reviewDTO{ID: r.ID, FindingID: r.FindingID, Decision: string(r.Decision), Reason: r.Reason, ReviewedAt: r.ReviewedAt}
}

func toFindings(findings []domain.Finding, reviews map[string][]domain.Review) []findingDTO {
	out := make([]findingDTO, 0, len(findings))
	for _, f := range findings {
		issues := f.Issues
		if issues == nil {
			issues = []string{}
		}
		history := make([]reviewDTO, 0, len(reviews[f.ID]))
		for _, r := range reviews[f.ID] {
			history = append(history, toReview(r))
		}
		var current *reviewDTO
		if len(history) > 0 {
			current = &history[len(history)-1]
		}
		out = append(out, findingDTO{
			ID: f.ID, RunID: f.RunID, Category: string(f.Category), Summary: f.Summary, CheckStep: f.CheckStep,
			ReferenceEvidence: toEvidence(f.ReferenceEvidence), ActualEvidence: toEvidence(f.ActualEvidence),
			EvidenceStatus: string(f.EvidenceStatus), Issues: issues,
			CurrentReview: current, ReviewHistory: history,
		})
	}
	return out
}

func toExport(e application.Export) exportDTO {
	caveats := e.Caveats
	if caveats == nil {
		caveats = []string{}
	}
	return exportDTO{
		GeneratedAt: e.GeneratedAt, Project: toProject(e.Project),
		Comparison: toComparisonDetail(e.Comparison), Run: toRun(e.Run),
		Findings: toFindings(e.Findings, e.Reviews), Caveats: caveats,
	}
}
