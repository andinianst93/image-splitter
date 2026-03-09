package trimmer

import (
	"image"
	"image/color"
	"testing"
)

// ── Helpers ───────────────────────────────────────────────────────────────────

func solidImage(w, h int, c color.RGBA) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetRGBA(x, y, c)
		}
	}
	return img
}

// borderedImage creates a w×h image with a uniform borderPx-wide border
// around an interior filled with interiorColor.
func borderedImage(w, h, borderPx int, borderColor, interiorColor color.RGBA) *image.RGBA {
	img := solidImage(w, h, borderColor)
	for y := borderPx; y < h-borderPx; y++ {
		for x := borderPx; x < w-borderPx; x++ {
			img.SetRGBA(x, y, interiorColor)
		}
	}
	return img
}

// mockImage wraps image.Image without exposing SubImage, forcing the fallback
// draw.Draw path in TrimBorder.
type mockImage struct{ image.Image }

// ── colorDiff ─────────────────────────────────────────────────────────────────

func TestColorDiff_SameColor(t *testing.T) {
	c := color.RGBA{R: 100, G: 150, B: 200, A: 255}
	if d := colorDiff(c, c); d != 0 {
		t.Errorf("same color: want 0, got %d", d)
	}
}

func TestColorDiff_MaxChannel(t *testing.T) {
	a := color.RGBA{R: 50, G: 50, B: 50, A: 255}
	b := color.RGBA{R: 50, G: 80, B: 50, A: 255}
	if d := colorDiff(a, b); d != 30 {
		t.Errorf("want 30 (G channel diff), got %d", d)
	}
}

func TestColorDiff_AlphaIgnored(t *testing.T) {
	a := color.RGBA{R: 128, G: 128, B: 128, A: 0}
	b := color.RGBA{R: 128, G: 128, B: 128, A: 255}
	if d := colorDiff(a, b); d != 0 {
		t.Errorf("alpha should be ignored: want 0, got %d", d)
	}
}

// ── TrimBorder ────────────────────────────────────────────────────────────────

func TestTrimBorder_SolidBorderRemoved(t *testing.T) {
	border := color.RGBA{R: 40, G: 40, B: 40, A: 255}
	interior := color.RGBA{R: 200, G: 100, B: 50, A: 255}
	img := borderedImage(100, 80, 10, border, interior)

	result := TrimBorder(img, 15)

	b := result.Bounds()
	if b.Dx() != 80 || b.Dy() != 60 {
		t.Errorf("want 80×60 after trim, got %d×%d", b.Dx(), b.Dy())
	}
	got := toRGBA(result.At(b.Min.X, b.Min.Y))
	if got != interior {
		t.Errorf("top-left after trim: want %v, got %v", interior, got)
	}
}

func TestTrimBorder_NoBorder_ReturnsUnchanged(t *testing.T) {
	// Solid image — all corners same color, walk finds nothing to remove.
	interior := color.RGBA{R: 200, G: 100, B: 50, A: 255}
	img := solidImage(60, 40, interior)

	result := TrimBorder(img, 15)

	if result.Bounds() != img.Bounds() {
		t.Errorf("no border: bounds should be unchanged; want %v, got %v",
			img.Bounds(), result.Bounds())
	}
}

func TestTrimBorder_CornersDisagree_NoTrim(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 60, 60))
	img.SetRGBA(0, 0, color.RGBA{R: 255, A: 255})
	img.SetRGBA(59, 0, color.RGBA{G: 255, A: 255})
	img.SetRGBA(0, 59, color.RGBA{B: 255, A: 255})
	img.SetRGBA(59, 59, color.RGBA{R: 255, G: 255, A: 255})

	result := TrimBorder(img, 15)

	if result != image.Image(img) {
		t.Error("corners disagree: expected src returned unchanged")
	}
}

func TestTrimBorder_TinyResult_ReturnsOriginal(t *testing.T) {
	// 100×100 with 46px border → 8×8 interior, below 10px minimum.
	border := color.RGBA{R: 40, G: 40, B: 40, A: 255}
	interior := color.RGBA{R: 200, G: 100, B: 50, A: 255}
	img := borderedImage(100, 100, 46, border, interior)

	result := TrimBorder(img, 15)

	if result.Bounds() != img.Bounds() {
		t.Error("tiny result: expected original returned unchanged")
	}
}

func TestTrimBorder_WithinTolerance_Trimmed(t *testing.T) {
	// Corners are (10,10,10); some border rows are (20,20,20) — diff 10 ≤ tol 15.
	nearBlack := color.RGBA{R: 10, G: 10, B: 10, A: 255}
	slightlyOff := color.RGBA{R: 20, G: 20, B: 20, A: 255}
	interior := color.RGBA{R: 200, G: 150, B: 100, A: 255}

	img := image.NewRGBA(image.Rect(0, 0, 80, 60))
	for y := 0; y < 60; y++ {
		for x := 0; x < 80; x++ {
			img.SetRGBA(x, y, nearBlack)
		}
	}
	for y := 1; y < 5; y++ {
		for x := 1; x < 79; x++ {
			img.SetRGBA(x, y, slightlyOff)
		}
	}
	for y := 55; y < 59; y++ {
		for x := 1; x < 79; x++ {
			img.SetRGBA(x, y, slightlyOff)
		}
	}
	for y := 5; y < 55; y++ {
		for x := 5; x < 75; x++ {
			img.SetRGBA(x, y, interior)
		}
	}

	result := TrimBorder(img, 15)

	if result.Bounds().Dx() >= 80 || result.Bounds().Dy() >= 60 {
		t.Errorf("within-tolerance border should be trimmed; got %v (original 80×60)",
			result.Bounds())
	}
}

func TestTrimBorder_SubImagePath(t *testing.T) {
	border := color.RGBA{R: 30, G: 30, B: 30, A: 255}
	interior := color.RGBA{R: 180, G: 120, B: 60, A: 255}
	img := borderedImage(80, 60, 8, border, interior)

	result := TrimBorder(img, 15)

	if _, ok := result.(*image.RGBA); !ok {
		t.Errorf("SubImage path: expected *image.RGBA, got %T", result)
	}
	b := result.Bounds()
	if b.Dx() != 64 || b.Dy() != 44 {
		t.Errorf("SubImage path: want 64×44, got %d×%d", b.Dx(), b.Dy())
	}
}

func TestTrimBorder_FallbackPath(t *testing.T) {
	border := color.RGBA{R: 30, G: 30, B: 30, A: 255}
	interior := color.RGBA{R: 180, G: 120, B: 60, A: 255}
	img := &mockImage{borderedImage(80, 60, 8, border, interior)}

	result := TrimBorder(img, 15)

	b := result.Bounds()
	if b.Dx() != 64 || b.Dy() != 44 {
		t.Errorf("fallback path: want 64×44, got %d×%d", b.Dx(), b.Dy())
	}
	if b.Min.X != 0 || b.Min.Y != 0 {
		t.Errorf("fallback path: want origin (0,0), got (%d,%d)", b.Min.X, b.Min.Y)
	}
	got := toRGBA(result.At(0, 0))
	if got != interior {
		t.Errorf("fallback path pixel (0,0): want %v, got %v", interior, got)
	}
}

func TestTrimBorder_NonZeroMinBounds(t *testing.T) {
	border := color.RGBA{R: 40, G: 40, B: 40, A: 255}
	interior := color.RGBA{R: 200, G: 100, B: 50, A: 255}
	// 100×100 with 20px border → interior at x:20-79, y:20-79.
	base := borderedImage(100, 100, 20, border, interior)

	// Sub-image has 10px border on all sides (base border extends into sub), Min=(10,10).
	sub := base.SubImage(image.Rect(10, 10, 90, 90)) // 80×80
	result := TrimBorder(sub, 15)

	b := result.Bounds()
	if b.Dx() != 60 || b.Dy() != 60 {
		t.Errorf("non-zero Min: want 60×60, got %d×%d", b.Dx(), b.Dy())
	}
}

func TestTrimBorder_AsymmetricBorder(t *testing.T) {
	// Top=5, Bottom=15, Left=8, Right=12.
	border := color.RGBA{R: 50, G: 50, B: 50, A: 255}
	interior := color.RGBA{R: 255, G: 0, B: 128, A: 255}
	w, h := 100, 80

	img := solidImage(w, h, border)
	for y := 5; y < h-15; y++ {
		for x := 8; x < w-12; x++ {
			img.SetRGBA(x, y, interior)
		}
	}

	result := TrimBorder(img, 15)

	b := result.Bounds()
	if b.Dx() != 80 || b.Dy() != 60 {
		t.Errorf("asymmetric border: want 80×60, got %d×%d", b.Dx(), b.Dy())
	}
}

func TestTrimBorder_ZeroSize(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 0, 0))
	result := TrimBorder(img, 15)
	if result != image.Image(img) {
		t.Error("zero-size: expected src unchanged")
	}
}
