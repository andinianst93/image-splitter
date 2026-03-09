package cmd

import (
	"fmt"
	"image"
	"image/draw"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/andinianst93/image-splitter/internal/imageio"
	"github.com/spf13/cobra"
)

var rCfg struct {
	input   string
	rows    int
	cols    int
	order   string
	output  string
	quality int
}

// cellNameRe matches filenames like cell_row02_col03.png or cell_row02_col03.jpg
var cellNameRe = regexp.MustCompile(`^cell_row(\d+)_col(\d+)\.(png|jpg|jpeg)$`)

type cellFile struct {
	row, col int
	path     string
	img      image.Image
}

var reassembleCmd = &cobra.Command{
	Use:   "reassemble",
	Short: "Reassemble split cells back into a collage",
	Long: `reassemble combines split cell images back into a single collage.

The input directory must contain files named cell_rowNN_colNN.{png,jpg}
as produced by the split command. Grid size is auto-detected from the files
unless --rows and --cols are explicitly provided.

Use --rows / --cols to change the output grid layout (e.g. swap 4x2 → 2x4).
Use --order to rearrange cells: comma-separated 0-based input indices placed
left-to-right, top-to-bottom in the new grid.

Examples:
  image-splitter reassemble --input ./output
  image-splitter reassemble --input ./output --rows 3 --cols 2
  image-splitter reassemble --input ./output --rows 2 --cols 3 --order 0,3,1,4,2,5`,
	RunE: runReassemble,
}

func init() {
	reassembleCmd.Flags().StringVar(&rCfg.input, "input", "./output", "directory containing split cell images")
	reassembleCmd.Flags().IntVar(&rCfg.rows, "rows", 0, "rows in the output collage (0 = auto from files)")
	reassembleCmd.Flags().IntVar(&rCfg.cols, "cols", 0, "columns in the output collage (0 = auto from files)")
	reassembleCmd.Flags().StringVar(&rCfg.order, "order", "", "comma-separated input cell indices for rearranging")
	reassembleCmd.Flags().StringVar(&rCfg.output, "output", "collage.png", "output collage file path")
	reassembleCmd.Flags().IntVar(&rCfg.quality, "quality", 0, "JPEG quality 1-100; 0 = PNG output")
	rootCmd.AddCommand(reassembleCmd)
}

func runReassemble(cmd *cobra.Command, args []string) error {
	// 1. Discover cell files in the input directory.
	cells, err := discoverCells(rCfg.input)
	if err != nil {
		return err
	}
	if len(cells) == 0 {
		return fmt.Errorf("no cell files found in %q", rCfg.input)
	}

	// Sort by (row, col) — canonical input order.
	sort.Slice(cells, func(i, j int) bool {
		if cells[i].row != cells[j].row {
			return cells[i].row < cells[j].row
		}
		return cells[i].col < cells[j].col
	})

	// 2. Determine output grid dimensions.
	maxRow, maxCol := 0, 0
	for _, c := range cells {
		if c.row > maxRow {
			maxRow = c.row
		}
		if c.col > maxCol {
			maxCol = c.col
		}
	}
	outRows := rCfg.rows
	outCols := rCfg.cols
	if outRows == 0 {
		outRows = maxRow + 1
	}
	if outCols == 0 {
		outCols = maxCol + 1
	}
	if outRows*outCols != len(cells) {
		return fmt.Errorf("grid %d×%d needs %d cells, found %d in %q",
			outRows, outCols, outRows*outCols, len(cells), rCfg.input)
	}

	// 3. Apply custom cell order if requested.
	ordered := make([]cellFile, len(cells))
	copy(ordered, cells)
	if rCfg.order != "" {
		ordered, err = applyOrder(cells, rCfg.order, outRows*outCols)
		if err != nil {
			return err
		}
	}

	// 4. Load all cell images.
	fmt.Printf("Reassembling %d cells from %q into %d×%d grid...\n",
		len(ordered), rCfg.input, outRows, outCols)
	for i := range ordered {
		img, _, loadErr := imageio.Load(ordered[i].path)
		if loadErr != nil {
			return fmt.Errorf("load %s: %w", ordered[i].path, loadErr)
		}
		ordered[i].img = img
	}

	// 5. Compute per-column widths and per-row heights (max of actual cell dims).
	colWidths := make([]int, outCols)
	rowHeights := make([]int, outRows)
	for r := 0; r < outRows; r++ {
		for c := 0; c < outCols; c++ {
			b := ordered[r*outCols+c].img.Bounds()
			if b.Dx() > colWidths[c] {
				colWidths[c] = b.Dx()
			}
			if b.Dy() > rowHeights[r] {
				rowHeights[r] = b.Dy()
			}
		}
	}

	// 6. Composite cells onto a single canvas.
	canvas := image.NewRGBA(image.Rect(0, 0, intSum(colWidths), intSum(rowHeights)))
	y := 0
	for r := 0; r < outRows; r++ {
		x := 0
		for c := 0; c < outCols; c++ {
			cell := ordered[r*outCols+c]
			b := cell.img.Bounds()
			draw.Draw(canvas, image.Rect(x, y, x+b.Dx(), y+b.Dy()), cell.img, b.Min, draw.Src)
			x += colWidths[c]
		}
		y += rowHeights[r]
	}

	// 7. Save the output collage.
	// Quality: use --quality flag; if 0 and output extension is .jpg/.jpeg, default to 85.
	quality := rCfg.quality
	if quality == 0 {
		outExt := strings.ToLower(filepath.Ext(rCfg.output))
		if outExt == ".jpg" || outExt == ".jpeg" {
			quality = 85
		}
	}

	outDir := filepath.Dir(rCfg.output)
	if outDir == "" {
		outDir = "."
	}
	base := filepath.Base(rCfg.output)
	filename := strings.TrimSuffix(base, filepath.Ext(base))

	opts := imageio.SaveOptions{
		OutputDir: outDir,
		Filename:  filename,
		Quality:   quality,
	}
	if err := imageio.Save(canvas, opts); err != nil {
		return fmt.Errorf("save collage: %w", err)
	}

	savedExt := ".png"
	if quality > 0 {
		savedExt = ".jpg"
	}
	fmt.Printf("Done. Collage saved to %s\n", filepath.Join(outDir, filename+savedExt))
	return nil
}

// discoverCells finds all cell_rowNN_colNN.{png,jpg} files in dir.
func discoverCells(dir string) ([]cellFile, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read directory %q: %w", dir, err)
	}
	var cells []cellFile
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		m := cellNameRe.FindStringSubmatch(e.Name())
		if m == nil {
			continue
		}
		row, _ := strconv.Atoi(m[1])
		col, _ := strconv.Atoi(m[2])
		cells = append(cells, cellFile{
			row:  row,
			col:  col,
			path: filepath.Join(dir, e.Name()),
		})
	}
	return cells, nil
}

// applyOrder rearranges cells according to a comma-separated list of 0-based
// input indices. Each index refers to the position in the sorted input list.
func applyOrder(cells []cellFile, orderStr string, n int) ([]cellFile, error) {
	parts := strings.Split(orderStr, ",")
	if len(parts) != n {
		return nil, fmt.Errorf("--order must have exactly %d indices, got %d", n, len(parts))
	}
	result := make([]cellFile, n)
	seen := make(map[int]bool, n)
	for i, p := range parts {
		idx, err := strconv.Atoi(strings.TrimSpace(p))
		if err != nil {
			return nil, fmt.Errorf("--order: invalid index %q at position %d", p, i)
		}
		if idx < 0 || idx >= n {
			return nil, fmt.Errorf("--order: index %d out of range [0, %d)", idx, n)
		}
		if seen[idx] {
			return nil, fmt.Errorf("--order: duplicate index %d", idx)
		}
		seen[idx] = true
		result[i] = cells[idx]
	}
	return result, nil
}

func intSum(s []int) int {
	n := 0
	for _, v := range s {
		n += v
	}
	return n
}
