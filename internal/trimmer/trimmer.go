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

// TrimBorder detects and removes a uniform-color border from src, iterating
// until no further border is found. Use this for whole-image pre-trim where
// multiple distinct border layers may need to be removed.
//
// Algorithm per iteration:
//  1. Try candidate border colors from the 4 corners and the 4 edge midpoints.
//     The first candidate whose color is shared by at least one full edge row or
//     column (within tolerance) becomes the border color.
//  2. If no candidate produces a uniform edge → stop iterating and return current src.
//  3. Walk inward from each edge to find the first row/col that is NOT
//     entirely the border color.
//  4. If the resulting crop would be smaller than 10×10 → return src unchanged.
//  5. Return the cropped image (zero-copy via SubImage when possible).
func TrimBorder(src image.Image, tolerance int) image.Image {
	for {
		prev := src
		src = trimBorderOnce(src, tolerance, -1)
		if src == prev {
			return src
		}
	}
}

// TrimBorderOnce applies a single pass of border detection and removal.
// Use this for per-cell trim after seam-based splitting: it removes only the
// thin separator residue at cell edges without iterating into the artistic
// background that becomes the new edge after the first pass.
//
// It also caps the trim depth per side at 15% of the image dimension.
// This prevents wide artistic backgrounds (grey mats, photo content at image
// boundaries) from being mistaken for separator borders.
func TrimBorderOnce(src image.Image, tolerance int) image.Image {
	return trimBorderOnce(src, tolerance, 0.15)
}

// trimBorderOnce performs one pass of border detection and removal.
// maxDepthFraction caps the trim depth per side as a fraction of image
// dimension; pass a negative value for no cap (used by TrimBorder).
func trimBorderOnce(src image.Image, tolerance int, maxDepthFraction float64) image.Image {
	b := src.Bounds()
	if b.Dx() < 1 || b.Dy() < 1 {
		return src
	}

	// edgeSkip excludes this many pixels at each end of a row/col when checking
	// uniformity. JPEG seam-boundary artifacts (compression block effects from the
	// adjacent cell) can corrupt the very last pixels of an edge, which would
	// otherwise cause the entire edge to fail the border test.
	// edgeSkip = 8 matches the JPEG 8×8 DCT block size: compression artifacts
	// from the adjacent cell can corrupt up to 8 pixels from a seam boundary.
	const edgeSkip = 8

	// Inner x-range used when checking rows (skip left/right corners).
	ix0, ix1 := b.Min.X+edgeSkip, b.Max.X-edgeSkip
	if ix0 >= ix1 {
		ix0, ix1 = b.Min.X, b.Max.X
	}
	// Inner y-range used when checking cols (skip top/bottom corners).
	// Recomputed after top/bottom are determined.

	border, ok := detectBorderColor(src, b, tolerance, edgeSkip)
	if !ok {
		return src
	}
	// Reject tinted border colors. Real collage separators (grey, white, black)
	// are near-neutral (R ≈ G ≈ B). Warm or cool photo content at cell edges
	// has a color cast; treating it as a border would crop actual photo content.
	if chromaSum(border) > 30 {
		return src
	}

	top := b.Min.Y
	for top < b.Max.Y && rowIsBorder(src, top, ix0, ix1, border, tolerance) {
		top++
	}

	bottom := b.Max.Y
	for bottom > top && rowIsBorder(src, bottom-1, ix0, ix1, border, tolerance) {
		bottom--
	}

	iy0, iy1 := top+edgeSkip, bottom-edgeSkip
	if iy0 >= iy1 {
		iy0, iy1 = top, bottom
	}

	left := b.Min.X
	for left < b.Max.X && colIsBorder(src, left, iy0, iy1, border, tolerance) {
		left++
	}

	right := b.Max.X
	for right > left && colIsBorder(src, right-1, iy0, iy1, border, tolerance) {
		right--
	}

	// Require borders on BOTH sides of each axis before trimming.
	// A uniform edge on only one side is likely actual photo content (e.g. a
	// dark background), not a frame. Real separators/frames appear on both sides.
	trimTop := top > b.Min.Y
	trimBottom := bottom < b.Max.Y
	trimLeft := left > b.Min.X
	trimRight := right < b.Max.X

	// Per-cell depth cap: if any trim side exceeds maxDepthFraction of the
	// image dimension, treat it as artistic content (grey mat, background at
	// image boundary) rather than a separator border, and skip that direction.
	if maxDepthFraction > 0 {
		maxW := int(float64(b.Dx()) * maxDepthFraction)
		maxH := int(float64(b.Dy()) * maxDepthFraction)
		if left-b.Min.X > maxW {
			trimLeft = false
		}
		if b.Max.X-right > maxW {
			trimRight = false
		}
		if top-b.Min.Y > maxH {
			trimTop = false
		}
		if b.Max.Y-bottom > maxH {
			trimBottom = false
		}
	}

	newTop, newBottom := b.Min.Y, b.Max.Y
	newLeft, newRight := b.Min.X, b.Max.X
	if trimTop && trimBottom {
		newTop, newBottom = top, bottom
	}
	if trimLeft && trimRight {
		newLeft, newRight = left, right
	}

	w := newRight - newLeft
	h := newBottom - newTop
	if w < 10 || h < 10 {
		return src
	}

	crop := image.Rect(newLeft, newTop, newRight, newBottom)
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

// detectBorderColor finds a border color by trying the 4 corner pixels and the
// 4 edge-midpoint pixels as candidates. Returns the first candidate for which at
// least one edge (top row, bottom row, left col, right col) consists entirely of
// pixels within tolerance of that candidate.
//
// edgeSkip pixels are excluded at each end of every row/col check to avoid
// seam-boundary JPEG artifacts at cell corners causing false negatives.
func detectBorderColor(src image.Image, b image.Rectangle, tolerance, edgeSkip int) (color.RGBA, bool) {
	midX := b.Min.X + b.Dx()/2
	midY := b.Min.Y + b.Dy()/2
	candidates := []color.RGBA{
		toRGBA(src.At(b.Min.X, b.Min.Y)),
		toRGBA(src.At(b.Max.X-1, b.Min.Y)),
		toRGBA(src.At(b.Min.X, b.Max.Y-1)),
		toRGBA(src.At(b.Max.X-1, b.Max.Y-1)),
		toRGBA(src.At(midX, b.Min.Y)),
		toRGBA(src.At(midX, b.Max.Y-1)),
		toRGBA(src.At(b.Min.X, midY)),
		toRGBA(src.At(b.Max.X-1, midY)),
	}
	// Inner ranges: skip edgeSkip pixels at each corner to avoid seam artifacts.
	rx0, rx1 := b.Min.X+edgeSkip, b.Max.X-edgeSkip
	if rx0 >= rx1 {
		rx0, rx1 = b.Min.X, b.Max.X
	}
	ry0, ry1 := b.Min.Y+edgeSkip, b.Max.Y-edgeSkip
	if ry0 >= ry1 {
		ry0, ry1 = b.Min.Y, b.Max.Y
	}
	for _, c := range candidates {
		if rowIsBorder(src, b.Min.Y, rx0, rx1, c, tolerance) ||
			rowIsBorder(src, b.Max.Y-1, rx0, rx1, c, tolerance) ||
			colIsBorder(src, b.Min.X, ry0, ry1, c, tolerance) ||
			colIsBorder(src, b.Max.X-1, ry0, ry1, c, tolerance) {
			return c, true
		}
	}
	return color.RGBA{}, false
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

// chromaSum returns |R-G| + |G-B| + |R-B| — a measure of how "tinted" a color
// is away from neutral grey. Pure grey/white/black returns 0.
func chromaSum(c color.RGBA) int {
	rg := int(c.R) - int(c.G)
	if rg < 0 {
		rg = -rg
	}
	gb := int(c.G) - int(c.B)
	if gb < 0 {
		gb = -gb
	}
	rb := int(c.R) - int(c.B)
	if rb < 0 {
		rb = -rb
	}
	return rg + gb + rb
}
