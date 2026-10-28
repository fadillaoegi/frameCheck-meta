package domain

import (
	"math"
	"testing"
)

func TestToOriginalIdentity(t *testing.T) {
	tr := IdentityTransform(1280, 800)
	box, err := tr.ToOriginal(NormalizedBox{X0: 0, Y0: 0, X1: 1000, Y1: 1000})
	if err != nil {
		t.Fatal(err)
	}
	if box != (Box{X: 0, Y: 0, Width: 1280, Height: 800}) {
		t.Fatalf("full-image box = %+v", box)
	}

	box, err = tr.ToOriginal(NormalizedBox{X0: 100.4, Y0: 250, X1: 200.2, Y1: 500.1})
	if err != nil {
		t.Fatal(err)
	}
	// floor(128.512)=128, ceil(256.256)=257, floor(200)=200, ceil(400.08)=401
	if box != (Box{X: 128, Y: 200, Width: 129, Height: 201}) {
		t.Fatalf("box = %+v", box)
	}
}

func TestToOriginalCropAndScale(t *testing.T) {
	// Analysis image is a 500x400 crop at (100,50), downscaled by 0.5 to 250x200.
	tr := ImageTransform{
		OriginalWidth: 2000, OriginalHeight: 1000,
		AnalysisWidth: 250, AnalysisHeight: 200,
		CropX: 100, CropY: 50, ScaleX: 0.5, ScaleY: 0.5,
	}
	box, err := tr.ToOriginal(NormalizedBox{X0: 0, Y0: 0, X1: 1000, Y1: 1000})
	if err != nil {
		t.Fatal(err)
	}
	if box != (Box{X: 100, Y: 50, Width: 500, Height: 400}) {
		t.Fatalf("box = %+v", box)
	}
}

func TestToOriginalRejectsBadBoxes(t *testing.T) {
	tr := IdentityTransform(100, 100)
	bad := []NormalizedBox{
		{X0: math.NaN(), Y0: 0, X1: 10, Y1: 10},
		{X0: 0, Y0: 0, X1: math.Inf(1), Y1: 10},
		{X0: -1, Y0: 0, X1: 10, Y1: 10},
		{X0: 0, Y0: 0, X1: 1000.5, Y1: 10},
		{X0: 50, Y0: 0, X1: 50, Y1: 10},
		{X0: 60, Y0: 0, X1: 50, Y1: 10},
	}
	for _, nb := range bad {
		if _, err := tr.ToOriginal(nb); err == nil {
			t.Errorf("expected rejection for %+v", nb)
		}
	}
	// A crop transform whose mapped box leaves the original raster is rejected.
	crop := ImageTransform{OriginalWidth: 100, OriginalHeight: 100, AnalysisWidth: 100, AnalysisHeight: 100, CropX: 50, ScaleX: 1, ScaleY: 1}
	if _, err := crop.ToOriginal(NormalizedBox{X0: 0, Y0: 0, X1: 1000, Y1: 1000}); err == nil {
		t.Error("expected out-of-bounds rejection")
	}
}

func evidenceContext() EvidenceContext {
	return EvidenceContext{
		Reference:          Asset{ID: "ref", WidthPx: 1000, HeightPx: 1000},
		Actual:             Asset{ID: "act", WidthPx: 1000, HeightPx: 1000},
		ReferenceTransform: IdentityTransform(1000, 1000),
		ActualTransform:    IdentityTransform(1000, 1000),
	}
}

func nbox(x0, y0, x1, y1 float64) *NormalizedBox {
	return &NormalizedBox{X0: x0, Y0: y0, X1: x1, Y1: y1}
}

func TestBuildFindingRejectsUnknownCategoryAndEmptySummary(t *testing.T) {
	if _, err := BuildFinding(RawFinding{Category: "color_mismatch", Summary: "x"}, evidenceContext()); err == nil {
		t.Error("unknown category accepted")
	}
	if _, err := BuildFinding(RawFinding{Category: "clipped_text", Summary: "  "}, evidenceContext()); err == nil {
		t.Error("empty summary accepted")
	}
}

func TestBuildFindingMissingElementDropsActualBox(t *testing.T) {
	f, err := BuildFinding(RawFinding{
		Category:  "missing_element",
		Summary:   "Primary button is absent",
		Reference: &RawEvidence{ImageRole: "reference", Box: nbox(10, 10, 20, 20)},
		Actual:    &RawEvidence{ImageRole: "actual", Box: nbox(10, 10, 20, 20)},
	}, evidenceContext())
	if err != nil {
		t.Fatal(err)
	}
	if f.ActualEvidence != nil {
		t.Error("missing element kept actual evidence")
	}
	if f.ReferenceEvidence == nil || f.ReferenceEvidence.Box == nil || f.ReferenceEvidence.AssetID != "ref" {
		t.Fatalf("reference evidence = %+v", f.ReferenceEvidence)
	}
	if f.EvidenceStatus != EvidenceNeedsReview {
		t.Error("dropping a provider claim should be surfaced for review")
	}

	f, _ = BuildFinding(RawFinding{
		Category:  "missing_element",
		Summary:   "Primary button is absent",
		Reference: &RawEvidence{Box: nbox(10, 10, 20, 20)},
	}, evidenceContext())
	if f.EvidenceStatus != EvidenceValidated {
		t.Errorf("clean missing element status = %s, issues %v", f.EvidenceStatus, f.Issues)
	}
}

func TestBuildFindingEvidenceRequirements(t *testing.T) {
	cases := []struct {
		name string
		raw  RawFinding
		want EvidenceStatus
	}{
		{"layout both", RawFinding{Category: "layout_mismatch", Summary: "s",
			Reference: &RawEvidence{Box: nbox(1, 1, 2, 2)}, Actual: &RawEvidence{Box: nbox(1, 1, 3, 3)}}, EvidenceValidated},
		{"layout one", RawFinding{Category: "layout_mismatch", Summary: "s",
			Reference: &RawEvidence{Box: nbox(1, 1, 2, 2)}}, EvidenceNeedsReview},
		{"clipped no ref", RawFinding{Category: "clipped_text", Summary: "s",
			Actual: &RawEvidence{Box: nbox(1, 1, 2, 2)}}, EvidenceNeedsReview},
		{"bad box", RawFinding{Category: "layout_mismatch", Summary: "s",
			Reference: &RawEvidence{Box: nbox(1, 1, 2, 2)}, Actual: &RawEvidence{Box: nbox(1, 1, 3000, 3)}}, EvidenceNeedsReview},
		{"wrong role", RawFinding{Category: "layout_mismatch", Summary: "s",
			Reference: &RawEvidence{ImageRole: "actual", Box: nbox(1, 1, 2, 2)}, Actual: &RawEvidence{Box: nbox(1, 1, 3, 3)}}, EvidenceNeedsReview},
	}
	for _, c := range cases {
		f, err := BuildFinding(c.raw, evidenceContext())
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		if f.EvidenceStatus != c.want {
			t.Errorf("%s: status %s, issues %v", c.name, f.EvidenceStatus, f.Issues)
		}
	}
}

func TestResolveRunStatus(t *testing.T) {
	ok := Finding{EvidenceStatus: EvidenceValidated}
	weak := Finding{EvidenceStatus: EvidenceNeedsReview}
	if s := ResolveRunStatus(true, 0, nil); s != RunCompleted {
		t.Errorf("complete empty = %s", s)
	}
	if s := ResolveRunStatus(false, 0, nil); s != RunNeedsReview {
		t.Errorf("incomplete empty = %s", s)
	}
	if s := ResolveRunStatus(true, 1, []Finding{ok}); s != RunNeedsReview {
		t.Errorf("rejected = %s", s)
	}
	if s := ResolveRunStatus(true, 0, []Finding{ok, weak}); s != RunNeedsReview {
		t.Errorf("weak = %s", s)
	}
}

func TestValidatePair(t *testing.T) {
	c := Capture{ViewportWidthCSS: 1280, ViewportHeightCSS: 800, DevicePixelRatio: 2, ContentStateKey: "default"}
	ref := Asset{ProjectID: "p", Role: RoleReference, Capture: c}
	act := Asset{ProjectID: "p", Role: RoleActual, Capture: c}
	if err := ValidatePair("p", ref, act); err != nil {
		t.Fatal(err)
	}
	act.Capture.DevicePixelRatio = 1
	if err := ValidatePair("p", ref, act); err == nil {
		t.Error("DPR mismatch accepted")
	}
	if err := ValidatePair("p", act, ref); err == nil {
		t.Error("swapped roles accepted")
	}
	if err := ValidatePair("other", ref, ref); err == nil {
		t.Error("wrong project accepted")
	}
}

func TestFingerprintChangesWithInputs(t *testing.T) {
	base := FingerprintInput{ReferenceSHA256: "a", ActualSHA256: "b", RequirementSHA256: "c",
		Origin: OriginMock, ModelID: "m", ConfigVersion: "1", PromptVersion: "1",
		Transform: NewTransformSpec(IdentityTransform(10, 10), IdentityTransform(10, 10))}
	changed := base
	changed.Transform.Actual.CropX = 1
	if base.Fingerprint() == changed.Fingerprint() {
		t.Error("transform change did not change fingerprint")
	}
	swapped := base
	swapped.ReferenceSHA256, swapped.ActualSHA256 = "b", "a"
	if base.Fingerprint() == swapped.Fingerprint() {
		t.Error("asset order is not part of fingerprint")
	}
}

func TestValidateReview(t *testing.T) {
	if _, err := ValidateReview("maybe", ""); err == nil {
		t.Error("invalid decision accepted")
	}
	if _, err := ValidateReview(DecisionRejected, " "); err == nil {
		t.Error("reasonless rejection accepted")
	}
	if r, err := ValidateReview(DecisionAccepted, "  looks right "); err != nil || r != "looks right" {
		t.Errorf("accept = %q, %v", r, err)
	}
}

func TestFitTransform(t *testing.T) {
	if tr := FitTransform(1280, 800, 2048); tr != IdentityTransform(1280, 800) {
		t.Errorf("small image changed: %+v", tr)
	}
	tr := FitTransform(4000, 3000, 2000)
	if tr.AnalysisWidth != 2000 || tr.AnalysisHeight != 1500 || tr.ScaleX != 0.5 {
		t.Fatalf("fit = %+v", tr)
	}
	// A full-image box in analysis space maps back to the full original.
	box, err := tr.ToOriginal(NormalizedBox{X0: 0, Y0: 0, X1: 1000, Y1: 1000})
	if err != nil || box != (Box{X: 0, Y: 0, Width: 4000, Height: 3000}) {
		t.Fatalf("box = %+v err %v", box, err)
	}
	// Odd sizes keep exact per-axis scales, so edges still land on the border.
	tr = FitTransform(3001, 7, 1000)
	box, err = tr.ToOriginal(NormalizedBox{X0: 0, Y0: 0, X1: 1000, Y1: 1000})
	if err != nil || box.Width != 3001 || box.Height != 7 {
		t.Fatalf("odd box = %+v err %v (%+v)", box, err, tr)
	}
}
