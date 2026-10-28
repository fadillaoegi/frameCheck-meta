package meta

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/color"
	_ "image/jpeg" // register decoder
	"image/png"

	"framecheck/internal/ports"
)

// dataURL returns the image as sent to the model: the original bytes when no
// downscale is needed, otherwise an area-averaged PNG at the transform's
// analysis size, so returned coordinates refer to exactly that raster.
func dataURL(img ports.AnalysisImage) (string, error) {
	t := img.Transform
	data, mime := img.Data, img.MIME
	if t.AnalysisWidth != t.OriginalWidth || t.AnalysisHeight != t.OriginalHeight || t.CropX != 0 || t.CropY != 0 {
		src, _, err := image.Decode(bytes.NewReader(img.Data))
		if err != nil {
			return "", fmt.Errorf("decode %s image: %w", img.Role, err)
		}
		if src.Bounds().Dx() != t.OriginalWidth || src.Bounds().Dy() != t.OriginalHeight {
			return "", fmt.Errorf("%s image size does not match its transform", img.Role)
		}
		var buf bytes.Buffer
		if err := png.Encode(&buf, downscale(src, t.CropX, t.CropY, t.AnalysisWidth, t.AnalysisHeight, t.ScaleX, t.ScaleY)); err != nil {
			return "", err
		}
		data, mime = buf.Bytes(), "image/png"
	}
	return "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(data), nil
}

// downscale averages every source pixel that falls in each destination pixel.
// It only shrinks; FitTransform never asks for enlargement.
func downscale(src image.Image, cropX, cropY, w, h int, sx, sy float64) *image.NRGBA {
	dst := image.NewNRGBA(image.Rect(0, 0, w, h))
	b := src.Bounds()
	for y := 0; y < h; y++ {
		y0 := b.Min.Y + cropY + int(float64(y)/sy)
		y1 := max(y0+1, b.Min.Y+cropY+int(float64(y+1)/sy))
		y1 = min(y1, b.Max.Y)
		for x := 0; x < w; x++ {
			x0 := b.Min.X + cropX + int(float64(x)/sx)
			x1 := max(x0+1, b.Min.X+cropX+int(float64(x+1)/sx))
			x1 = min(x1, b.Max.X)
			var r, g, bl, a, n uint64
			for yy := y0; yy < y1; yy++ {
				for xx := x0; xx < x1; xx++ {
					c := color.NRGBAModel.Convert(src.At(xx, yy)).(color.NRGBA)
					r, g, bl, a = r+uint64(c.R), g+uint64(c.G), bl+uint64(c.B), a+uint64(c.A)
					n++
				}
			}
			if n > 0 {
				dst.SetNRGBA(x, y, color.NRGBA{R: uint8(r / n), G: uint8(g / n), B: uint8(bl / n), A: uint8(a / n)})
			}
		}
	}
	return dst
}
