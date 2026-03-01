package imageio

import (
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"os"
)

// Format represents the detected image format.
type Format string

const (
	FormatJPEG Format = "jpeg"
	FormatPNG  Format = "png"
)

// Load opens the file at path, decodes it, and returns the image along with
// the detected format. Supported formats: JPEG, PNG.
func Load(path string) (image.Image, Format, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, "", fmt.Errorf("open %q: %w", path, err)
	}
	defer f.Close()

	img, fmtName, err := image.Decode(f)
	if err != nil {
		return nil, "", fmt.Errorf("decode %q: %w", path, err)
	}

	switch fmtName {
	case "jpeg":
		return img, FormatJPEG, nil
	case "png":
		return img, FormatPNG, nil
	default:
		return img, Format(fmtName), nil
	}
}
