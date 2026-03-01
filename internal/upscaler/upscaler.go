package upscaler

import (
	"fmt"
	"image"
	"math"

	xdraw "golang.org/x/image/draw"
)

// Scale returns a new image scaled by factor using CatmullRom resampling.
// If factor <= 1.0 the original image is returned unchanged (no allocation).
func Scale(src image.Image, factor float64) (image.Image, error) {
	if factor <= 1.0 {
		return src, nil
	}

	b := src.Bounds()
	newW := int(math.Round(float64(b.Dx()) * factor))
	newH := int(math.Round(float64(b.Dy()) * factor))

	if newW <= 0 || newH <= 0 {
		return nil, fmt.Errorf("scaled dimensions are invalid: %dx%d", newW, newH)
	}

	dst := image.NewRGBA(image.Rect(0, 0, newW, newH))
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), src, b, xdraw.Src, nil)
	return dst, nil
}
