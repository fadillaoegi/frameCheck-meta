// Package fixture replays labelled ground truth from demo/fixtures/manifest.json
// for known image pairs. Runs are labelled origin "fixture": they exercise the
// pipeline and UI with correct categories but are never model output.
package fixture

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"framecheck/internal/domain"
	"framecheck/internal/ports"
)

type box struct {
	X      int `json:"x"`
	Y      int `json:"y"`
	Width  int `json:"width"`
	Height int `json:"height"`
}

type expected struct {
	Category     string `json:"category"`
	Component    string `json:"component"`
	ReferenceBox *box   `json:"reference_box"`
	ActualBox    *box   `json:"actual_box"`
}

type image struct {
	SHA256 string `json:"sha256"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

type manifest struct {
	Origin    string `json:"origin"`
	Reference image  `json:"reference"`
	Cases     []struct {
		ID       string     `json:"id"`
		Actual   image      `json:"actual"`
		Expected []expected `json:"expected"`
	} `json:"cases"`
}

type pairKey struct{ ref, act string }

type entry struct {
	caseID   string
	expected []expected
	refW     int
	refH     int
	actW     int
	actH     int
}

type Provider struct {
	cases   map[pairKey]entry
	version string
}

// Load reads one or more fixture directories (each with a manifest.json)
// once at startup.
func Load(dirs ...string) (*Provider, error) {
	p := &Provider{cases: map[pairKey]entry{}}
	var all []byte
	for _, dir := range dirs {
		raw, err := os.ReadFile(filepath.Join(dir, "manifest.json"))
		if err != nil {
			return nil, fmt.Errorf("read fixture manifest: %w", err)
		}
		var m manifest
		if err := json.Unmarshal(raw, &m); err != nil {
			return nil, fmt.Errorf("parse fixture manifest: %w", err)
		}
		if m.Origin != string(domain.OriginFixture) {
			return nil, fmt.Errorf("fixture manifest origin is %q, want fixture", m.Origin)
		}
		all = append(all, raw...)
		for _, c := range m.Cases {
			p.cases[pairKey{m.Reference.SHA256, c.Actual.SHA256}] = entry{
				caseID: c.ID, expected: c.Expected,
				refW: m.Reference.Width, refH: m.Reference.Height,
				actW: c.Actual.Width, actH: c.Actual.Height,
			}
		}
	}
	p.version = "fixture-" + domain.SHA256Hex(all)[:12]
	return p, nil
}

func (*Provider) Origin() domain.RunOrigin { return domain.OriginFixture }
func (*Provider) ModelID() string          { return "fixture-ground-truth" }
func (p *Provider) ConfigVersion() string  { return p.version }
func (*Provider) PromptVersion() string    { return "none" }

func (p *Provider) Analyze(_ context.Context, in ports.AnalysisInput) (ports.AnalysisOutput, error) {
	e, ok := p.cases[pairKey{domain.SHA256Hex(in.Reference.Data), domain.SHA256Hex(in.Actual.Data)}]
	if !ok {
		return ports.AnalysisOutput{}, &ports.ProviderError{
			Code:    "NO_FIXTURE",
			Message: "the fixture provider only knows the image pairs in demo/fixtures; upload those or use the mock provider",
		}
	}
	out := ports.AnalysisOutput{Complete: true, ProviderRequestID: "fixture:" + e.caseID}
	for _, x := range e.expected {
		f := domain.RawFinding{
			Category:  x.Category,
			Summary:   fmt.Sprintf("Fixture ground truth: %s on %q", x.Category, x.Component),
			CheckStep: "Labelled defect from the fixture set; confirm it in both images.",
		}
		if x.ReferenceBox != nil {
			f.Reference = &domain.RawEvidence{ImageRole: "reference", Box: normalize(*x.ReferenceBox, e.refW, e.refH),
				Description: x.Component + " as designed"}
		}
		if x.ActualBox != nil {
			f.Actual = &domain.RawEvidence{ImageRole: "actual", Box: normalize(*x.ActualBox, e.actW, e.actH),
				Description: x.Component + " in the implementation"}
		}
		out.Findings = append(out.Findings, f)
	}
	return out, nil
}

func normalize(b box, w, h int) *domain.NormalizedBox {
	s := domain.NormalizedScale
	return &domain.NormalizedBox{
		X0: float64(b.X) * s / float64(w), Y0: float64(b.Y) * s / float64(h),
		X1: float64(b.X+b.Width) * s / float64(w), Y1: float64(b.Y+b.Height) * s / float64(h),
	}
}
