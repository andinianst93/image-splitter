# CLAUDE.md — image-splitter

Instructions for Claude when working in this repository.

## Project

Go CLI tool that splits a grid/collage image into individual cell files.

- **Module:** `github.com/andinianst93/image-splitter`
- **Go version:** 1.23
- **Binary name:** `image-splitter`

## Package Structure

```
main.go                          ← calls cmd.Execute()
cmd/root.go                      ← cobra command, flags, pipeline orchestration
cmd/reassemble.go                ← reassemble subcommand (stub: "not yet implemented")
internal/config/config.go        ← Config struct (no imports)
internal/imageio/reader.go       ← Load(path) → (image.Image, Format, error)
internal/imageio/writer.go       ← Save(img, SaveOptions) error
internal/splitter/splitter.go    ← Split(src, rows, cols) → ([]Cell, error)
internal/splitter/seams.go       ← DetectHorizSeams / DetectVertSeams / snapToGapCenter
internal/trimmer/trimmer.go      ← TrimBorder (iterative) / TrimBorderOnce (single-pass, capped)
internal/upscaler/upscaler.go    ← Scale(src, factor) → (image.Image, error)
```

Dependency rule: no package may import `cmd`. Packages under `internal/` must not import each other.

## Dependencies

```
github.com/spf13/cobra v1.8.1
golang.org/x/image v0.24.0
```

## Common Commands

```bash
make build       # compile binary
make test        # go test ./...
make build-all   # cross-compile mac/linux/windows
make clean       # remove binary and output/
go test ./... -v # verbose tests
go test ./... -run TestSplit  # run specific tests
```

## CLI Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--rows` | `-r` | 0 | Number of rows in the grid |
| `--cols` | `-c` | 0 | Number of columns in the grid |
| `--auto` | `-a` | false | Auto-detect seam positions; also auto-detects grid size if rows/cols omitted |
| `--trim` | `-t` | false | Remove uniform-color border pixels from each cell (single-pass, 15% depth cap) |
| `--trim-tolerance` | — | 60 | Max RGB channel diff for border color detection |
| `--output` | `-o` | `./output` | Output directory |
| `--quality` | `-q` | 0 (PNG) | JPEG quality 1–100; 0 = PNG |
| `--scale` | `-s` | 1.0 | Upscale factor per cell (CatmullRom) |

## Conventions

- Table-driven tests with `t.Run()` for all multi-case scenarios
- Use `t.TempDir()` for all temporary files in tests — never leave temp files behind
- Error messages use lowercase and wrap with `%w` for unwrapping
- No global state outside `cmd/` package
- `Quality == 0` is the sentinel value for PNG output (not an error)
- Last grid row/column absorbs remainder pixels when dimensions are not evenly divisible
- SubImage is zero-copy; fallback to `draw.Draw` for image types that don't implement it

## Trim Behaviour

Two exported functions with different semantics:

- **`TrimBorder(src, tol)`** — iterative, unlimited depth. Used as a pre-trim on the whole
  collage image before splitting. Removes thick outer borders layer by layer.
- **`TrimBorderOnce(src, tol)`** — single pass, max depth = 15% of image dimension per side.
  Used for per-cell trim after splitting. Safely removes thin separator residue (1–15px)
  without touching artistic grey mats or photo backgrounds.

Border detection rules:
1. Try 8 candidate colors (4 corners + 4 edge midpoints); accept first near-neutral (chromaSum ≤ 30)
   candidate that satisfies at least one full edge row/column.
2. Walk inward from each edge to find trim depths.
3. 15% depth cap: if any side exceeds 15% of its dimension, mark that side as no-trim.
4. Bilateral requirement: only trim an axis if BOTH sides show the border.
5. Minimum result: 10×10 px.

Known limitation: when the separator color equals the photo background color (e.g. white
separator + white photo mat), trimming may be suppressed by the bilateral check.

## Seam Detection (`--auto`)

- Energy = sum of absolute row-to-row (or col-to-col) brightness differences across all pixels
- Smoothed with a box filter, then `snapToGapCenter` shifts each seam to the center of the
  nearest uniform-color gap (separator gap)
- When rows/cols are omitted: counts the number of distinct gap regions to auto-infer the grid size

## Tested Image Inventory

| File | Size | True Grid | Separator | Recommended Command |
|------|------|-----------|-----------|---------------------|
| `image-1.png` | 1333×1999 | 3×3 | Grey ~216 | `--rows 3 --cols 3 --auto --trim` |
| `image-2.png` | 1333×1999 | 2×2 | Grey ~241 | `--rows 2 --cols 2 --auto --trim` |
| `image-3.png` | 1333×2000 | 2×3 | None (edge-to-edge) | `--rows 2 --cols 3 --auto --trim` |
| `image-4.png` | 1333×1999 | 4×4 | White 255 | `--rows 4 --cols 4 --auto --trim` |
| `sample.jpeg` | 1031×1280 | 4×2 | None (edge-to-edge) | `--rows 4 --cols 2 --auto --trim` |

## Planned Features

- **`reassemble` subcommand** (`cmd/reassemble.go`): rebuild a collage from split cells with
  optional `--order` flag for rearranging cells into a different grid layout (e.g. 4×2 → 2×4).
  Currently a stub that returns "not yet implemented".

## Do Not

- Do not add external dependencies without discussing first
- Do not commit the `output/` directory or the built binary
- Do not use `go test -count=1` by default — caching is fine for unit tests
