package splitter

import (
	"image"
	"image/color"
	"testing"
)

// solidImage creates a w×h image filled with a single solid color.
func solidImage(w, h int, c color.RGBA) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetRGBA(x, y, c)
		}
	}
	return img
}

// quadrantImage creates a w×h image with 4 distinct solid-color quadrants
// (w and h must be even):
//
//	top-left=red  top-right=green
//	bottom-left=blue  bottom-right=yellow
func quadrantImage(w, h int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	palette := [2][2]color.RGBA{
		{{R: 255, A: 255}, {G: 255, A: 255}},
		{{B: 255, A: 255}, {R: 255, G: 255, A: 255}},
	}
	hw, hh := w/2, h/2
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			qr, qc := y/hh, x/hw
			if qr >= 2 {
				qr = 1
			}
			if qc >= 2 {
				qc = 1
			}
			img.SetRGBA(x, y, palette[qr][qc])
		}
	}
	return img
}

// mockImage wraps image.Image without exposing SubImage, forcing the fallback
// draw.Draw path in Split.
type mockImage struct{ image.Image }

// ── Count & row-major order ──────────────────────────────────────────────────

func TestSplit_CountAndRowMajorOrder(t *testing.T) {
	src := solidImage(600, 400, color.RGBA{R: 255, A: 255})
	cells, err := Split(src, 2, 3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cells) != 6 {
		t.Fatalf("want 6 cells, got %d", len(cells))
	}
	for i, cell := range cells {
		wantRow, wantCol := i/3, i%3
		if cell.Row != wantRow || cell.Col != wantCol {
			t.Errorf("cells[%d]: want [%d,%d], got [%d,%d]", i, wantRow, wantCol, cell.Row, cell.Col)
		}
	}
}

// ── Cell dimensions for evenly-divisible inputs ──────────────────────────────

func TestSplit_EvenDimensions(t *testing.T) {
	tests := []struct {
		name             string
		w, h, rows, cols int
		wantW, wantH     int
	}{
		{"2x3 grid", 600, 400, 2, 3, 200, 200},
		{"1x1 grid", 300, 200, 1, 1, 300, 200},
		{"4x4 grid", 400, 400, 4, 4, 100, 100},
		{"1x4 horizontal strip", 400, 100, 1, 4, 100, 100},
		{"4x1 vertical strip", 100, 400, 4, 1, 100, 100},
		{"single pixel cells", 3, 2, 2, 3, 1, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src := solidImage(tt.w, tt.h, color.RGBA{G: 255, A: 255})
			cells, err := Split(src, tt.rows, tt.cols)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(cells) != tt.rows*tt.cols {
				t.Fatalf("want %d cells, got %d", tt.rows*tt.cols, len(cells))
			}
			// Interior cells must have exact dimensions (last row/col may differ).
			for _, cell := range cells {
				if cell.Row == tt.rows-1 || cell.Col == tt.cols-1 {
					continue
				}
				b := cell.Image.Bounds()
				if b.Dx() != tt.wantW || b.Dy() != tt.wantH {
					t.Errorf("cell [%d,%d]: want %dx%d, got %dx%d",
						cell.Row, cell.Col, tt.wantW, tt.wantH, b.Dx(), b.Dy())
				}
			}
		})
	}
}

// ── Last row/col absorbs remainder pixels ────────────────────────────────────

func TestSplit_NonDivisibleDimensions(t *testing.T) {
	// 601×401 into 2×3: cellW=200, cellH=200; last col→201 wide, last row→201 tall.
	src := solidImage(601, 401, color.RGBA{G: 200, A: 255})
	cells, err := Split(src, 2, 3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, cell := range cells {
		b := cell.Image.Bounds()
		wantW := 200
		if cell.Col == 2 {
			wantW = 201
		}
		wantH := 200
		if cell.Row == 1 {
			wantH = 201
		}
		if b.Dx() != wantW {
			t.Errorf("cell [%d,%d]: want width %d, got %d", cell.Row, cell.Col, wantW, b.Dx())
		}
		if b.Dy() != wantH {
			t.Errorf("cell [%d,%d]: want height %d, got %d", cell.Row, cell.Col, wantH, b.Dy())
		}
	}
}

func TestSplit_SinglePixelRemainder(t *testing.T) {
	// 5 wide, 3 cols → cellW=1; last col gets 3 extra pixels.
	src := solidImage(5, 3, color.RGBA{R: 128, A: 255})
	cells, err := Split(src, 1, 3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	widths := []int{1, 1, 3} // 5/3=1, last col: 5-(2*1)=3
	for _, cell := range cells {
		if cell.Image.Bounds().Dx() != widths[cell.Col] {
			t.Errorf("col %d: want width %d, got %d", cell.Col, widths[cell.Col], cell.Image.Bounds().Dx())
		}
	}
}

// ── Pixel-level correctness ──────────────────────────────────────────────────

func TestSplit_PixelCorrectness(t *testing.T) {
	red := color.RGBA{R: 255, A: 255}
	green := color.RGBA{G: 255, A: 255}
	blue := color.RGBA{B: 255, A: 255}
	yellow := color.RGBA{R: 255, G: 255, A: 255}

	src := quadrantImage(4, 4) // 2x2 split → each quadrant 2x2 pixels
	cells, err := Split(src, 2, 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := map[[2]int]color.RGBA{
		{0, 0}: red, {0, 1}: green,
		{1, 0}: blue, {1, 1}: yellow,
	}
	for _, cell := range cells {
		b := cell.Image.Bounds()
		got := color.RGBAModel.Convert(cell.Image.At(b.Min.X, b.Min.Y)).(color.RGBA)
		wantColor := want[[2]int{cell.Row, cell.Col}]
		if got != wantColor {
			t.Errorf("cell [%d,%d]: want %v, got %v", cell.Row, cell.Col, wantColor, got)
		}
	}
}

// ── Origin-aware bounds (image with non-zero Min) ────────────────────────────

func TestSplit_NonZeroBounds(t *testing.T) {
	// Create a 400×200 base then sub-image at offset (100,50) → Min=(100,50).
	base := image.NewRGBA(image.Rect(0, 0, 400, 200))
	for y := 0; y < 200; y++ {
		for x := 0; x < 400; x++ {
			base.SetRGBA(x, y, color.RGBA{R: uint8(x), G: uint8(y), A: 255})
		}
	}
	src := base.SubImage(image.Rect(100, 50, 300, 150)) // 200×100, Min=(100,50)
	if src.Bounds().Min.X != 100 || src.Bounds().Min.Y != 50 {
		t.Fatal("test setup: expected non-zero bounds Min")
	}

	cells, err := Split(src, 2, 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cells) != 4 {
		t.Fatalf("want 4 cells, got %d", len(cells))
	}
	for _, cell := range cells {
		b := cell.Image.Bounds()
		if b.Dx() != 100 || b.Dy() != 50 {
			t.Errorf("cell [%d,%d]: want 100×50, got %d×%d", cell.Row, cell.Col, b.Dx(), b.Dy())
		}
	}
}

// ── Fallback path (no SubImage) ──────────────────────────────────────────────

func TestSplit_FallbackPath_Dimensions(t *testing.T) {
	src := &mockImage{solidImage(100, 80, color.RGBA{R: 128, G: 64, A: 255})}
	cells, err := Split(src, 2, 4)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cells) != 8 {
		t.Fatalf("want 8 cells, got %d", len(cells))
	}
	for _, cell := range cells {
		b := cell.Image.Bounds()
		// Fallback copies to a new image that always starts at (0,0).
		if b.Min.X != 0 || b.Min.Y != 0 {
			t.Errorf("fallback cell [%d,%d]: want Min=(0,0), got %v", cell.Row, cell.Col, b.Min)
		}
		if cell.Row < 1 && cell.Col < 3 {
			if b.Dx() != 25 || b.Dy() != 40 {
				t.Errorf("fallback cell [%d,%d]: want 25×40, got %d×%d", cell.Row, cell.Col, b.Dx(), b.Dy())
			}
		}
	}
}

func TestSplit_FallbackPath_Pixels(t *testing.T) {
	// Verify pixel values survive the draw.Draw copy.
	src := &mockImage{quadrantImage(4, 4)}
	cells, err := Split(src, 2, 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	red := color.RGBA{R: 255, A: 255}
	// Fallback cells start at (0,0), so use At(0,0) directly.
	got := color.RGBAModel.Convert(cells[0].Image.At(0, 0)).(color.RGBA)
	if got != red {
		t.Errorf("fallback top-left pixel: want %v, got %v", red, got)
	}
}

// ── Error cases ──────────────────────────────────────────────────────────────

func TestSplit_Errors(t *testing.T) {
	src := solidImage(100, 100, color.RGBA{})
	tests := []struct {
		name       string
		rows, cols int
	}{
		{"zero rows", 0, 2},
		{"zero cols", 2, 0},
		{"negative rows", -1, 2},
		{"negative cols", 2, -1},
		{"both zero", 0, 0},
		{"rows causes zero cellH", 101, 1}, // 100/101 = 0
		{"cols causes zero cellW", 1, 101}, // 100/101 = 0
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Split(src, tt.rows, tt.cols)
			if err == nil {
				t.Errorf("rows=%d cols=%d: expected error, got nil", tt.rows, tt.cols)
			}
		})
	}
}

// ── Edge: 1×1 grid returns the full image ────────────────────────────────────

func TestSplit_OneByOne(t *testing.T) {
	src := solidImage(300, 200, color.RGBA{R: 255, G: 128, A: 255})
	cells, err := Split(src, 1, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cells) != 1 {
		t.Fatalf("want 1 cell, got %d", len(cells))
	}
	b := cells[0].Image.Bounds()
	if b.Dx() != 300 || b.Dy() != 200 {
		t.Errorf("want 300×200, got %d×%d", b.Dx(), b.Dy())
	}
}

// ── Total area of all cells equals source image area ─────────────────────────

func TestSplit_TotalAreaPreserved(t *testing.T) {
	tests := []struct{ w, h, rows, cols int }{
		{600, 400, 2, 3},
		{601, 401, 2, 3}, // non-divisible
		{100, 100, 7, 7}, // large remainder
		{50, 50, 1, 1},
	}
	for _, tt := range tests {
		src := solidImage(tt.w, tt.h, color.RGBA{A: 255})
		cells, err := Split(src, tt.rows, tt.cols)
		if err != nil {
			t.Fatalf("%dx%d %d×%d: %v", tt.w, tt.h, tt.rows, tt.cols, err)
		}
		var totalArea int
		for _, cell := range cells {
			b := cell.Image.Bounds()
			totalArea += b.Dx() * b.Dy()
		}
		want := tt.w * tt.h
		if totalArea != want {
			t.Errorf("%dx%d %d×%d: total area %d ≠ %d", tt.w, tt.h, tt.rows, tt.cols, totalArea, want)
		}
	}
}
