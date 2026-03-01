package upscaler

import (
	"image"
	"image/color"
	"math"
	"testing"
)

func solidImage(w, h int, c color.RGBA) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetRGBA(x, y, c)
		}
	}
	return img
}

// ── No-op: factor ≤ 1.0 returns the same pointer ────────────────────────────

func TestScale_NoOp(t *testing.T) {
	factors := []float64{1.0, 0.99, 0.5, 0.0, -1.0}
	for _, f := range factors {
		src := solidImage(100, 100, color.RGBA{R: 200, A: 255})
		result, err := Scale(src, f)
		if err != nil {
			t.Errorf("factor=%.2f: unexpected error: %v", f, err)
		}
		// No allocation: must be the exact same interface value.
		if result != src {
			t.Errorf("factor=%.2f: expected same image pointer (no allocation)", f)
		}
	}
}

// ── Dimensions after upscaling ───────────────────────────────────────────────

func TestScale_Dimensions(t *testing.T) {
	tests := []struct {
		name          string
		w, h          int
		factor        float64
	}{
		{"2× square", 100, 100, 2.0},
		{"3× rectangle", 100, 200, 3.0},
		{"1.5× fractional", 200, 100, 1.5},
		{"1.1× small step", 100, 100, 1.1},
		{"2× odd dimensions", 99, 99, 2.0},
		{"2× tiny image", 1, 1, 2.0},
		{"4× upscale", 50, 50, 4.0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src := solidImage(tt.w, tt.h, color.RGBA{G: 200, A: 255})
			result, err := Scale(src, tt.factor)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			wantW := int(math.Round(float64(tt.w) * tt.factor))
			wantH := int(math.Round(float64(tt.h) * tt.factor))
			b := result.Bounds()
			if b.Dx() != wantW || b.Dy() != wantH {
				t.Errorf("want %d×%d, got %d×%d", wantW, wantH, b.Dx(), b.Dy())
			}
		})
	}
}

// ── Output is a new allocation ───────────────────────────────────────────────

func TestScale_NewAllocation(t *testing.T) {
	src := solidImage(100, 100, color.RGBA{B: 200, A: 255})
	result, err := Scale(src, 2.0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == src {
		t.Error("factor=2.0: expected a new image (not the same pointer)")
	}
}

// ── Output type is *image.RGBA ────────────────────────────────────────────────

func TestScale_OutputIsRGBA(t *testing.T) {
	src := solidImage(50, 50, color.RGBA{R: 255, G: 128, A: 255})
	result, err := Scale(src, 2.0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := result.(*image.RGBA); !ok {
		t.Errorf("expected *image.RGBA output, got %T", result)
	}
}

// ── Bounds start at (0,0) ────────────────────────────────────────────────────

func TestScale_BoundsOrigin(t *testing.T) {
	src := solidImage(60, 40, color.RGBA{R: 100, G: 150, B: 200, A: 255})
	result, err := Scale(src, 2.0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	b := result.Bounds()
	if b.Min.X != 0 || b.Min.Y != 0 {
		t.Errorf("expected bounds origin (0,0), got (%d,%d)", b.Min.X, b.Min.Y)
	}
}

// ── Non-RGBA input (e.g. sub-image from *image.YCbCr) works ──────────────────

func TestScale_NRGBAInput(t *testing.T) {
	src := image.NewNRGBA(image.Rect(0, 0, 40, 40))
	for y := 0; y < 40; y++ {
		for x := 0; x < 40; x++ {
			src.SetNRGBA(x, y, color.NRGBA{R: 200, G: 100, B: 50, A: 255})
		}
	}
	result, err := Scale(src, 1.5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	wantW := int(math.Round(40 * 1.5))
	wantH := int(math.Round(40 * 1.5))
	if result.Bounds().Dx() != wantW || result.Bounds().Dy() != wantH {
		t.Errorf("NRGBA input: want %d×%d, got %d×%d", wantW, wantH, result.Bounds().Dx(), result.Bounds().Dy())
	}
}

// ── Exact factor=1.0 boundary ────────────────────────────────────────────────

func TestScale_ExactlyOne(t *testing.T) {
	src := solidImage(200, 150, color.RGBA{R: 50, G: 100, B: 150, A: 255})
	result, err := Scale(src, 1.0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Must return src unchanged — no resampling, no new allocation.
	if result != src {
		t.Error("Scale(src, 1.0) must return src unchanged")
	}
}
