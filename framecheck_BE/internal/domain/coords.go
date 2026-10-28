package domain

import (
	"fmt"
	"math"
)

// NormalizedScale is the coordinate range Muse Spark uses for grounding on the
// exact image submitted to it.
const NormalizedScale = 1000.0

// ImageTransform records how an analysis image was derived from the original:
// a crop starting at (CropX, CropY) in original pixels, then a scale.
// The identity transform has no crop and scale 1.
type ImageTransform struct {
	OriginalWidth  int     `json:"original_width"`
	OriginalHeight int     `json:"original_height"`
	AnalysisWidth  int     `json:"analysis_width"`
	AnalysisHeight int     `json:"analysis_height"`
	CropX          int     `json:"crop_x"`
	CropY          int     `json:"crop_y"`
	ScaleX         float64 `json:"scale_x"`
	ScaleY         float64 `json:"scale_y"`
}

// TransformSpec is persisted with a run so stored boxes can be audited.
type TransformSpec struct {
	Coordinates string         `json:"coordinates"`
	Rounding    string         `json:"rounding"`
	Reference   ImageTransform `json:"reference"`
	Actual      ImageTransform `json:"actual"`
}

func IdentityTransform(width, height int) ImageTransform {
	return ImageTransform{
		OriginalWidth: width, OriginalHeight: height,
		AnalysisWidth: width, AnalysisHeight: height,
		ScaleX: 1, ScaleY: 1,
	}
}

// FitTransform scales an image down (never up) so its longer side is at most
// maxSide. The scale per axis is recorded exactly as analysis/original so
// boxes map back without drift. maxSide <= 0 means no limit.
func FitTransform(width, height, maxSide int) ImageTransform {
	longest := max(width, height)
	if maxSide <= 0 || longest <= maxSide {
		return IdentityTransform(width, height)
	}
	s := float64(maxSide) / float64(longest)
	aw := max(1, int(math.Round(float64(width)*s)))
	ah := max(1, int(math.Round(float64(height)*s)))
	return ImageTransform{
		OriginalWidth: width, OriginalHeight: height,
		AnalysisWidth: aw, AnalysisHeight: ah,
		ScaleX: float64(aw) / float64(width), ScaleY: float64(ah) / float64(height),
	}
}

func NewTransformSpec(ref, act ImageTransform) TransformSpec {
	return TransformSpec{
		Coordinates: "normalized_0_1000",
		Rounding:    "floor_min_ceil_max",
		Reference:   ref,
		Actual:      act,
	}
}

// NormalizedBox is an untrusted provider box as corners in 0..1000 space.
type NormalizedBox struct {
	X0, Y0, X1, Y1 float64
}

// roundingSlack absorbs float error so an exact edge (1000) does not ceil past
// the original raster boundary.
const roundingSlack = 1e-6

// ToOriginal maps a normalized analysis-image box into original raster pixels
// and rejects anything non-finite, inverted, empty, or out of bounds.
func (t ImageTransform) ToOriginal(nb NormalizedBox) (Box, error) {
	for _, v := range []float64{nb.X0, nb.Y0, nb.X1, nb.Y1} {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return Box{}, fmt.Errorf("box has a non-finite coordinate")
		}
		if v < 0 || v > NormalizedScale {
			return Box{}, fmt.Errorf("box coordinate %.3f is outside 0..1000", v)
		}
	}
	if nb.X1 <= nb.X0 || nb.Y1 <= nb.Y0 {
		return Box{}, fmt.Errorf("box corners are inverted or empty")
	}
	if t.AnalysisWidth <= 0 || t.AnalysisHeight <= 0 || t.ScaleX <= 0 || t.ScaleY <= 0 {
		return Box{}, fmt.Errorf("transform is not usable")
	}

	toX := func(n float64) float64 {
		return float64(t.CropX) + (n*float64(t.AnalysisWidth)/NormalizedScale)/t.ScaleX
	}
	toY := func(n float64) float64 {
		return float64(t.CropY) + (n*float64(t.AnalysisHeight)/NormalizedScale)/t.ScaleY
	}

	x0 := int(math.Floor(toX(nb.X0) + roundingSlack))
	y0 := int(math.Floor(toY(nb.Y0) + roundingSlack))
	x1 := int(math.Ceil(toX(nb.X1) - roundingSlack))
	y1 := int(math.Ceil(toY(nb.Y1) - roundingSlack))

	box := Box{X: x0, Y: y0, Width: x1 - x0, Height: y1 - y0}
	if err := box.Within(t.OriginalWidth, t.OriginalHeight); err != nil {
		return Box{}, err
	}
	return box, nil
}

// Within checks that the box is non-empty and inside a width × height raster.
func (b Box) Within(width, height int) error {
	if b.Width <= 0 || b.Height <= 0 {
		return fmt.Errorf("box is empty")
	}
	if b.X < 0 || b.Y < 0 || b.X+b.Width > width || b.Y+b.Height > height {
		return fmt.Errorf("box %d,%d %dx%d is outside the %dx%d image", b.X, b.Y, b.Width, b.Height, width, height)
	}
	return nil
}
