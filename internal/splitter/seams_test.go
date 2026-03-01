package splitter

import (
	"image"
	"image/color"
	"testing"
)

// ── Test helpers ─────────────────────────────────────────────────────────────

// stripedHoriz builds a w×(sum(heights)) image with alternating dark/light
// horizontal bands whose pixel-exact heights are given.
func stripedHoriz(w int, heights []int) *image.RGBA {
	colors := []color.RGBA{
		{R: 20, G: 20, B: 20, A: 255},
		{R: 220, G: 220, B: 220, A: 255},
	}
	img := image.NewRGBA(image.Rect(0, 0, w, sumInts(heights)))
	y := 0
	for i := range heights {
		c := colors[i%2]
		for ; y < sumInts(heights[:i+1]); y++ {
			for x := 0; x < w; x++ {
				img.SetRGBA(x, y, c)
			}
		}
	}
	return img
}

// stripedVert builds a (sum(widths))×h image with alternating dark/light
// vertical bands.
func stripedVert(h int, widths []int) *image.RGBA {
	colors := []color.RGBA{
		{R: 20, G: 20, B: 20, A: 255},
		{R: 220, G: 220, B: 220, A: 255},
	}
	img := image.NewRGBA(image.Rect(0, 0, sumInts(widths), h))
	x := 0
	for i := range widths {
		c := colors[i%2]
		for ; x < sumInts(widths[:i+1]); x++ {
			for y := 0; y < h; y++ {
				img.SetRGBA(x, y, c)
			}
		}
	}
	return img
}

// gappedHoriz builds a striped image with uniform-color gap lines between bands.
// gapColor is the separator color (e.g. white or black).
func gappedHoriz(w, bandH, bands, gapPx int, gapColor color.RGBA) *image.RGBA {
	totalH := bands*bandH + (bands-1)*gapPx
	img := image.NewRGBA(image.Rect(0, 0, w, totalH))
	colors := []color.RGBA{
		{R: 30, G: 60, B: 120, A: 255},
		{R: 200, G: 160, B: 80, A: 255},
	}
	y := 0
	for i := 0; i < bands; i++ {
		c := colors[i%2]
		for j := 0; j < bandH; j++ {
			for x := 0; x < w; x++ {
				img.SetRGBA(x, y, c)
			}
			y++
		}
		if i < bands-1 {
			for j := 0; j < gapPx; j++ {
				for x := 0; x < w; x++ {
					img.SetRGBA(x, y, gapColor)
				}
				y++
			}
		}
	}
	return img
}

// noiseHoriz builds a striped image where the interior of each band has
// a strong internal horizontal edge (simulates a photo with prominent edges).
func noiseHoriz(w int, heights []int) *image.RGBA {
	base := stripedHoriz(w, heights)
	// Add a strong internal edge at the midpoint of every band.
	y := 0
	for i := range heights {
		mid := y + heights[i]/2
		// Flip brightness at the midpoint row (creates a fake strong internal edge).
		for x := 0; x < w; x++ {
			px := base.RGBAAt(x, mid)
			base.SetRGBA(x, mid, color.RGBA{
				R: 255 - px.R, G: 255 - px.G, B: 255 - px.B, A: 255,
			})
		}
		y += heights[i]
	}
	return base
}

func sumInts(s []int) int {
	n := 0
	for _, v := range s {
		n += v
	}
	return n
}

// withinTol returns true if |got-want| <= tol.
func withinTol(got, want, tol int) bool {
	d := got - want
	if d < 0 {
		d = -d
	}
	return d <= tol
}

// ── DetectHorizSeams ─────────────────────────────────────────────────────────

func TestDetectHorizSeams_EqualBands(t *testing.T) {
	src := stripedHoriz(300, []int{100, 100, 100, 100})
	seams := DetectHorizSeams(src, 4)
	if len(seams) != 3 {
		t.Fatalf("want 3 seams, got %d: %v", len(seams), seams)
	}
	for i, got := range seams {
		want := (i + 1) * 100
		if !withinTol(got, want, 2) {
			t.Errorf("seam %d: want ~%d, got %d", i, want, got)
		}
	}
}

func TestDetectHorizSeams_UnequalBands(t *testing.T) {
	// Simulate real collage: rows with slightly different heights (±4px).
	heights := []int{118, 122, 121, 119} // total 480
	src := stripedHoriz(300, heights)
	seams := DetectHorizSeams(src, 4)
	if len(seams) != 3 {
		t.Fatalf("want 3 seams, got %d: %v", len(seams), seams)
	}
	expected := []int{118, 240, 361}
	for i, got := range seams {
		if !withinTol(got, expected[i], 3) {
			t.Errorf("seam %d: want ~%d, got %d", i, expected[i], got)
		}
	}
}

func TestDetectHorizSeams_WithGap_White(t *testing.T) {
	// Many collage apps add a white gap between photos.
	white := color.RGBA{R: 255, G: 255, B: 255, A: 255}
	src := gappedHoriz(300, 100, 3, 4, white) // 3 bands × 100px + 2 gaps × 4px = 308px
	seams := DetectHorizSeams(src, 3)
	if len(seams) != 2 {
		t.Fatalf("want 2 seams, got %d: %v", len(seams), seams)
	}
	// Seams should fall within or near the gap regions (y≈100..103 and y≈204..207).
	for i, got := range seams {
		center := (i + 1) * 100 // approximate center of gap
		if !withinTol(got, center, 8) {
			t.Errorf("seam %d: want near %d (gap center), got %d", i, center, got)
		}
	}
}

func TestDetectHorizSeams_WithGap_Black(t *testing.T) {
	black := color.RGBA{A: 255}
	// 4 bands × 80px + 3 gaps × 6px = 338px total
	// Gap regions start at: y=80, y=166, y=252
	src := gappedHoriz(200, 80, 4, 6, black)
	seams := DetectHorizSeams(src, 4)
	if len(seams) != 3 {
		t.Fatalf("want 3 seams, got %d: %v", len(seams), seams)
	}
	// Each seam should land near the start of its gap region.
	gapStarts := []int{80, 166, 252}
	for i, got := range seams {
		if !withinTol(got, gapStarts[i], 12) {
			t.Errorf("seam %d: want near %d (gap start), got %d", i, gapStarts[i], got)
		}
	}
}

func TestDetectHorizSeams_StrongInternalEdge(t *testing.T) {
	// Each band has a strong internal edge at its midpoint. The algorithm must
	// still find the real seam between bands, not the internal edge.
	heights := []int{100, 100, 100}
	src := noiseHoriz(200, heights)
	seams := DetectHorizSeams(src, 3)
	if len(seams) != 2 {
		t.Fatalf("want 2 seams, got %d: %v", len(seams), seams)
	}
	// Real seams at 100, 200 — internal edges at 50, 150, 250.
	for i, got := range seams {
		want := (i + 1) * 100
		if !withinTol(got, want, 10) {
			t.Errorf("seam %d: want ~%d (real seam), got %d (possibly grabbed internal edge)", i, want, got)
		}
	}
}

func TestDetectHorizSeams_SingleRow(t *testing.T) {
	src := solidImage(200, 200, color.RGBA{R: 128, A: 255})
	if seams := DetectHorizSeams(src, 1); len(seams) != 0 {
		t.Errorf("1 row: want 0 seams, got %d", len(seams))
	}
}

func TestDetectHorizSeams_TwoRows(t *testing.T) {
	src := stripedHoriz(200, []int{80, 80})
	seams := DetectHorizSeams(src, 2)
	if len(seams) != 1 {
		t.Fatalf("want 1 seam, got %d: %v", len(seams), seams)
	}
	if !withinTol(seams[0], 80, 2) {
		t.Errorf("seam: want ~80, got %d", seams[0])
	}
}

func TestDetectHorizSeams_ManyRows(t *testing.T) {
	// 6 equal rows, 50px each.
	heights := []int{50, 50, 50, 50, 50, 50}
	src := stripedHoriz(150, heights)
	seams := DetectHorizSeams(src, 6)
	if len(seams) != 5 {
		t.Fatalf("want 5 seams, got %d: %v", len(seams), seams)
	}
	for i, got := range seams {
		want := (i + 1) * 50
		if !withinTol(got, want, 3) {
			t.Errorf("seam %d: want ~%d, got %d", i, want, got)
		}
	}
}

func TestDetectHorizSeams_SeamsOrdered(t *testing.T) {
	// Seams must always be in ascending order.
	src := stripedHoriz(200, []int{60, 80, 70, 90})
	seams := DetectHorizSeams(src, 4)
	for i := 1; i < len(seams); i++ {
		if seams[i] <= seams[i-1] {
			t.Errorf("seams not ascending: %v", seams)
		}
	}
}

// ── DetectVertSeams ───────────────────────────────────────────────────────────

func TestDetectVertSeams_EqualBands(t *testing.T) {
	src := stripedVert(300, []int{100, 100, 100})
	seams := DetectVertSeams(src, 3)
	if len(seams) != 2 {
		t.Fatalf("want 2 seams, got %d: %v", len(seams), seams)
	}
	for i, got := range seams {
		want := (i + 1) * 100
		if !withinTol(got, want, 2) {
			t.Errorf("seam %d: want ~%d, got %d", i, want, got)
		}
	}
}

func TestDetectVertSeams_UnequalBands(t *testing.T) {
	src := stripedVert(200, []int{115, 125, 120})
	seams := DetectVertSeams(src, 3)
	if len(seams) != 2 {
		t.Fatalf("want 2 seams, got %d: %v", len(seams), seams)
	}
	expected := []int{115, 240}
	for i, got := range seams {
		// Allow ±8px: the constrained search window is centred on W/cols, not
		// the actual seam, so there can be a small offset for unequal bands.
		if !withinTol(got, expected[i], 8) {
			t.Errorf("seam %d: want ~%d, got %d", i, expected[i], got)
		}
	}
}

func TestDetectVertSeams_SingleCol(t *testing.T) {
	src := solidImage(200, 200, color.RGBA{G: 200, A: 255})
	if seams := DetectVertSeams(src, 1); len(seams) != 0 {
		t.Errorf("1 col: want 0 seams, got %d", len(seams))
	}
}

// ── SplitAt ───────────────────────────────────────────────────────────────────

func TestSplitAt_Basic(t *testing.T) {
	src := solidImage(400, 300, color.RGBA{R: 100, G: 150, B: 200, A: 255})
	cells, err := SplitAt(src, []int{100, 200}, []int{200})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cells) != 6 {
		t.Fatalf("want 6 cells, got %d", len(cells))
	}
}

func TestSplitAt_RowMajorOrder(t *testing.T) {
	src := solidImage(200, 300, color.RGBA{A: 255})
	cells, err := SplitAt(src, []int{100, 200}, []int{100})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for i, cell := range cells {
		wantRow, wantCol := i/2, i%2
		if cell.Row != wantRow || cell.Col != wantCol {
			t.Errorf("cells[%d]: want [%d,%d], got [%d,%d]", i, wantRow, wantCol, cell.Row, cell.Col)
		}
	}
}

func TestSplitAt_CellDimensions(t *testing.T) {
	hSeams := []int{118, 240, 361}
	vSeams := []int{200}
	src := solidImage(401, 480, color.RGBA{B: 200, A: 255})
	cells, err := SplitAt(src, hSeams, vSeams)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	wantH := []int{118, 122, 121, 119}
	wantW := []int{200, 201}
	for _, cell := range cells {
		b := cell.Image.Bounds()
		if b.Dy() != wantH[cell.Row] {
			t.Errorf("cell [%d,%d]: want height %d, got %d", cell.Row, cell.Col, wantH[cell.Row], b.Dy())
		}
		if b.Dx() != wantW[cell.Col] {
			t.Errorf("cell [%d,%d]: want width %d, got %d", cell.Row, cell.Col, wantW[cell.Col], b.Dx())
		}
	}
}

func TestSplitAt_PixelCorrectness(t *testing.T) {
	src := stripedHoriz(100, []int{50, 50})
	cells, err := SplitAt(src, []int{50}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	dark := color.RGBA{R: 20, G: 20, B: 20, A: 255}
	light := color.RGBA{R: 220, G: 220, B: 220, A: 255}
	wantColors := []color.RGBA{dark, light}
	for _, cell := range cells {
		b := cell.Image.Bounds()
		got := color.RGBAModel.Convert(cell.Image.At(b.Min.X, b.Min.Y)).(color.RGBA)
		if got != wantColors[cell.Row] {
			t.Errorf("cell row %d: want %v, got %v", cell.Row, wantColors[cell.Row], got)
		}
	}
}

func TestSplitAt_NoSeams(t *testing.T) {
	src := solidImage(100, 80, color.RGBA{R: 255, A: 255})
	cells, err := SplitAt(src, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cells) != 1 {
		t.Fatalf("want 1 cell, got %d", len(cells))
	}
	b := cells[0].Image.Bounds()
	if b.Dx() != 100 || b.Dy() != 80 {
		t.Errorf("want 100×80, got %d×%d", b.Dx(), b.Dy())
	}
}

func TestSplitAt_TotalAreaPreserved(t *testing.T) {
	src := solidImage(401, 480, color.RGBA{A: 255})
	cells, err := SplitAt(src, []int{118, 240, 361}, []int{200})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var total int
	for _, cell := range cells {
		b := cell.Image.Bounds()
		total += b.Dx() * b.Dy()
	}
	if total != 401*480 {
		t.Errorf("total area: want %d, got %d", 401*480, total)
	}
}

// ── End-to-end: detect → split ────────────────────────────────────────────────

func TestDetectAndSplit_EqualRows(t *testing.T) {
	src := stripedHoriz(400, []int{100, 100, 100, 100})
	hSeams := DetectHorizSeams(src, 4)
	cells, err := SplitAt(src, hSeams, nil)
	if err != nil {
		t.Fatalf("SplitAt: %v", err)
	}
	if len(cells) != 4 {
		t.Fatalf("want 4 cells, got %d", len(cells))
	}
	for _, cell := range cells {
		b := cell.Image.Bounds()
		if !withinTol(b.Dy(), 100, 2) {
			t.Errorf("cell row %d: want ~100px, got %d", cell.Row, b.Dy())
		}
	}
}

func TestDetectAndSplit_UnequalRows(t *testing.T) {
	heights := []int{118, 122, 121, 119}
	src := stripedHoriz(300, heights)
	hSeams := DetectHorizSeams(src, 4)
	cells, err := SplitAt(src, hSeams, nil)
	if err != nil {
		t.Fatalf("SplitAt: %v", err)
	}
	for _, cell := range cells {
		b := cell.Image.Bounds()
		if !withinTol(b.Dy(), heights[cell.Row], 3) {
			t.Errorf("cell row %d: want ~%dpx, got %d", cell.Row, heights[cell.Row], b.Dy())
		}
	}
}

func TestDetectAndSplit_WithGap(t *testing.T) {
	// Collage with 3px white gaps — each photo must NOT include gap pixels.
	white := color.RGBA{R: 255, G: 255, B: 255, A: 255}
	src := gappedHoriz(300, 100, 3, 3, white)
	hSeams := DetectHorizSeams(src, 3)
	cells, err := SplitAt(src, hSeams, nil)
	if err != nil {
		t.Fatalf("SplitAt: %v", err)
	}
	if len(cells) != 3 {
		t.Fatalf("want 3 cells, got %d", len(cells))
	}
	// Total area should equal source area.
	var total int
	for _, cell := range cells {
		b := cell.Image.Bounds()
		total += b.Dx() * b.Dy()
	}
	srcArea := src.Bounds().Dx() * src.Bounds().Dy()
	if total != srcArea {
		t.Errorf("area mismatch: cells=%d src=%d", total, srcArea)
	}
}

func TestDetectAndSplit_StrongInternalEdge(t *testing.T) {
	// Robust to internal edges within each photo band.
	heights := []int{100, 100, 100}
	src := noiseHoriz(300, heights)
	hSeams := DetectHorizSeams(src, 3)
	cells, err := SplitAt(src, hSeams, nil)
	if err != nil {
		t.Fatalf("SplitAt: %v", err)
	}
	for _, cell := range cells {
		b := cell.Image.Bounds()
		if !withinTol(b.Dy(), 100, 10) {
			t.Errorf("cell row %d: want ~100px, got %d", cell.Row, b.Dy())
		}
	}
}

func TestDetectAndSplit_2x3Grid(t *testing.T) {
	heights := []int{200, 200}
	widths := []int{150, 150, 150}
	// Build a 450×400 image with 2 rows × 3 cols.
	src := image.NewRGBA(image.Rect(0, 0, sumInts(widths), sumInts(heights)))
	colors := [2][3]color.RGBA{
		{{R: 200, A: 255}, {G: 200, A: 255}, {B: 200, A: 255}},
		{{R: 100, G: 100, A: 255}, {G: 100, B: 100, A: 255}, {R: 100, B: 100, A: 255}},
	}
	for r, h := range heights {
		for c, w := range widths {
			col := colors[r][c]
			y0 := sumInts(heights[:r])
			x0 := sumInts(widths[:c])
			for y := y0; y < y0+h; y++ {
				for x := x0; x < x0+w; x++ {
					src.SetRGBA(x, y, col)
				}
			}
		}
	}

	hSeams := DetectHorizSeams(src, 2)
	vSeams := DetectVertSeams(src, 3)
	cells, err := SplitAt(src, hSeams, vSeams)
	if err != nil {
		t.Fatalf("SplitAt: %v", err)
	}
	if len(cells) != 6 {
		t.Fatalf("want 6 cells, got %d", len(cells))
	}
	// Verify each cell's dominant color.
	for _, cell := range cells {
		b := cell.Image.Bounds()
		got := color.RGBAModel.Convert(cell.Image.At(b.Min.X+b.Dx()/2, b.Min.Y+b.Dy()/2)).(color.RGBA)
		want := colors[cell.Row][cell.Col]
		if got != want {
			t.Errorf("cell [%d,%d]: want color %v, got %v", cell.Row, cell.Col, want, got)
		}
	}
}

func abs2(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
