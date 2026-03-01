package imageio

import (
	"fmt"
	"image"
	"image/draw"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
)

// SaveOptions controls how an image cell is written to disk.
type SaveOptions struct {
	OutputDir string // destination directory (will be created if missing)
	Filename  string // file name without extension
	Quality   int    // JPEG quality 1-100; 0 → PNG
}

// Save writes img to disk according to opts.
// Quality > 0 → JPEG; Quality == 0 → PNG.
func Save(img image.Image, opts SaveOptions) error {
	if err := os.MkdirAll(opts.OutputDir, 0o755); err != nil {
		return fmt.Errorf("create output dir %q: %w", opts.OutputDir, err)
	}

	var ext string
	if opts.Quality > 0 {
		ext = ".jpg"
	} else {
		ext = ".png"
	}

	fullPath := filepath.Join(opts.OutputDir, opts.Filename+ext)
	f, err := os.Create(fullPath)
	if err != nil {
		return fmt.Errorf("create file %q: %w", fullPath, err)
	}
	defer f.Close()

	if opts.Quality > 0 {
		// JPEG cannot encode paletted images — convert to RGBA first.
		enc := toRGBA(img)
		if err := jpeg.Encode(f, enc, &jpeg.Options{Quality: opts.Quality}); err != nil {
			return fmt.Errorf("encode JPEG %q: %w", fullPath, err)
		}
	} else {
		if err := png.Encode(f, img); err != nil {
			return fmt.Errorf("encode PNG %q: %w", fullPath, err)
		}
	}

	return nil
}

// toRGBA returns img as *image.RGBA, converting if necessary.
func toRGBA(img image.Image) *image.RGBA {
	if r, ok := img.(*image.RGBA); ok {
		return r
	}
	b := img.Bounds()
	dst := image.NewRGBA(b)
	draw.Draw(dst, b, img, b.Min, draw.Src)
	return dst
}
