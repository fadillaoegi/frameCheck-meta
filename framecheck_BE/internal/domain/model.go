// Package domain holds FrameCheck entities and pure validation rules.
// It must not import HTTP, SQL, storage, or provider SDK packages.
package domain

import "time"

type ImageRole string

const (
	RoleReference ImageRole = "reference"
	RoleActual    ImageRole = "actual"
)

func (r ImageRole) Valid() bool { return r == RoleReference || r == RoleActual }

type Category string

const (
	CategoryMissingElement Category = "missing_element"
	CategoryClippedText    Category = "clipped_text"
	CategoryLayoutMismatch Category = "layout_mismatch"
)

func (c Category) Valid() bool {
	switch c {
	case CategoryMissingElement, CategoryClippedText, CategoryLayoutMismatch:
		return true
	}
	return false
}

type RunStatus string

const (
	RunQueued      RunStatus = "queued"
	RunRunning     RunStatus = "running"
	RunCompleted   RunStatus = "completed"
	RunNeedsReview RunStatus = "needs_review"
	RunFailed      RunStatus = "failed"
)

// Finished reports whether a run has reached a terminal state that can be
// reviewed and exported.
func (s RunStatus) Finished() bool { return s == RunCompleted || s == RunNeedsReview }

type RunOrigin string

const (
	OriginMetaAPI RunOrigin = "meta_api"
	OriginFixture RunOrigin = "fixture"
	OriginMock    RunOrigin = "mock"
)

type EvidenceStatus string

const (
	EvidenceValidated   EvidenceStatus = "validated"
	EvidenceNeedsReview EvidenceStatus = "needs_review"
)

type Decision string

const (
	DecisionAccepted Decision = "accepted"
	DecisionRejected Decision = "rejected"
)

func (d Decision) Valid() bool { return d == DecisionAccepted || d == DecisionRejected }

type Project struct {
	ID        string
	Name      string
	CreatedAt time.Time
}

// Capture describes the conditions a screenshot was taken under. Two captures
// are comparable only when all fields agree.
type Capture struct {
	ViewportWidthCSS  int
	ViewportHeightCSS int
	DevicePixelRatio  float64
	ContentStateKey   string
}

type Asset struct {
	ID         string
	ProjectID  string
	Role       ImageRole
	SHA256     string
	MIME       string
	Bytes      int64
	WidthPx    int
	HeightPx   int
	Capture    Capture
	StorageKey string
	CreatedAt  time.Time
}

type RequirementVersion struct {
	ID        string
	ProjectID string
	Version   int
	Text      string
	SHA256    string
	CreatedAt time.Time
}

type Comparison struct {
	ID                   string
	ProjectID            string
	ReferenceAssetID     string
	ActualAssetID        string
	RequirementVersionID string
	CreatedAt            time.Time
}

type Run struct {
	ID                string
	ProjectID         string
	ComparisonID      string
	Status            RunStatus
	Origin            RunOrigin
	InputFingerprint  string
	IdempotencyKey    string
	ModelID           string
	ConfigVersion     string
	PromptVersion     string
	Transform         TransformSpec
	ProviderRequestID string
	LatencyMS         int64
	InputTokens       int64
	OutputTokens      int64
	FailureCode       string
	FailureMessage    string
	Issues            []string
	CreatedAt         time.Time
	StartedAt         *time.Time
	FinishedAt        *time.Time
}

// Box is a rectangle in original raster pixel coordinates, origin top left.
type Box struct {
	X      int
	Y      int
	Width  int
	Height int
}

type Evidence struct {
	AssetID     string
	ImageRole   ImageRole
	Box         *Box
	Description string
}

type Finding struct {
	ID                string
	RunID             string
	Category          Category
	Summary           string
	CheckStep         string
	ReferenceEvidence *Evidence
	ActualEvidence    *Evidence
	EvidenceStatus    EvidenceStatus
	Issues            []string
	CreatedAt         time.Time
}

type Review struct {
	ID         string
	FindingID  string
	Decision   Decision
	Reason     string
	ReviewedAt time.Time
}
