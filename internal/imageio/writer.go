package imageio

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"hash/crc32"
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
	XPPU      uint32 // PNG pHYs pixels-per-unit X (meters); 0 = omit pHYs chunk
	YPPU      uint32 // PNG pHYs pixels-per-unit Y (meters); 0 = omit pHYs chunk
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
		var buf bytes.Buffer
		if err := png.Encode(&buf, img); err != nil {
			return fmt.Errorf("encode PNG %q: %w", fullPath, err)
		}
		data := buf.Bytes()
		if opts.XPPU > 0 {
			data = insertPHYs(data, opts.XPPU, opts.YPPU)
		}
		if _, err := f.Write(data); err != nil {
			return fmt.Errorf("write PNG %q: %w", fullPath, err)
		}
	}

	return nil
}

// insertPHYs inserts a pHYs chunk into a PNG byte slice immediately after the
// IHDR chunk (offset 33). This preserves the DPI metadata from the source image
// in each output cell, since Go's png.Encode does not write pHYs.
func insertPHYs(data []byte, xppu, yppu uint32) []byte {
	// PNG layout: 8-byte signature + IHDR chunk (4+4+13+4 = 25 bytes) = 33 bytes.
	const insertAt = 33

	// pHYs chunk: length(4) + type(4) + data(9) + CRC(4) = 21 bytes.
	var chunk [21]byte
	binary.BigEndian.PutUint32(chunk[0:4], 9)
	copy(chunk[4:8], "pHYs")
	binary.BigEndian.PutUint32(chunk[8:12], xppu)
	binary.BigEndian.PutUint32(chunk[12:16], yppu)
	chunk[16] = 1 // unit = meter
	crc := crc32.ChecksumIEEE(chunk[4:17])
	binary.BigEndian.PutUint32(chunk[17:21], crc)

	result := make([]byte, len(data)+21)
	copy(result, data[:insertAt])
	copy(result[insertAt:], chunk[:])
	copy(result[insertAt+21:], data[insertAt:])
	return result
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
