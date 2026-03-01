package cmd

import (
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/andinianst93/image-splitter/internal/config"
)

// ── Helpers ──────────────────────────────────────────────────────────────────

func newRGBA(w, h int, c color.RGBA) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetRGBA(x, y, c)
		}
	}
	return img
}

// tempPNG creates a w×h PNG file in a temp directory and returns its path.
func tempPNG(t *testing.T, w, h int) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "*.png")
	if err != nil {
		t.Fatalf("create temp PNG: %v", err)
	}
	defer f.Close()
	img := newRGBA(w, h, color.RGBA{R: 128, G: 64, B: 32, A: 255})
	if err := png.Encode(f, img); err != nil {
		t.Fatalf("encode PNG: %v", err)
	}
	return f.Name()
}

// tempJPEG creates a w×h JPEG file and returns its path.
func tempJPEG(t *testing.T, w, h int) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "*.jpg")
	if err != nil {
		t.Fatalf("create temp JPEG: %v", err)
	}
	defer f.Close()
	img := newRGBA(w, h, color.RGBA{R: 200, G: 100, B: 50, A: 255})
	if err := jpeg.Encode(f, img, &jpeg.Options{Quality: 85}); err != nil {
		t.Fatalf("encode JPEG: %v", err)
	}
	return f.Name()
}

// assertFileExists fails the test if path does not exist.
func assertFileExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Errorf("expected file not found: %s", path)
	}
}

// ── validate ─────────────────────────────────────────────────────────────────

func TestValidate_ValidConfigs(t *testing.T) {
	cases := []struct {
		name string
		cfg  config.Config
	}{
		{"minimal", config.Config{Rows: 1, Cols: 1, Quality: 0, Scale: 1.0}},
		{"typical PNG", config.Config{Rows: 2, Cols: 3, Quality: 0, Scale: 1.0}},
		{"JPEG quality 85", config.Config{Rows: 3, Cols: 4, Quality: 85, Scale: 1.0}},
		{"max quality", config.Config{Rows: 1, Cols: 1, Quality: 100, Scale: 1.0}},
		{"min quality", config.Config{Rows: 1, Cols: 1, Quality: 1, Scale: 1.0}},
		{"scale 2x", config.Config{Rows: 2, Cols: 2, Quality: 0, Scale: 2.0}},
		{"scale 1.5x", config.Config{Rows: 2, Cols: 2, Quality: 90, Scale: 1.5}},
		{"large grid", config.Config{Rows: 10, Cols: 10, Quality: 75, Scale: 3.0}},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			if err := validate(&tt.cfg); err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestValidate_InvalidRows(t *testing.T) {
	for _, rows := range []int{0, -1, -100} {
		cfg := config.Config{Rows: rows, Cols: 1, Quality: 0, Scale: 1.0}
		if err := validate(&cfg); err == nil {
			t.Errorf("rows=%d: expected error, got nil", rows)
		}
	}
}

func TestValidate_InvalidCols(t *testing.T) {
	for _, cols := range []int{0, -1, -50} {
		cfg := config.Config{Rows: 1, Cols: cols, Quality: 0, Scale: 1.0}
		if err := validate(&cfg); err == nil {
			t.Errorf("cols=%d: expected error, got nil", cols)
		}
	}
}

func TestValidate_InvalidQuality(t *testing.T) {
	invalids := []int{-1, -100, 101, 200, 999}
	for _, q := range invalids {
		cfg := config.Config{Rows: 1, Cols: 1, Quality: q, Scale: 1.0}
		if err := validate(&cfg); err == nil {
			t.Errorf("quality=%d: expected error, got nil", q)
		}
	}
}

func TestValidate_QualityZeroIsValid(t *testing.T) {
	// 0 is the sentinel for PNG — must not return an error.
	cfg := config.Config{Rows: 2, Cols: 2, Quality: 0, Scale: 1.0}
	if err := validate(&cfg); err != nil {
		t.Errorf("quality=0 (PNG sentinel): unexpected error: %v", err)
	}
}

func TestValidate_InvalidScale(t *testing.T) {
	for _, s := range []float64{0.0, 0.5, 0.99, -1.0, -0.001} {
		cfg := config.Config{Rows: 1, Cols: 1, Quality: 0, Scale: s}
		if err := validate(&cfg); err == nil {
			t.Errorf("scale=%.3f: expected error, got nil", s)
		}
	}
}

// ── run: happy paths ──────────────────────────────────────────────────────────

func TestRun_PNG_2x3(t *testing.T) {
	imgPath := tempPNG(t, 600, 400)
	outDir := t.TempDir()

	cfg := &config.Config{
		InputPath: imgPath, OutputDir: outDir,
		Rows: 2, Cols: 3, Quality: 0, Scale: 1.0,
	}
	if err := run(cfg); err != nil {
		t.Fatalf("run: %v", err)
	}

	for r := 0; r < 2; r++ {
		for c := 0; c < 3; c++ {
			name := fmt.Sprintf("cell_row%02d_col%02d.png", r, c)
			assertFileExists(t, filepath.Join(outDir, name))
		}
	}
}

func TestRun_JPEG_2x2(t *testing.T) {
	imgPath := tempPNG(t, 400, 400)
	outDir := t.TempDir()

	cfg := &config.Config{
		InputPath: imgPath, OutputDir: outDir,
		Rows: 2, Cols: 2, Quality: 80, Scale: 1.0,
	}
	if err := run(cfg); err != nil {
		t.Fatalf("run: %v", err)
	}

	for r := 0; r < 2; r++ {
		for c := 0; c < 2; c++ {
			name := fmt.Sprintf("cell_row%02d_col%02d.jpg", r, c)
			assertFileExists(t, filepath.Join(outDir, name))
		}
	}
}

func TestRun_WithUpscale(t *testing.T) {
	imgPath := tempPNG(t, 200, 200)
	outDir := t.TempDir()

	cfg := &config.Config{
		InputPath: imgPath, OutputDir: outDir,
		Rows: 2, Cols: 2, Quality: 90, Scale: 2.0,
	}
	if err := run(cfg); err != nil {
		t.Fatalf("run: %v", err)
	}

	// Verify one cell is doubled in size (100×100 cell × 2 = 200×200).
	f, err := os.Open(filepath.Join(outDir, "cell_row00_col00.jpg"))
	if err != nil {
		t.Fatalf("open cell: %v", err)
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	if err != nil {
		t.Fatalf("decode cell: %v", err)
	}
	if img.Bounds().Dx() != 200 || img.Bounds().Dy() != 200 {
		t.Errorf("upscaled cell: want 200×200, got %d×%d", img.Bounds().Dx(), img.Bounds().Dy())
	}
}

func TestRun_JPEG_Source(t *testing.T) {
	imgPath := tempJPEG(t, 300, 200)
	outDir := t.TempDir()

	cfg := &config.Config{
		InputPath: imgPath, OutputDir: outDir,
		Rows: 2, Cols: 3, Quality: 0, Scale: 1.0,
	}
	if err := run(cfg); err != nil {
		t.Fatalf("run with JPEG source: %v", err)
	}
	assertFileExists(t, filepath.Join(outDir, "cell_row00_col00.png"))
}

func TestRun_OutputCellDimensions(t *testing.T) {
	// 600×400 → 2×3 → each cell 200×200 (PNG exact).
	imgPath := tempPNG(t, 600, 400)
	outDir := t.TempDir()

	cfg := &config.Config{
		InputPath: imgPath, OutputDir: outDir,
		Rows: 2, Cols: 3, Quality: 0, Scale: 1.0,
	}
	if err := run(cfg); err != nil {
		t.Fatalf("run: %v", err)
	}

	for r := 0; r < 2; r++ {
		for c := 0; c < 3; c++ {
			name := fmt.Sprintf("cell_row%02d_col%02d.png", r, c)
			f, err := os.Open(filepath.Join(outDir, name))
			if err != nil {
				t.Fatalf("open %s: %v", name, err)
			}
			img, _, err := image.Decode(f)
			f.Close()
			if err != nil {
				t.Fatalf("decode %s: %v", name, err)
			}
			if img.Bounds().Dx() != 200 || img.Bounds().Dy() != 200 {
				t.Errorf("%s: want 200×200, got %d×%d", name, img.Bounds().Dx(), img.Bounds().Dy())
			}
		}
	}
}

func TestRun_CellCount(t *testing.T) {
	tests := []struct {
		rows, cols int
	}{
		{1, 1}, {2, 2}, {3, 4}, {1, 5}, {5, 1},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("%dx%d", tt.rows, tt.cols), func(t *testing.T) {
			imgPath := tempPNG(t, 500, 500)
			outDir := t.TempDir()

			cfg := &config.Config{
				InputPath: imgPath, OutputDir: outDir,
				Rows: tt.rows, Cols: tt.cols, Quality: 0, Scale: 1.0,
			}
			if err := run(cfg); err != nil {
				t.Fatalf("run: %v", err)
			}

			entries, err := os.ReadDir(outDir)
			if err != nil {
				t.Fatal(err)
			}
			if len(entries) != tt.rows*tt.cols {
				t.Errorf("want %d files, got %d", tt.rows*tt.cols, len(entries))
			}
		})
	}
}

// ── run: filename padding ─────────────────────────────────────────────────────

func TestRun_FilenamePaddingTwoDigits(t *testing.T) {
	// Default: pad to at least 2 digits for rows/cols < 10.
	imgPath := tempPNG(t, 300, 200)
	outDir := t.TempDir()

	cfg := &config.Config{
		InputPath: imgPath, OutputDir: outDir,
		Rows: 2, Cols: 3, Quality: 0, Scale: 1.0,
	}
	if err := run(cfg); err != nil {
		t.Fatalf("run: %v", err)
	}
	assertFileExists(t, filepath.Join(outDir, "cell_row00_col00.png"))
	assertFileExists(t, filepath.Join(outDir, "cell_row01_col02.png"))
}

// ── run: error paths ──────────────────────────────────────────────────────────

func TestRun_Error_InvalidConfig(t *testing.T) {
	imgPath := tempPNG(t, 100, 100)
	cases := []struct {
		name string
		cfg  config.Config
	}{
		{"rows=0", config.Config{InputPath: imgPath, OutputDir: t.TempDir(), Rows: 0, Cols: 1, Scale: 1.0}},
		{"cols=0", config.Config{InputPath: imgPath, OutputDir: t.TempDir(), Rows: 1, Cols: 0, Scale: 1.0}},
		{"quality=101", config.Config{InputPath: imgPath, OutputDir: t.TempDir(), Rows: 1, Cols: 1, Quality: 101, Scale: 1.0}},
		{"scale=0.5", config.Config{InputPath: imgPath, OutputDir: t.TempDir(), Rows: 1, Cols: 1, Scale: 0.5}},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			if err := run(&tt.cfg); err == nil {
				t.Error("expected error, got nil")
			}
		})
	}
}

func TestRun_Error_FileNotFound(t *testing.T) {
	cfg := &config.Config{
		InputPath: "/nonexistent/does_not_exist.png",
		OutputDir: t.TempDir(),
		Rows: 2, Cols: 2, Quality: 0, Scale: 1.0,
	}
	if err := run(cfg); err == nil {
		t.Error("expected error for non-existent input file, got nil")
	}
}

func TestRun_Error_RowsExceedHeight(t *testing.T) {
	// Image is 100×50, requesting 200 rows → rows > image height.
	imgPath := tempPNG(t, 100, 50)
	cfg := &config.Config{
		InputPath: imgPath, OutputDir: t.TempDir(),
		Rows: 200, Cols: 1, Quality: 0, Scale: 1.0,
	}
	if err := run(cfg); err == nil {
		t.Error("expected error for rows > image height, got nil")
	}
}

func TestRun_Error_ColsExceedWidth(t *testing.T) {
	// Image is 50×100, requesting 200 cols → cols > image width.
	imgPath := tempPNG(t, 50, 100)
	cfg := &config.Config{
		InputPath: imgPath, OutputDir: t.TempDir(),
		Rows: 1, Cols: 200, Quality: 0, Scale: 1.0,
	}
	if err := run(cfg); err == nil {
		t.Error("expected error for cols > image width, got nil")
	}
}
