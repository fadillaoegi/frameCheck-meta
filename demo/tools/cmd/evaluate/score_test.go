package main

import (
	"testing"

	"framecheck/demo/tools/internal/fixtures"
)

func bx(x, y, w, h int) *fixtures.Box { return &fixtures.Box{X: x, Y: y, Width: w, Height: h} }

func TestIoU(t *testing.T) {
	if v := bx(0, 0, 10, 10).IoU(*bx(0, 0, 10, 10)); v != 1 {
		t.Errorf("identical = %v", v)
	}
	if v := bx(0, 0, 10, 10).IoU(*bx(5, 0, 10, 10)); v < 0.333 || v > 0.334 {
		t.Errorf("half overlap = %v", v)
	}
	if v := bx(0, 0, 10, 10).IoU(*bx(20, 20, 5, 5)); v != 0 {
		t.Errorf("disjoint = %v", v)
	}
}

func TestScore(t *testing.T) {
	missing := []fixtures.Expected{{Category: "missing_element", ReferenceBox: bx(64, 376, 196, 52)}}
	hit := finding{Category: "missing_element", ReferenceEvidence: &evidence{Box: bx(60, 370, 200, 60)}}
	wrongCat := finding{Category: "layout_mismatch", ActualEvidence: &evidence{Box: bx(64, 376, 196, 52)}}
	noBox := finding{Category: "missing_element", ReferenceEvidence: &evidence{}}

	tp, fn, fps, _, _ := score(missing, []finding{hit}, 0.5)
	if tp != 1 || fn != 0 || len(fps) != 0 {
		t.Errorf("hit: tp=%d fn=%d fps=%v", tp, fn, fps)
	}
	tp, fn, fps, _, _ = score(missing, []finding{wrongCat, noBox}, 0.5)
	if tp != 0 || fn != 1 || fps["layout_mismatch"] != 1 || fps["missing_element"] != 1 {
		t.Errorf("misses: tp=%d fn=%d fps=%v", tp, fn, fps)
	}
	// A second finding on the same region is a duplicate, counted as a false positive.
	tp, _, fps, _, _ = score(missing, []finding{hit, hit}, 0.5)
	if tp != 1 || fps["missing_element"] != 1 {
		t.Errorf("duplicate: tp=%d fps=%v", tp, fps)
	}
	// Controls: any finding is a false positive.
	_, _, fps, _, _ = score(nil, []finding{hit}, 0.5)
	if fps["missing_element"] != 1 {
		t.Errorf("control fps=%v", fps)
	}
}
