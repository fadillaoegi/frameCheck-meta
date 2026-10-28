// Package mock is a deterministic, offline inference provider for development.
// It finds regions where pixels differ and reports them as layout_mismatch
// candidates. It has no understanding of UI semantics; every run it produces
// is labelled origin "mock" and must never be presented as model output.
package mock

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/color"
	_ "image/jpeg" // register decoder
	_ "image/png"  // register decoder
	"sort"

	"framecheck/internal/domain"
	"framecheck/internal/ports"
)

const (
	cellSize = 16
	// channelTolerance ignores tiny compression or anti-aliasing noise (0..65535 scale).
	channelTolerance = 24 * 257
	maxRegions       = 5
)

type Provider struct{}

func New() *Provider { return &Provider{} }

func (*Provider) Origin() domain.RunOrigin { return domain.OriginMock }
func (*Provider) ModelID() string          { return "mock-pixel-diff" }
func (*Provider) ConfigVersion() string    { return "mock-1" }
func (*Provider) PromptVersion() string    { return "none" }

func (*Provider) Analyze(ctx context.Context, in ports.AnalysisInput) (ports.AnalysisOutput, error) {
	ref, _, err := image.Decode(bytes.NewReader(in.Reference.Data))
	if err != nil {
		return ports.AnalysisOutput{}, fmt.Errorf("decode reference: %w", err)
	}
	act, _, err := image.Decode(bytes.NewReader(in.Actual.Data))
	if err != nil {
		return ports.AnalysisOutput{}, fmt.Errorf("decode actual: %w", err)
	}
	rb, ab := ref.Bounds(), act.Bounds()
	if rb.Dx() != ab.Dx() || rb.Dy() != ab.Dy() {
		// A pixel diff across different raster sizes would be meaningless.
		return ports.AnalysisOutput{Complete: false}, nil
	}

	w, h := rb.Dx(), rb.Dy()
	cols, rows := (w+cellSize-1)/cellSize, (h+cellSize-1)/cellSize
	diff := make([]bool, cols*rows)
	for y := 0; y < h; y++ {
		if y%cellSize == 0 && ctx.Err() != nil {
			return ports.AnalysisOutput{}, ctx.Err()
		}
		for x := 0; x < w; x++ {
			cell := (y/cellSize)*cols + x/cellSize
			if diff[cell] {
				continue
			}
			if pixelsDiffer(ref.At(rb.Min.X+x, rb.Min.Y+y), act.At(ab.Min.X+x, ab.Min.Y+y)) {
				diff[cell] = true
			}
		}
	}

	regions := connectedRegions(diff, cols, rows)
	sort.Slice(regions, func(i, j int) bool { return regions[i].cells > regions[j].cells })
	if len(regions) > maxRegions {
		regions = regions[:maxRegions]
	}

	out := ports.AnalysisOutput{Complete: true}
	for i, r := range regions {
		x0, y0 := r.minCol*cellSize, r.minRow*cellSize
		x1, y1 := min((r.maxCol+1)*cellSize, w), min((r.maxRow+1)*cellSize, h)
		nb := &domain.NormalizedBox{
			X0: float64(x0) * domain.NormalizedScale / float64(w),
			Y0: float64(y0) * domain.NormalizedScale / float64(h),
			X1: float64(x1) * domain.NormalizedScale / float64(w),
			Y1: float64(y1) * domain.NormalizedScale / float64(h),
		}
		out.Findings = append(out.Findings, domain.RawFinding{
			Category:  string(domain.CategoryLayoutMismatch),
			Summary:   fmt.Sprintf("Pixel difference region %d (mock heuristic, not a semantic finding)", i+1),
			CheckStep: "Compare this region in both images and decide whether the difference is a real defect.",
			Reference: &domain.RawEvidence{ImageRole: "reference", Box: nb, Description: "Region as designed"},
			Actual:    &domain.RawEvidence{ImageRole: "actual", Box: nb, Description: "Same region in the implementation"},
		})
	}
	return out, nil
}

func pixelsDiffer(p, q color.Color) bool {
	r1, g1, b1, a1 := p.RGBA()
	r2, g2, b2, a2 := q.RGBA()
	return absDiff(r1, r2) > channelTolerance || absDiff(g1, g2) > channelTolerance ||
		absDiff(b1, b2) > channelTolerance || absDiff(a1, a2) > channelTolerance
}

func absDiff(a, b uint32) uint32 {
	if a > b {
		return a - b
	}
	return b - a
}

type region struct {
	minCol, minRow, maxCol, maxRow, cells int
}

// connectedRegions groups 8-connected differing cells.
func connectedRegions(diff []bool, cols, rows int) []region {
	seen := make([]bool, len(diff))
	var out []region
	stack := make([]int, 0, 64)
	for start, d := range diff {
		if !d || seen[start] {
			continue
		}
		r := region{minCol: cols, minRow: rows, maxCol: -1, maxRow: -1}
		stack = append(stack[:0], start)
		seen[start] = true
		for len(stack) > 0 {
			i := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			c, rw := i%cols, i/cols
			r.cells++
			r.minCol, r.maxCol = min(r.minCol, c), max(r.maxCol, c)
			r.minRow, r.maxRow = min(r.minRow, rw), max(r.maxRow, rw)
			for dy := -1; dy <= 1; dy++ {
				for dx := -1; dx <= 1; dx++ {
					nc, nr := c+dx, rw+dy
					if nc < 0 || nr < 0 || nc >= cols || nr >= rows {
						continue
					}
					j := nr*cols + nc
					if diff[j] && !seen[j] {
						seen[j] = true
						stack = append(stack, j)
					}
				}
			}
		}
		out = append(out, r)
	}
	return out
}
