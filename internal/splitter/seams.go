package splitter

import (
	"image"
	"math"
)

// DetectHorizSeams returns the y-coordinates of the (rows-1) best horizontal
// seam positions that divide src into `rows` bands.
//
// Strategy: "constrained local maximum"
//   - For each expected seam position (H*i/rows), search within a ±25% window
//     for the highest-energy row. This ignores strong internal photo edges that
//     are far from expected positions, making it robust across image types.
//   - Energy is smoothed with a box filter to suppress pixel-level noise.
func DetectHorizSeams(src image.Image, rows int) []int {
	if rows <= 1 {
		return nil
	}
	b := src.Bounds()
	H := b.Dy()

	energy := make([]float64, H)
	for y := 1; y < H; y++ {
		energy[y] = horizEnergy(src, b.Min.Y+y)
	}
	smoothed := boxSmooth(energy, 7)

	seams := make([]int, rows-1)
	tol := tolerancePx(H, rows)
	for i := 0; i < rows-1; i++ {
		expected := (i + 1) * H / rows
		seams[i] = b.Min.Y + bestInWindow(smoothed, expected, tol)
	}
	return seams
}

// DetectVertSeams returns the x-coordinates of the (cols-1) best vertical
// seam positions that divide src into `cols` bands.
func DetectVertSeams(src image.Image, cols int) []int {
	if cols <= 1 {
		return nil
	}
	b := src.Bounds()
	W := b.Dx()

	energy := make([]float64, W)
	for x := 1; x < W; x++ {
		energy[x] = vertEnergy(src, b.Min.X+x)
	}
	smoothed := boxSmooth(energy, 7)

	seams := make([]int, cols-1)
	tol := tolerancePx(W, cols)
	for i := 0; i < cols-1; i++ {
		expected := (i + 1) * W / cols
		seams[i] = b.Min.X + bestInWindow(smoothed, expected, tol)
	}
	return seams
}

// horizEnergy returns the average per-channel absolute difference between
// row y and the row above it.
func horizEnergy(src image.Image, y int) float64 {
	b := src.Bounds()
	var sum float64
	for x := b.Min.X; x < b.Max.X; x++ {
		r0, g0, b0, _ := src.At(x, y-1).RGBA()
		r1, g1, b1, _ := src.At(x, y).RGBA()
		sum += pixelDiff(r0, g0, b0, r1, g1, b1)
	}
	return sum / float64(b.Dx())
}

// vertEnergy returns the average per-channel absolute difference between
// col x and the col to its left.
func vertEnergy(src image.Image, x int) float64 {
	b := src.Bounds()
	var sum float64
	for y := b.Min.Y; y < b.Max.Y; y++ {
		r0, g0, b0, _ := src.At(x-1, y).RGBA()
		r1, g1, b1, _ := src.At(x, y).RGBA()
		sum += pixelDiff(r0, g0, b0, r1, g1, b1)
	}
	return sum / float64(b.Dy())
}

func pixelDiff(r0, g0, b0, r1, g1, b1 uint32) float64 {
	dr := math.Abs(float64(r0>>8) - float64(r1>>8))
	dg := math.Abs(float64(g0>>8) - float64(g1>>8))
	db := math.Abs(float64(b0>>8) - float64(b1>>8))
	return (dr + dg + db) / 3.0
}

// boxSmooth applies a box filter of the given radius to reduce noise.
func boxSmooth(energy []float64, radius int) []float64 {
	n := len(energy)
	out := make([]float64, n)
	for i := range energy {
		lo := i - radius
		if lo < 0 {
			lo = 0
		}
		hi := i + radius
		if hi >= n {
			hi = n - 1
		}
		var sum float64
		for j := lo; j <= hi; j++ {
			sum += energy[j]
		}
		out[i] = sum / float64(hi-lo+1)
	}
	return out
}

// tolerancePx returns the search window half-width for each seam.
// Set to 25% of the expected cell size, clamped to a minimum of 10px.
func tolerancePx(total, n int) int {
	t := total / n / 4
	if t < 10 {
		t = 10
	}
	return t
}

// bestInWindow finds the index with the highest energy value in
// [expected-tol, expected+tol], clamped to valid array bounds.
// Falls back to `expected` if all values in the window are zero.
func bestInWindow(energy []float64, expected, tol int) int {
	lo := expected - tol
	if lo < 0 {
		lo = 0
	}
	hi := expected + tol
	if hi >= len(energy) {
		hi = len(energy) - 1
	}

	best, bestE := expected, energy[expected]
	for i := lo; i <= hi; i++ {
		if energy[i] > bestE {
			bestE = energy[i]
			best = i
		}
	}
	return best
}

func iabs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
