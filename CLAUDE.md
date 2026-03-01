# CLAUDE.md — image-splitter

Instructions for Claude when working in this repository.

## Project

Go CLI tool that splits a grid/collage image into individual cell files.

- **Module:** `github.com/andinianst93/image-splitter`
- **Go version:** 1.23
- **Binary name:** `image-splitter`

## Package Structure

```
main.go                        ← calls cmd.Execute()
cmd/root.go                    ← cobra command, flags, pipeline orchestration
internal/config/config.go      ← Config struct (no imports)
internal/imageio/reader.go     ← Load(path) → (image.Image, Format, error)
internal/imageio/writer.go     ← Save(img, SaveOptions) error
internal/splitter/splitter.go  ← Split(src, rows, cols) → ([]Cell, error)
internal/upscaler/upscaler.go  ← Scale(src, factor) → (image.Image, error)
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

## Conventions

- Table-driven tests with `t.Run()` for all multi-case scenarios
- Use `t.TempDir()` for all temporary files in tests — never leave temp files behind
- Error messages use lowercase and wrap with `%w` for unwrapping
- No global state outside `cmd/` package
- `Quality == 0` is the sentinel value for PNG output (not an error)
- Last grid row/column absorbs remainder pixels when dimensions are not evenly divisible
- SubImage is zero-copy; fallback to `draw.Draw` for image types that don't implement it

## Do Not

- Do not add external dependencies without discussing first
- Do not commit the `output/` directory or the built binary
- Do not use `go test -count=1` by default — caching is fine for unit tests
