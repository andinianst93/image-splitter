package splitter

import (
	"image"
	"math"
	"sort"
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

	raw := make([]float64, H)
	for y := 1; y < H; y++ {
		raw[y] = horizEnergy(src, b.Min.Y+y)
	}
	smoothed := boxSmooth(raw, 7)

	seams := make([]int, rows-1)
	tol := tolerancePx(H, rows)
	for i := 0; i < rows-1; i++ {
		expected := (i + 1) * H / rows
		approx := bestInWindow(smoothed, expected, tol)
		seams[i] = b.Min.Y + snapToGapCenter(raw, approx, tol)
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

	raw := make([]float64, W)
	for x := 1; x < W; x++ {
		raw[x] = vertEnergy(src, b.Min.X+x)
	}
	smoothed := boxSmooth(raw, 7)

	seams := make([]int, cols-1)
	tol := tolerancePx(W, cols)
	for i := 0; i < cols-1; i++ {
		expected := (i + 1) * W / cols
		approx := bestInWindow(smoothed, expected, tol)
		seams[i] = b.Min.X + snapToGapCenter(raw, approx, tol)
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

// snapToGapCenter tries to move the seam from a high-energy transition point
// to the center of the solid-color gap that it borders. It scans forward then
// backward from pos looking for a bounded low-energy run (rows with energy
// < 15% of the peak) that is terminated on both sides by high energy.
// This "bounded" requirement prevents the algorithm from snapping into the
// interior of a photo band (which also has low energy but extends much further).
// If no such gap is found within maxScan rows, pos is returned unchanged.
func snapToGapCenter(raw []float64, pos, maxScan int) int {
	n := len(raw)
	peakE := raw[pos]
	if peakE < 5 {
		return pos // already inside a gap or trivial image
	}
	gapThresh := peakE * 0.15

	// Forward scan: gap lies just after this transition (photo→gap direction).
	{
		runStart := -1
		for i := pos + 1; i <= pos+maxScan && i < n; i++ {
			if raw[i] <= gapThresh {
				if runStart < 0 {
					runStart = i
				}
			} else if runStart >= 0 {
				// High energy found after the low run → bounded gap confirmed.
				if i-runStart >= 3 {
					return (runStart + i - 1) / 2
				}
				break
			}
		}
	}

	// Backward scan: gap lies just before this transition (gap→photo direction).
	{
		runEnd := -1
		for i := pos - 1; i >= pos-maxScan && i >= 0; i-- {
			if raw[i] <= gapThresh {
				if runEnd < 0 {
					runEnd = i
				}
			} else if runEnd >= 0 {
				// High energy found before the low run → bounded gap confirmed.
				if runEnd-i >= 3 {
					return (i + 1 + runEnd) / 2
				}
				break
			}
		}
	}

	return pos
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

// DetectGridSize automatically determines the number of rows and columns in a
// grid/collage image by finding significant high-energy seam lines.
// Returns at least (1, 1) for any input.
func DetectGridSize(src image.Image) (rows, cols int) {
	b := src.Bounds()
	H := b.Dy()
	W := b.Dx()

	hEnergy := make([]float64, H)
	for y := 1; y < H; y++ {
		hEnergy[y] = horizEnergy(src, b.Min.Y+y)
	}
	hMinDist := H / 10
	if hMinDist < 20 {
		hMinDist = 20
	}
	hPeaks := findSignificantPeaks(boxSmooth(hEnergy, 7), hMinDist)

	vEnergy := make([]float64, W)
	for x := 1; x < W; x++ {
		vEnergy[x] = vertEnergy(src, b.Min.X+x)
	}
	vMinDist := W / 10
	if vMinDist < 20 {
		vMinDist = 20
	}
	vPeaks := findSignificantPeaks(boxSmooth(vEnergy, 7), vMinDist)

	return len(hPeaks) + 1, len(vPeaks) + 1
}

// findSignificantPeaks finds peaks in a smoothed energy profile that likely
// represent grid/collage seam boundaries. minDist is the minimum pixel
// separation required between two distinct peaks.
func findSignificantPeaks(energy []float64, minDist int) []int {
	n := len(energy)
	if n < 3 {
		return nil
	}

	// Sort a copy to compute a dynamic threshold.
	sorted := make([]float64, n)
	copy(sorted, energy)
	sort.Float64s(sorted)

	// Find the largest gap in the top 25% of energy values.
	// The lower bound of that gap separates seam-level energy from photo content.
	startIdx := n * 3 / 4
	threshold := sorted[n*9/10] // fallback: 90th percentile
	maxGap := -1.0
	for i := startIdx; i < n-1; i++ {
		if gap := sorted[i+1] - sorted[i]; gap > maxGap {
			maxGap = gap
			threshold = sorted[i] // lower boundary: everything strictly above is seam-level
		}
	}

	// Collect local maxima strictly above the threshold.
	var candidates []int
	for i := 1; i < n-1; i++ {
		if energy[i] <= threshold {
			continue
		}
		if energy[i] >= energy[i-1] && energy[i] >= energy[i+1] {
			candidates = append(candidates, i)
		}
	}

	// Enforce minimum distance: when two candidates are too close, keep the
	// one with higher energy.
	var peaks []int
	for _, idx := range candidates {
		if len(peaks) == 0 || idx-peaks[len(peaks)-1] >= minDist {
			peaks = append(peaks, idx)
		} else if energy[idx] > energy[peaks[len(peaks)-1]] {
			peaks[len(peaks)-1] = idx
		}
	}

	return peaks
}
