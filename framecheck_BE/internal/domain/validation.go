package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
	"regexp"
	"strings"
	"unicode/utf8"
)

const (
	MaxImageBytes        = 10 << 20
	MaxImageSide         = 8000
	MaxProjectNameLen    = 120
	MaxRequirementsLen   = 4000
	MaxReviewReasonLen   = 1000
	MaxIdempotencyKeyLen = 128
	maxViewportCSS       = 10000
)

var contentStateKeyPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{0,63}$`)

func NormalizeProjectName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" || utf8.RuneCountInString(name) > MaxProjectNameLen {
		return "", Invalid("INVALID_NAME", "project name must be 1-%d characters", MaxProjectNameLen)
	}
	return name, nil
}

func (c Capture) Validate() error {
	if c.ViewportWidthCSS < 1 || c.ViewportWidthCSS > maxViewportCSS ||
		c.ViewportHeightCSS < 1 || c.ViewportHeightCSS > maxViewportCSS {
		return Invalid("INVALID_CAPTURE", "viewport CSS width and height must be 1-%d", maxViewportCSS)
	}
	if math.IsNaN(c.DevicePixelRatio) || c.DevicePixelRatio < 0.5 || c.DevicePixelRatio > 5 {
		return Invalid("INVALID_CAPTURE", "device pixel ratio must be between 0.5 and 5")
	}
	if !contentStateKeyPattern.MatchString(c.ContentStateKey) {
		return Invalid("INVALID_CAPTURE", "content state key must be 1-64 lowercase letters, digits, '.', '_' or '-'")
	}
	return nil
}

// Matches reports whether two captures were taken under the same conditions.
func (c Capture) Matches(o Capture) bool {
	return c.ViewportWidthCSS == o.ViewportWidthCSS &&
		c.ViewportHeightCSS == o.ViewportHeightCSS &&
		math.Abs(c.DevicePixelRatio-o.DevicePixelRatio) < 1e-9 &&
		c.ContentStateKey == o.ContentStateKey
}

// ValidatePair checks that two assets can form a comparison.
func ValidatePair(projectID string, ref, act Asset) error {
	if ref.ProjectID != projectID || act.ProjectID != projectID {
		return Conflict("WRONG_PROJECT", "both assets must belong to this project")
	}
	if ref.Role != RoleReference || act.Role != RoleActual {
		return Conflict("WRONG_ROLE", "reference_asset_id must be a reference image and actual_asset_id an actual image")
	}
	if !ref.Capture.Matches(act.Capture) {
		return Conflict("CAPTURE_MISMATCH", "viewport CSS size, device pixel ratio, and content state must match")
	}
	return nil
}

func NormalizeRequirements(text string) (string, error) {
	text = strings.TrimSpace(strings.ReplaceAll(text, "\r\n", "\n"))
	if utf8.RuneCountInString(text) > MaxRequirementsLen {
		return "", Invalid("INVALID_REQUIREMENTS", "requirements must be at most %d characters", MaxRequirementsLen)
	}
	return text, nil
}

func ValidateReview(decision Decision, reason string) (string, error) {
	if !decision.Valid() {
		return "", Invalid("INVALID_DECISION", "decision must be accepted or rejected")
	}
	reason = strings.TrimSpace(reason)
	if utf8.RuneCountInString(reason) > MaxReviewReasonLen {
		return "", Invalid("INVALID_REASON", "reason must be at most %d characters", MaxReviewReasonLen)
	}
	if decision == DecisionRejected && reason == "" {
		return "", Invalid("REASON_REQUIRED", "a rejection needs a reason")
	}
	return reason, nil
}

func ValidateIdempotencyKey(key string) error {
	if len(key) > MaxIdempotencyKeyLen {
		return Invalid("INVALID_IDEMPOTENCY_KEY", "Idempotency-Key must be at most %d bytes", MaxIdempotencyKeyLen)
	}
	return nil
}

func SHA256Hex(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// FingerprintInput lists everything that determines a run's result. Two runs
// with the same fingerprint analysed identical inputs with identical settings.
type FingerprintInput struct {
	ReferenceSHA256   string
	ActualSHA256      string
	RequirementSHA256 string
	Origin            RunOrigin
	ModelID           string
	ConfigVersion     string
	PromptVersion     string
	Transform         TransformSpec
}

func (f FingerprintInput) Fingerprint() string {
	t := f.Transform
	parts := []string{
		"v1",
		f.ReferenceSHA256, f.ActualSHA256, f.RequirementSHA256,
		string(f.Origin), f.ModelID, f.ConfigVersion, f.PromptVersion,
		t.Coordinates, t.Rounding, transformKey(t.Reference), transformKey(t.Actual),
	}
	return SHA256Hex([]byte(strings.Join(parts, "\x1f")))
}

func transformKey(t ImageTransform) string {
	return fmt.Sprintf("%d,%d,%d,%d,%d,%d,%g,%g",
		t.OriginalWidth, t.OriginalHeight, t.AnalysisWidth, t.AnalysisHeight,
		t.CropX, t.CropY, t.ScaleX, t.ScaleY)
}
