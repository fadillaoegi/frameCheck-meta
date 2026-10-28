package domain

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

const (
	maxSummaryLen     = 600
	maxDescriptionLen = 600
	maxCheckStepLen   = 600
)

// RawEvidence is a provider claim about one image. Nothing in it is trusted.
type RawEvidence struct {
	ImageRole   string
	Box         *NormalizedBox
	Description string
}

// RawFinding is one untrusted finding as returned by an inference provider.
type RawFinding struct {
	Category  string
	Summary   string
	CheckStep string
	Reference *RawEvidence
	Actual    *RawEvidence
}

// EvidenceContext binds evidence to the run's frozen assets and transforms.
type EvidenceContext struct {
	Reference          Asset
	Actual             Asset
	ReferenceTransform ImageTransform
	ActualTransform    ImageTransform
}

// BuildFinding validates a raw provider finding. It returns an error when the
// finding cannot be stored at all (unknown category, no explanation). Weak or
// invalid evidence is kept off the record and marks the finding needs_review.
func BuildFinding(raw RawFinding, ec EvidenceContext) (Finding, error) {
	category := Category(strings.TrimSpace(raw.Category))
	if !category.Valid() {
		return Finding{}, fmt.Errorf("unknown category %q", truncate(raw.Category, 40))
	}
	summary := strings.TrimSpace(raw.Summary)
	if summary == "" {
		return Finding{}, fmt.Errorf("%s finding has an empty summary", category)
	}

	f := Finding{
		Category:       category,
		Summary:        truncate(summary, maxSummaryLen),
		CheckStep:      truncate(strings.TrimSpace(raw.CheckStep), maxCheckStepLen),
		EvidenceStatus: EvidenceValidated,
	}

	ref, refIssue := buildEvidence(raw.Reference, RoleReference, ec.Reference, ec.ReferenceTransform)
	act, actIssue := buildEvidence(raw.Actual, RoleActual, ec.Actual, ec.ActualTransform)
	if refIssue != "" {
		f.Issues = append(f.Issues, refIssue)
	}
	if actIssue != "" {
		f.Issues = append(f.Issues, actIssue)
	}

	hasRef := ref != nil && ref.Box != nil
	hasAct := act != nil && act.Box != nil

	switch category {
	case CategoryMissingElement:
		// Absence is the claim: an actual-image box would be a fabricated
		// location, so it is never stored as evidence.
		if act != nil && act.Box != nil {
			f.Issues = append(f.Issues, "actual box dropped: a missing element has no actual evidence")
		}
		act = nil
		if !hasRef {
			f.Issues = append(f.Issues, "missing element needs a reference box")
		}
	case CategoryLayoutMismatch:
		if !hasRef || !hasAct {
			f.Issues = append(f.Issues, "layout mismatch needs both a reference and an actual box")
		}
	case CategoryClippedText:
		if !hasAct {
			f.Issues = append(f.Issues, "clipped text needs an actual box")
		}
		if !hasRef {
			f.Issues = append(f.Issues, "clipped text needs reference text or region evidence")
		}
	}

	f.ReferenceEvidence = ref
	f.ActualEvidence = act
	if len(f.Issues) > 0 {
		f.EvidenceStatus = EvidenceNeedsReview
	}
	return f, nil
}

func buildEvidence(raw *RawEvidence, role ImageRole, asset Asset, t ImageTransform) (*Evidence, string) {
	if raw == nil {
		return nil, ""
	}
	if raw.ImageRole != "" && ImageRole(raw.ImageRole) != role {
		return nil, fmt.Sprintf("%s evidence claimed image role %q", role, truncate(raw.ImageRole, 20))
	}
	ev := &Evidence{
		AssetID:     asset.ID,
		ImageRole:   role,
		Description: truncate(strings.TrimSpace(raw.Description), maxDescriptionLen),
	}
	if raw.Box == nil {
		return ev, ""
	}
	box, err := t.ToOriginal(*raw.Box)
	if err != nil {
		return ev, fmt.Sprintf("%s box rejected: %v", role, err)
	}
	ev.Box = &box
	return ev, ""
}

// ResolveRunStatus decides the terminal status of an analysed run. A run is
// completed only when the provider reported a complete answer and every
// finding has validated evidence; anything else needs a human.
func ResolveRunStatus(providerComplete bool, rejected int, findings []Finding) RunStatus {
	if !providerComplete || rejected > 0 {
		return RunNeedsReview
	}
	for _, f := range findings {
		if f.EvidenceStatus != EvidenceValidated {
			return RunNeedsReview
		}
	}
	return RunCompleted
}

func truncate(s string, max int) string {
	if utf8.RuneCountInString(s) <= max {
		return s
	}
	r := []rune(s)
	return string(r[:max])
}
