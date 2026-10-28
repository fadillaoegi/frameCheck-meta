package mock

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"testing"

	"framecheck/internal/domain"
	"framecheck/internal/ports"
)

func encode(t *testing.T, img image.Image) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func canvas(w, h int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(img, img.Bounds(), image.NewUniform(color.White), image.Point{}, draw.Src)
	return img
}

func input(ref, act []byte) ports.AnalysisInput {
	return ports.AnalysisInput{
		Reference: ports.AnalysisImage{Data: ref},
		Actual:    ports.AnalysisImage{Data: act},
	}
}

func TestIdenticalImagesHaveNoFindings(t *testing.T) {
	b := encode(t, canvas(64, 64))
	out, err := New().Analyze(context.Background(), input(b, b))
	if err != nil {
		t.Fatal(err)
	}
	if !out.Complete || len(out.Findings) != 0 {
		t.Fatalf("complete=%v findings=%d", out.Complete, len(out.Findings))
	}
}

func TestDifferentSizesAreIncomplete(t *testing.T) {
	out, err := New().Analyze(context.Background(), input(encode(t, canvas(64, 64)), encode(t, canvas(32, 64))))
	if err != nil {
		t.Fatal(err)
	}
	if out.Complete {
		t.Fatal("size mismatch reported as complete")
	}
}

func TestDiffRegionMapsBackToOriginalPixels(t *testing.T) {
	ref := canvas(320, 160)
	act := canvas(320, 160)
	// A button-sized block that exists only in the reference.
	draw.Draw(ref, image.Rect(40, 20, 100, 50), image.NewUniform(color.Black), image.Point{}, draw.Src)

	out, err := New().Analyze(context.Background(), input(encode(t, ref), encode(t, act)))
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Findings) != 1 {
		t.Fatalf("findings = %d", len(out.Findings))
	}
	box, err := domain.IdentityTransform(320, 160).ToOriginal(*out.Findings[0].Reference.Box)
	if err != nil {
		t.Fatal(err)
	}
	// The diff is snapped outward to 16 px cells: x 32..112, y 16..64.
	if box != (domain.Box{X: 32, Y: 16, Width: 80, Height: 48}) {
		t.Fatalf("box = %+v", box)
	}
}
