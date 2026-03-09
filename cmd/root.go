package cmd

import (
	"fmt"
	"os"

	"github.com/andinianst93/image-splitter/internal/config"
	"github.com/andinianst93/image-splitter/internal/imageio"
	"github.com/andinianst93/image-splitter/internal/splitter"
	"github.com/andinianst93/image-splitter/internal/trimmer"
	"github.com/andinianst93/image-splitter/internal/upscaler"
	"github.com/spf13/cobra"
)

var cfg config.Config

var rootCmd = &cobra.Command{
	Use:   "image-splitter <image>",
	Short: "Split a grid/collage image into individual cell files",
	Long: `image-splitter splits a grid or collage image into its individual cells.

Specify the number of rows and columns in the grid, and optionally an output
directory, JPEG quality, and upscaling factor.

Use --auto to let the tool detect the exact photo boundaries automatically.
When --rows/--cols are omitted, --auto also detects the grid size automatically.

Examples:
  image-splitter photo.jpg --rows 2 --cols 3
  image-splitter photo.jpg --auto
  image-splitter photo.jpg --rows 2 --cols 3 --auto
  image-splitter photo.jpg --rows 2 --cols 3 --quality 90 --scale 2.0 --output ./tiles`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg.InputPath = args[0]
		return run(&cfg)
	},
}

// Execute is the entry point called from main.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().StringVarP(&cfg.OutputDir, "output", "o", "./output", "directory for output cell images")
	rootCmd.Flags().IntVarP(&cfg.Rows, "rows", "r", 0, "number of rows in the grid (required)")
	rootCmd.Flags().IntVarP(&cfg.Cols, "cols", "c", 0, "number of columns in the grid (required)")
	rootCmd.Flags().IntVarP(&cfg.Quality, "quality", "q", 0, "JPEG quality 1-100; omit or 0 for PNG output")
	rootCmd.Flags().Float64VarP(&cfg.Scale, "scale", "s", 1.0, "upscale factor applied to each cell (>= 1.0)")
	rootCmd.Flags().BoolVarP(&cfg.AutoDetect, "auto", "a", false, "auto-detect exact seam positions (recommended for real collages)")
	rootCmd.Flags().BoolVarP(&cfg.Trim, "trim", "t", false, "auto-detect and remove uniform-color border pixels from each cell")
	rootCmd.Flags().IntVar(&cfg.TrimTolerance, "trim-tolerance", 60, "max RGB channel difference for border color detection (default 60 handles JPEG artifacts and color-variable borders)")
}

// validate checks all config constraints before any I/O.
func validate(cfg *config.Config) error {
	if !cfg.AutoDetect {
		if cfg.Rows < 1 {
			return fmt.Errorf("--rows must be >= 1 (or use --auto to detect grid size)")
		}
		if cfg.Cols < 1 {
			return fmt.Errorf("--cols must be >= 1 (or use --auto to detect grid size)")
		}
	}
	if cfg.Quality != 0 && (cfg.Quality < 1 || cfg.Quality > 100) {
		return fmt.Errorf("--quality must be 0 (PNG) or between 1 and 100, got %d", cfg.Quality)
	}
	if cfg.Scale < 1.0 {
		return fmt.Errorf("--scale must be >= 1.0, got %.2f", cfg.Scale)
	}
	return nil
}

// run executes the full pipeline.
func run(cfg *config.Config) error {
	if err := validate(cfg); err != nil {
		return err
	}

	// Load source image.
	img, _, err := imageio.Load(cfg.InputPath)
	if err != nil {
		return err
	}

	// Pre-trim: remove uniform-color outer border from the source image before
	// splitting. This handles collages where the dark/light border surrounds the
	// entire image. The trimmed image is then used for all subsequent steps.
	if cfg.Trim {
		img = trimmer.TrimBorder(img, cfg.TrimTolerance)
	}

	// Auto-detect grid size when rows/cols are not explicitly provided.
	if cfg.AutoDetect && (cfg.Rows == 0 || cfg.Cols == 0) {
		detectedRows, detectedCols := splitter.DetectGridSize(img)
		if cfg.Rows == 0 {
			cfg.Rows = detectedRows
		}
		if cfg.Cols == 0 {
			cfg.Cols = detectedCols
		}
		fmt.Printf("Auto-detected grid size: %d rows × %d cols\n", cfg.Rows, cfg.Cols)
	}

	b := img.Bounds()
	if cfg.Rows > b.Dy() {
		return fmt.Errorf("--rows %d exceeds image height %d", cfg.Rows, b.Dy())
	}
	if cfg.Cols > b.Dx() {
		return fmt.Errorf("--cols %d exceeds image width %d", cfg.Cols, b.Dx())
	}

	// Split image into cells.
	var cells []splitter.Cell
	if cfg.AutoDetect {
		fmt.Printf("Detecting seams in %q...\n", cfg.InputPath)
		hSeams := splitter.DetectHorizSeams(img, cfg.Rows)
		vSeams := splitter.DetectVertSeams(img, cfg.Cols)
		fmt.Printf("  horizontal seams: %v\n", hSeams)
		fmt.Printf("  vertical seams:   %v\n", vSeams)
		cells, err = splitter.SplitAt(img, hSeams, vSeams)
	} else {
		cells, err = splitter.Split(img, cfg.Rows, cfg.Cols)
	}
	if err != nil {
		return err
	}

	// Determine zero-padded format widths for the filename.
	rowPad := len(fmt.Sprintf("%d", cfg.Rows-1))
	colPad := len(fmt.Sprintf("%d", cfg.Cols-1))
	if rowPad < 2 {
		rowPad = 2
	}
	if colPad < 2 {
		colPad = 2
	}
	nameFmt := fmt.Sprintf("cell_row%%0%dd_col%%0%dd", rowPad, colPad)

	ext := "png"
	if cfg.Quality > 0 {
		ext = "jpg"
	}

	fmt.Printf("Splitting %q into %d×%d cells → %s\n", cfg.InputPath, cfg.Rows, cfg.Cols, cfg.OutputDir)

	for _, cell := range cells {
		out := cell.Image

		if cfg.Trim {
			out = trimmer.TrimBorder(out, cfg.TrimTolerance)
		}

		if cfg.Scale > 1.0 {
			out, err = upscaler.Scale(out, cfg.Scale)
			if err != nil {
				return fmt.Errorf("upscale cell [%d,%d]: %w", cell.Row, cell.Col, err)
			}
		}

		filename := fmt.Sprintf(nameFmt, cell.Row, cell.Col)
		opts := imageio.SaveOptions{
			OutputDir: cfg.OutputDir,
			Filename:  filename,
			Quality:   cfg.Quality,
		}
		if err := imageio.Save(out, opts); err != nil {
			return fmt.Errorf("save cell [%d,%d]: %w", cell.Row, cell.Col, err)
		}

		fmt.Printf("  wrote %s.%s\n", filename, ext)
	}

	fmt.Printf("Done. %d cells written to %s\n", len(cells), cfg.OutputDir)
	fmt.Println()
	fmt.Println("To rebuild a collage from these cells:")
	fmt.Printf("  ImageMagick:  montage %s/cell_*.%s -tile %dx%d -geometry +0+0 collage.%s\n",
		cfg.OutputDir, ext, cfg.Cols, cfg.Rows, ext)
	fmt.Printf("  Reassemble:   image-splitter reassemble --input %s --rows %d --cols %d\n",
		cfg.OutputDir, cfg.Rows, cfg.Cols)
	return nil
}
