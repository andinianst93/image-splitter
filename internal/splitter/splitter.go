package splitter

import (
	"fmt"
	"image"
	"image/draw"
)

// Cell holds a sub-image and its grid coordinates.
type Cell struct {
	Image image.Image
	Row   int
	Col   int
}

// subImager is satisfied by the standard library image types that support
// zero-copy sub-images (e.g. *image.RGBA, *image.NRGBA, *image.YCbCr).
type subImager interface {
	SubImage(r image.Rectangle) image.Image
}

// Split divides src into a rows×cols grid using equal-width columns and
// equal-height rows. The last column / row absorbs any remainder pixels when
// the image dimensions are not evenly divisible.
func Split(src image.Image, rows, cols int) ([]Cell, error) {
	if rows < 1 || cols < 1 {
		return nil, fmt.Errorf("rows and cols must be >= 1, got rows=%d cols=%d", rows, cols)
	}

	b := src.Bounds()
	totalW := b.Dx()
	totalH := b.Dy()

	cellW := totalW / cols
	cellH := totalH / rows

	if cellW == 0 || cellH == 0 {
		return nil, fmt.Errorf("grid too large: image %dx%d cannot be split into %d×%d cells", totalW, totalH, rows, cols)
	}

	si, canSubImage := src.(subImager)

	cells := make([]Cell, 0, rows*cols)

	for r := 0; r < rows; r++ {
		for c := 0; c < cols; c++ {
			x0 := b.Min.X + c*cellW
			y0 := b.Min.Y + r*cellH

			// Last column / row gets the remaining pixels.
			var x1, y1 int
			if c == cols-1 {
				x1 = b.Max.X
			} else {
				x1 = x0 + cellW
			}
			if r == rows-1 {
				y1 = b.Max.Y
			} else {
				y1 = y0 + cellH
			}

			rect := image.Rect(x0, y0, x1, y1)

			var cellImg image.Image
			if canSubImage {
				cellImg = si.SubImage(rect)
			} else {
				// Fallback: pixel copy for image types that don't implement SubImage.
				dst := image.NewRGBA(image.Rect(0, 0, rect.Dx(), rect.Dy()))
				draw.Draw(dst, dst.Bounds(), src, rect.Min, draw.Src)
				cellImg = dst
			}

			cells = append(cells, Cell{Image: cellImg, Row: r, Col: c})
		}
	}

	return cells, nil
}

// SplitAt divides src using explicit seam coordinates.
// hSeams contains (rows-1) absolute y-coordinates of horizontal boundaries.
// vSeams contains (cols-1) absolute x-coordinates of vertical boundaries.
// Both slices must be sorted ascending. Cells are returned in row-major order.
func SplitAt(src image.Image, hSeams, vSeams []int) ([]Cell, error) {
	b := src.Bounds()

	// Build row boundaries: [Min.Y, hSeams..., Max.Y]
	yBounds := make([]int, 0, len(hSeams)+2)
	yBounds = append(yBounds, b.Min.Y)
	yBounds = append(yBounds, hSeams...)
	yBounds = append(yBounds, b.Max.Y)

	// Build col boundaries: [Min.X, vSeams..., Max.X]
	xBounds := make([]int, 0, len(vSeams)+2)
	xBounds = append(xBounds, b.Min.X)
	xBounds = append(xBounds, vSeams...)
	xBounds = append(xBounds, b.Max.X)

	rows := len(yBounds) - 1
	cols := len(xBounds) - 1

	if rows < 1 || cols < 1 {
		return nil, fmt.Errorf("SplitAt: invalid seam configuration")
	}

	si, canSubImage := src.(subImager)
	cells := make([]Cell, 0, rows*cols)

	for r := 0; r < rows; r++ {
		for c := 0; c < cols; c++ {
			rect := image.Rect(xBounds[c], yBounds[r], xBounds[c+1], yBounds[r+1])

			var cellImg image.Image
			if canSubImage {
				cellImg = si.SubImage(rect)
			} else {
				dst := image.NewRGBA(image.Rect(0, 0, rect.Dx(), rect.Dy()))
				draw.Draw(dst, dst.Bounds(), src, rect.Min, draw.Src)
				cellImg = dst
			}

			cells = append(cells, Cell{Image: cellImg, Row: r, Col: c})
		}
	}

	return cells, nil
}
