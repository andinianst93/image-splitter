package imageio

import (
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"testing"
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

// writeTempPNG saves img to a temporary PNG file and returns its path.
func writeTempPNG(t *testing.T, img image.Image) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "*.png")
	if err != nil {
		t.Fatalf("create temp PNG: %v", err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		t.Fatalf("encode PNG: %v", err)
	}
	return f.Name()
}

// writeTempJPEG saves img to a temporary JPEG file and returns its path.
func writeTempJPEG(t *testing.T, img image.Image) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "*.jpg")
	if err != nil {
		t.Fatalf("create temp JPEG: %v", err)
	}
	defer f.Close()
	if err := jpeg.Encode(f, img, &jpeg.Options{Quality: 90}); err != nil {
		t.Fatalf("encode JPEG: %v", err)
	}
	return f.Name()
}

// ── Load: happy paths ─────────────────────────────────────────────────────────

func TestLoad_PNG(t *testing.T) {
	src := newRGBA(120, 80, color.RGBA{R: 255, A: 255})
	path := writeTempPNG(t, src)

	img, fmt, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fmt != FormatPNG {
		t.Errorf("want format %q, got %q", FormatPNG, fmt)
	}
	if img.Bounds().Dx() != 120 || img.Bounds().Dy() != 80 {
		t.Errorf("want 120×80, got %d×%d", img.Bounds().Dx(), img.Bounds().Dy())
	}
}

func TestLoad_JPEG(t *testing.T) {
	src := newRGBA(160, 90, color.RGBA{G: 200, A: 255})
	path := writeTempJPEG(t, src)

	img, fmt, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fmt != FormatJPEG {
		t.Errorf("want format %q, got %q", FormatJPEG, fmt)
	}
	if img.Bounds().Dx() != 160 || img.Bounds().Dy() != 90 {
		t.Errorf("want 160×90, got %d×%d", img.Bounds().Dx(), img.Bounds().Dy())
	}
}

// Format is detected from magic bytes, not file extension.
func TestLoad_FormatFromMagicBytes(t *testing.T) {
	src := newRGBA(10, 10, color.RGBA{B: 200, A: 255})

	// Encode as PNG but name the file .jpg — Load should still detect PNG.
	f, err := os.CreateTemp(t.TempDir(), "*.jpg")
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(f, src); err != nil {
		t.Fatal(err)
	}
	f.Close()

	_, fmt, err := Load(f.Name())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fmt != FormatPNG {
		t.Errorf("want FormatPNG detected from magic bytes, got %q", fmt)
	}
}

// ── Load: error paths ─────────────────────────────────────────────────────────

func TestLoad_NonExistentFile(t *testing.T) {
	_, _, err := Load("/nonexistent/path/image.png")
	if err == nil {
		t.Error("expected error for non-existent file, got nil")
	}
}

func TestLoad_InvalidImageData(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "*.png")
	if err != nil {
		t.Fatal(err)
	}
	f.WriteString("this is definitely not an image file")
	f.Close()

	_, _, err = Load(f.Name())
	if err == nil {
		t.Error("expected error for invalid image data, got nil")
	}
}

func TestLoad_EmptyFile(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "*.png")
	if err != nil {
		t.Fatal(err)
	}
	f.Close()

	_, _, err = Load(f.Name())
	if err == nil {
		t.Error("expected error for empty file, got nil")
	}
}

func TestLoad_TruncatedJPEG(t *testing.T) {
	path := writeTempJPEG(t, newRGBA(50, 50, color.RGBA{R: 128, A: 255}))
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	truncated := filepath.Join(t.TempDir(), "truncated.jpg")
	// Write only the first 10 bytes.
	if err := os.WriteFile(truncated, data[:10], 0o644); err != nil {
		t.Fatal(err)
	}
	_, _, err = Load(truncated)
	if err == nil {
		t.Error("expected error for truncated JPEG, got nil")
	}
}

// ── Save: happy paths ─────────────────────────────────────────────────────────

func TestSave_PNG(t *testing.T) {
	dir := t.TempDir()
	img := newRGBA(50, 50, color.RGBA{B: 200, A: 255})

	err := Save(img, SaveOptions{OutputDir: dir, Filename: "test", Quality: 0})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	path := filepath.Join(dir, "test.png")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatalf("output file not created: %s", path)
	}
	// Verify the file is a valid PNG.
	loaded, fmt, err := Load(path)
	if err != nil {
		t.Fatalf("load saved PNG: %v", err)
	}
	if fmt != FormatPNG {
		t.Errorf("want PNG format, got %q", fmt)
	}
	if loaded.Bounds().Dx() != 50 || loaded.Bounds().Dy() != 50 {
		t.Errorf("want 50×50, got %d×%d", loaded.Bounds().Dx(), loaded.Bounds().Dy())
	}
}

func TestSave_JPEG(t *testing.T) {
	dir := t.TempDir()
	img := newRGBA(80, 60, color.RGBA{R: 200, G: 100, A: 255})

	err := Save(img, SaveOptions{OutputDir: dir, Filename: "cell", Quality: 85})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	path := filepath.Join(dir, "cell.jpg")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatalf("output file not created: %s", path)
	}
	loaded, fmt, err := Load(path)
	if err != nil {
		t.Fatalf("load saved JPEG: %v", err)
	}
	if fmt != FormatJPEG {
		t.Errorf("want JPEG format, got %q", fmt)
	}
	if loaded.Bounds().Dx() != 80 || loaded.Bounds().Dy() != 60 {
		t.Errorf("want 80×60, got %d×%d", loaded.Bounds().Dx(), loaded.Bounds().Dy())
	}
}

func TestSave_CreatesNestedOutputDir(t *testing.T) {
	parent := t.TempDir()
	dir := filepath.Join(parent, "a", "b", "c")

	err := Save(newRGBA(10, 10, color.RGBA{}), SaveOptions{OutputDir: dir, Filename: "x", Quality: 0})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		t.Error("nested output directory was not created")
	}
}

// ── Save: paletted image with JPEG ───────────────────────────────────────────

func TestSave_PalettedToJPEG(t *testing.T) {
	dir := t.TempDir()
	pal := color.Palette{
		color.RGBA{R: 255, A: 255},
		color.RGBA{G: 255, A: 255},
	}
	img := image.NewPaletted(image.Rect(0, 0, 30, 30), pal)
	// All pixels index 0 (red).
	for i := range img.Pix {
		img.Pix[i] = 0
	}

	err := Save(img, SaveOptions{OutputDir: dir, Filename: "pal", Quality: 80})
	if err != nil {
		t.Fatalf("unexpected error saving paletted image as JPEG: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "pal.jpg")); os.IsNotExist(err) {
		t.Error("JPEG output file not created for paletted input")
	}
}

// toRGBA should pass through *image.RGBA unchanged (no copy).
func TestToRGBA_AlreadyRGBA(t *testing.T) {
	src := newRGBA(20, 20, color.RGBA{R: 100, G: 150, B: 200, A: 255})
	result := toRGBA(src)
	if result != src {
		t.Error("toRGBA: expected same *image.RGBA pointer for RGBA input")
	}
}

// ── Save: quality boundary values ────────────────────────────────────────────

func TestSave_QualityBoundaries(t *testing.T) {
	img := newRGBA(10, 10, color.RGBA{R: 128, A: 255})
	tests := []struct {
		quality int
		ext     string
	}{
		{0, ".png"},   // sentinel: write PNG
		{1, ".jpg"},   // minimum JPEG quality
		{100, ".jpg"}, // maximum JPEG quality
	}
	for _, tt := range tests {
		dir := t.TempDir()
		err := Save(img, SaveOptions{OutputDir: dir, Filename: "out", Quality: tt.quality})
		if err != nil {
			t.Errorf("quality=%d: unexpected error: %v", tt.quality, err)
		}
		path := filepath.Join(dir, "out"+tt.ext)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("quality=%d: expected file %s, not found", tt.quality, path)
		}
	}
}

// ── Round-trip ────────────────────────────────────────────────────────────────

func TestRoundTrip_PNG_ExactPixels(t *testing.T) {
	// PNG is lossless — pixel values must survive the round-trip.
	dir := t.TempDir()
	want := color.RGBA{R: 10, G: 200, B: 155, A: 255}
	orig := newRGBA(20, 20, want)

	if err := Save(orig, SaveOptions{OutputDir: dir, Filename: "rt", Quality: 0}); err != nil {
		t.Fatalf("save: %v", err)
	}
	loaded, _, err := Load(filepath.Join(dir, "rt.png"))
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	got := color.RGBAModel.Convert(loaded.At(0, 0)).(color.RGBA)
	if got != want {
		t.Errorf("pixel mismatch after PNG round-trip: want %v, got %v", want, got)
	}
}

func TestRoundTrip_JPEG_Dimensions(t *testing.T) {
	// JPEG is lossy (color may shift) but dimensions must be preserved.
	dir := t.TempDir()
	orig := newRGBA(100, 75, color.RGBA{R: 200, G: 100, B: 50, A: 255})

	if err := Save(orig, SaveOptions{OutputDir: dir, Filename: "rt", Quality: 90}); err != nil {
		t.Fatalf("save: %v", err)
	}
	loaded, _, err := Load(filepath.Join(dir, "rt.jpg"))
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if loaded.Bounds().Dx() != 100 || loaded.Bounds().Dy() != 75 {
		t.Errorf("JPEG round-trip dimensions: want 100×75, got %d×%d",
			loaded.Bounds().Dx(), loaded.Bounds().Dy())
	}
}

// ── Multiple cells to same directory ─────────────────────────────────────────

func TestSave_MultipleCellsSameDir(t *testing.T) {
	dir := t.TempDir()
	img := newRGBA(10, 10, color.RGBA{A: 255})

	for i := 0; i < 6; i++ {
		name := filepath.Base(filepath.Join(dir, "cell_"))
		_ = name
		opts := SaveOptions{
			OutputDir: dir,
			Filename:  filepath.Base(filepath.Join("cell_", string(rune('0'+i)))),
			Quality:   0,
		}
		// simpler: just use a direct filename
		opts.Filename = "cell_" + string(rune('0'+i))
		if err := Save(img, opts); err != nil {
			t.Errorf("cell %d: unexpected error: %v", i, err)
		}
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 6 {
		t.Errorf("want 6 files in output dir, got %d", len(entries))
	}
}
