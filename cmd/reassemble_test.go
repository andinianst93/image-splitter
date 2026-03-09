package cmd

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

// writeCellPNG creates a solid-color PNG cell file in dir with the given row/col.
func writeCellPNG(t *testing.T, dir string, row, col, w, h int, c color.RGBA) string {
	t.Helper()
	name := fmt.Sprintf("cell_row%02d_col%02d.png", row, col)
	path := filepath.Join(dir, name)
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create cell: %v", err)
	}
	defer f.Close()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetRGBA(x, y, c)
		}
	}
	if err := png.Encode(f, img); err != nil {
		t.Fatalf("encode cell: %v", err)
	}
	return path
}

// make2x3CellDir creates a directory with a 2-row × 3-col set of cell PNG files.
// Each cell is 100×80 and filled with a distinct color.
func make2x3CellDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	colors := [2][3]color.RGBA{
		{{R: 255, A: 255}, {G: 255, A: 255}, {B: 255, A: 255}},
		{{R: 128, A: 255}, {G: 128, A: 255}, {B: 128, A: 255}},
	}
	for r := range colors {
		for c, col := range colors[r] {
			writeCellPNG(t, dir, r, c, 100, 80, col)
		}
	}
	return dir
}

// ── discoverCells ─────────────────────────────────────────────────────────────

func TestDiscoverCells_FindsAll(t *testing.T) {
	dir := make2x3CellDir(t)
	cells, err := discoverCells(dir)
	if err != nil {
		t.Fatalf("discoverCells: %v", err)
	}
	if len(cells) != 6 {
		t.Errorf("want 6 cells, got %d", len(cells))
	}
}

func TestDiscoverCells_IgnoresNonCellFiles(t *testing.T) {
	dir := t.TempDir()
	writeCellPNG(t, dir, 0, 0, 10, 10, color.RGBA{A: 255})
	// Create a non-cell file that should be ignored.
	if err := os.WriteFile(filepath.Join(dir, "README.txt"), []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}
	cells, err := discoverCells(dir)
	if err != nil {
		t.Fatalf("discoverCells: %v", err)
	}
	if len(cells) != 1 {
		t.Errorf("want 1 cell, got %d", len(cells))
	}
}

func TestDiscoverCells_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	cells, err := discoverCells(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cells) != 0 {
		t.Errorf("want 0 cells, got %d", len(cells))
	}
}

func TestDiscoverCells_MissingDir(t *testing.T) {
	_, err := discoverCells("/nonexistent/path")
	if err == nil {
		t.Error("expected error for missing directory, got nil")
	}
}

// ── applyOrder ────────────────────────────────────────────────────────────────

func TestApplyOrder_Reverses(t *testing.T) {
	cells := []cellFile{{row: 0, col: 0}, {row: 0, col: 1}, {row: 1, col: 0}, {row: 1, col: 1}}
	result, err := applyOrder(cells, "3,2,1,0", 4)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for i, want := range []cellFile{cells[3], cells[2], cells[1], cells[0]} {
		if result[i] != want {
			t.Errorf("result[%d]: want %+v, got %+v", i, want, result[i])
		}
	}
}

func TestApplyOrder_WrongCount(t *testing.T) {
	cells := []cellFile{{}, {}, {}}
	if _, err := applyOrder(cells, "0,1", 3); err == nil {
		t.Error("expected error for wrong index count, got nil")
	}
}

func TestApplyOrder_OutOfRange(t *testing.T) {
	cells := []cellFile{{}, {}}
	if _, err := applyOrder(cells, "0,5", 2); err == nil {
		t.Error("expected error for out-of-range index, got nil")
	}
}

func TestApplyOrder_Duplicate(t *testing.T) {
	cells := []cellFile{{}, {}, {}}
	if _, err := applyOrder(cells, "0,0,1", 3); err == nil {
		t.Error("expected error for duplicate index, got nil")
	}
}

// ── runReassemble integration ─────────────────────────────────────────────────

func TestReassemble_SameLayout(t *testing.T) {
	inputDir := make2x3CellDir(t)
	outDir := t.TempDir()
	outPath := filepath.Join(outDir, "collage.png")

	rCfg.input = inputDir
	rCfg.rows = 0
	rCfg.cols = 0
	rCfg.order = ""
	rCfg.output = outPath
	rCfg.quality = 0

	if err := runReassemble(reassembleCmd, nil); err != nil {
		t.Fatalf("runReassemble: %v", err)
	}

	// Output should exist and be 300×160 (3 cols×100 + 2 rows×80).
	f, err := os.Open(outPath)
	if err != nil {
		t.Fatalf("open collage: %v", err)
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	if err != nil {
		t.Fatalf("decode collage: %v", err)
	}
	b := img.Bounds()
	if b.Dx() != 300 || b.Dy() != 160 {
		t.Errorf("want 300×160, got %d×%d", b.Dx(), b.Dy())
	}
}

func TestReassemble_SwapLayout(t *testing.T) {
	// Swap 2×3 → 3×2 layout: 6 cells, new grid 3 rows × 2 cols.
	inputDir := make2x3CellDir(t)
	outDir := t.TempDir()
	outPath := filepath.Join(outDir, "collage.png")

	rCfg.input = inputDir
	rCfg.rows = 3
	rCfg.cols = 2
	rCfg.order = ""
	rCfg.output = outPath
	rCfg.quality = 0

	if err := runReassemble(reassembleCmd, nil); err != nil {
		t.Fatalf("runReassemble: %v", err)
	}

	f, err := os.Open(outPath)
	if err != nil {
		t.Fatalf("open collage: %v", err)
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	if err != nil {
		t.Fatalf("decode collage: %v", err)
	}
	b := img.Bounds()
	// 3 rows×80 = 240 height, 2 cols×100 = 200 width.
	if b.Dx() != 200 || b.Dy() != 240 {
		t.Errorf("want 200×240, got %d×%d", b.Dx(), b.Dy())
	}
}

func TestReassemble_CustomOrder(t *testing.T) {
	inputDir := make2x3CellDir(t)
	outDir := t.TempDir()
	outPath := filepath.Join(outDir, "collage.png")

	rCfg.input = inputDir
	rCfg.rows = 2
	rCfg.cols = 3
	rCfg.order = "5,4,3,2,1,0" // reverse order
	rCfg.output = outPath
	rCfg.quality = 0

	if err := runReassemble(reassembleCmd, nil); err != nil {
		t.Fatalf("runReassemble: %v", err)
	}
	// Dimensions should be same as normal 2×3.
	f, err := os.Open(outPath)
	if err != nil {
		t.Fatalf("open collage: %v", err)
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	if err != nil {
		t.Fatalf("decode collage: %v", err)
	}
	b := img.Bounds()
	if b.Dx() != 300 || b.Dy() != 160 {
		t.Errorf("want 300×160, got %d×%d", b.Dx(), b.Dy())
	}
}

func TestReassemble_Error_NoCells(t *testing.T) {
	rCfg.input = t.TempDir()
	rCfg.rows, rCfg.cols, rCfg.order, rCfg.quality = 0, 0, "", 0
	rCfg.output = filepath.Join(t.TempDir(), "out.png")
	if err := runReassemble(reassembleCmd, nil); err == nil {
		t.Error("expected error for empty directory, got nil")
	}
}

func TestReassemble_Error_WrongGridSize(t *testing.T) {
	inputDir := make2x3CellDir(t) // 6 cells
	rCfg.input = inputDir
	rCfg.rows = 2
	rCfg.cols = 2 // 2×2=4 ≠ 6
	rCfg.order, rCfg.quality = "", 0
	rCfg.output = filepath.Join(t.TempDir(), "out.png")
	if err := runReassemble(reassembleCmd, nil); err == nil {
		t.Error("expected error for mismatched grid size, got nil")
	}
}
