// Package ports declares the interfaces application use cases depend on.
// Adapters implement them; the application never imports an adapter.
package ports

import (
	"context"
	"time"

	"framecheck/internal/domain"
)

type Clock interface {
	Now() time.Time
}

type IDGenerator interface {
	NewID() string
}

// Repository persists FrameCheck records. Methods return domain errors with
// KindNotFound when a record is absent.
type Repository interface {
	CreateProject(ctx context.Context, p domain.Project) error
	ListProjects(ctx context.Context) ([]domain.Project, error)
	GetProject(ctx context.Context, id string) (domain.Project, error)

	CreateAsset(ctx context.Context, a domain.Asset) error
	GetAsset(ctx context.Context, id string) (domain.Asset, error)
	ListAssets(ctx context.Context, projectID string) ([]domain.Asset, error)

	// LatestRequirement returns nil when the project has no requirement yet.
	LatestRequirement(ctx context.Context, projectID string) (*domain.RequirementVersion, error)
	CreateRequirement(ctx context.Context, r domain.RequirementVersion) error
	GetRequirement(ctx context.Context, id string) (domain.RequirementVersion, error)

	CreateComparison(ctx context.Context, c domain.Comparison) error
	GetComparison(ctx context.Context, id string) (domain.Comparison, error)
	ListComparisons(ctx context.Context, projectID string) ([]domain.Comparison, error)

	// CreateRun inserts a queued run. It returns a domain conflict with code
	// RUN_ACTIVE when the project already has a queued or running run, and
	// IDEMPOTENCY_CONFLICT when the key is already used on the comparison.
	CreateRun(ctx context.Context, r domain.Run) error
	GetRun(ctx context.Context, id string) (domain.Run, error)
	ListRuns(ctx context.Context, comparisonID string) ([]domain.Run, error)
	FindRunByIdempotencyKey(ctx context.Context, comparisonID, key string) (*domain.Run, error)
	// FindReusableRun returns the newest finished run with this fingerprint.
	FindReusableRun(ctx context.Context, fingerprint string) (*domain.Run, error)
	MarkRunRunning(ctx context.Context, id string, at time.Time) error
	// FinishRun stores the terminal run fields and its findings atomically.
	FinishRun(ctx context.Context, r domain.Run, findings []domain.Finding) error
	// FailInterruptedRuns fails runs left queued or running by a previous process.
	FailInterruptedRuns(ctx context.Context, at time.Time) (int, error)

	GetFinding(ctx context.Context, id string) (domain.Finding, error)
	ListFindings(ctx context.Context, runID string) ([]domain.Finding, error)
	CreateReview(ctx context.Context, r domain.Review) error
	// ListReviewsForRun returns every review of the run's findings, oldest first.
	ListReviewsForRun(ctx context.Context, runID string) ([]domain.Review, error)
}

// AssetStore keeps image bytes outside the database.
type AssetStore interface {
	Put(ctx context.Context, key string, data []byte) error
	Get(ctx context.Context, key string) ([]byte, error)
}

type AnalysisImage struct {
	AssetID   string
	Role      domain.ImageRole
	MIME      string
	Data      []byte
	Transform domain.ImageTransform
}

type AnalysisInput struct {
	Reference    AnalysisImage
	Actual       AnalysisImage
	Requirements string
}

type AnalysisOutput struct {
	Findings []domain.RawFinding
	// Complete is the provider's own claim that it inspected both images
	// fully. FrameCheck still validates every finding.
	Complete          bool
	ProviderRequestID string
	InputTokens       int64
	OutputTokens      int64
}

// ProviderError is a provider failure whose message is safe to store and show.
// Any other error from Analyze is logged but replaced by a generic message.
type ProviderError struct {
	Code    string
	Message string
}

func (e *ProviderError) Error() string { return e.Code + ": " + e.Message }

// ImageSizeLimiter is implemented by providers that want images downscaled
// so the longer side is at most MaxAnalysisSide pixels. The provider must then
// send images at the AnalysisWidth × AnalysisHeight of each input transform.
type ImageSizeLimiter interface {
	MaxAnalysisSide() int
}

// InferenceProvider runs visual analysis. Its output is untrusted.
type InferenceProvider interface {
	Origin() domain.RunOrigin
	ModelID() string
	ConfigVersion() string
	PromptVersion() string
	Analyze(ctx context.Context, in AnalysisInput) (AnalysisOutput, error)
}
