package fixture

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"framecheck/internal/domain"
	"framecheck/internal/ports"
)

const fixturesDir = "../../../../demo/fixtures"

func load(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(fixturesDir, name))
	if err != nil {
		t.Skipf("fixtures not generated: %v", err)
	}
	return b
}

func TestReplaysGroundTruthWithinBounds(t *testing.T) {
	p, err := Load(fixturesDir)
	if err != nil {
		t.Skipf("fixtures not generated: %v", err)
	}
	ref := load(t, "reference.png")
	act := load(t, "actual-missing-hero-cta.png")
	out, err := p.Analyze(context.Background(), ports.AnalysisInput{
		Reference: ports.AnalysisImage{Data: ref}, Actual: ports.AnalysisImage{Data: act}})
	if err != nil {
		t.Fatal(err)
	}
	if !out.Complete || len(out.Findings) != 1 || out.Findings[0].Category != "missing_element" {
		t.Fatalf("out = %+v", out)
	}
	// The replayed box must survive the same validation as model output.
	f, err := domain.BuildFinding(out.Findings[0], domain.EvidenceContext{
		ReferenceTransform: domain.IdentityTransform(1280, 800),
		ActualTransform:    domain.IdentityTransform(1280, 800),
	})
	if err != nil || f.EvidenceStatus != domain.EvidenceValidated {
		t.Fatalf("finding %+v err %v", f, err)
	}
	if b := *f.ReferenceEvidence.Box; b != (domain.Box{X: 64, Y: 376, Width: 196, Height: 52}) {
		t.Errorf("box round-trip = %+v", b)
	}
}

func TestUnknownPairIsSafeError(t *testing.T) {
	p, err := Load(fixturesDir)
	if err != nil {
		t.Skipf("fixtures not generated: %v", err)
	}
	_, err = p.Analyze(context.Background(), ports.AnalysisInput{
		Reference: ports.AnalysisImage{Data: []byte("a")}, Actual: ports.AnalysisImage{Data: []byte("b")}})
	var pe *ports.ProviderError
	if !errors.As(err, &pe) || pe.Code != "NO_FIXTURE" {
		t.Fatalf("err = %v", err)
	}
}
