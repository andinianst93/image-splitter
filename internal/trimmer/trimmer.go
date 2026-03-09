package trimmer

import (
	"image"
	"image/color"
	"image/draw"
)

// subImager is satisfied by standard library image types that support zero-copy
// sub-images (*image.RGBA, *image.NRGBA, *image.YCbCr, etc.).
type subImager interface {
	SubImage(r image.Rectangle) image.Image
}

// TrimBorder detects and removes a uniform-color border from src.
//
// Algorithm:
//  1. Sample the 4 corner pixels. If all 4 are within tolerance of each other
//     (max RGB channel diff), the top-left corner is used as the border color.
//  2. If corners disagree → return src unchanged (can't determine border color).
//  3. Walk inward from each edge to find the first row/col that is NOT
//     entirely the border color.
//  4. If the resulting crop would be smaller than 10×10 → return src unchanged.
//  5. Return the cropped image (zero-copy via SubImage when possible).
func TrimBorder(src image.Image, tolerance int) image.Image {
	b := src.Bounds()
	if b.Dx() < 1 || b.Dy() < 1 {
		return src
	}

	// Sample all 4 corners.
	tl := toRGBA(src.At(b.Min.X, b.Min.Y))
	tr := toRGBA(src.At(b.Max.X-1, b.Min.Y))
	bl := toRGBA(src.At(b.Min.X, b.Max.Y-1))
	br := toRGBA(src.At(b.Max.X-1, b.Max.Y-1))

	// All corners must agree within tolerance.
	if colorDiff(tl, tr) > tolerance ||
		colorDiff(tl, bl) > tolerance ||
		colorDiff(tl, br) > tolerance {
		return src
	}

	border := tl

	top := b.Min.Y
	for top < b.Max.Y && rowIsBorder(src, top, b.Min.X, b.Max.X, border, tolerance) {
		top++
	}

	bottom := b.Max.Y
	for bottom > top && rowIsBorder(src, bottom-1, b.Min.X, b.Max.X, border, tolerance) {
		bottom--
	}

	left := b.Min.X
	for left < b.Max.X && colIsBorder(src, left, top, bottom, border, tolerance) {
		left++
	}

	right := b.Max.X
	for right > left && colIsBorder(src, right-1, top, bottom, border, tolerance) {
		right--
	}

	w := right - left
	h := bottom - top
	if w < 10 || h < 10 {
		return src
	}

	crop := image.Rect(left, top, right, bottom)
	if crop == b {
		return src
	}

	if si, ok := src.(subImager); ok {
		return si.SubImage(crop)
	}

	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(dst, dst.Bounds(), src, crop.Min, draw.Src)
	return dst
}

func toRGBA(c color.Color) color.RGBA {
	return color.RGBAModel.Convert(c).(color.RGBA)
}

// colorDiff returns the maximum absolute difference across R, G, B channels.
// Alpha is intentionally ignored.
func colorDiff(a, b color.RGBA) int {
	dr := int(a.R) - int(b.R)
	dg := int(a.G) - int(b.G)
	db := int(a.B) - int(b.B)
	if dr < 0 {
		dr = -dr
	}
	if dg < 0 {
		dg = -dg
	}
	if db < 0 {
		db = -db
	}
	m := dr
	if dg > m {
		m = dg
	}
	if db > m {
		m = db
	}
	return m
}

func rowIsBorder(src image.Image, y, x0, x1 int, border color.RGBA, tolerance int) bool {
	for x := x0; x < x1; x++ {
		if colorDiff(toRGBA(src.At(x, y)), border) > tolerance {
			return false
		}
	}
	return true
}

func colIsBorder(src image.Image, x, y0, y1 int, border color.RGBA, tolerance int) bool {
	for y := y0; y < y1; y++ {
		if colorDiff(toRGBA(src.At(x, y)), border) > tolerance {
			return false
		}
	}
	return true
}
