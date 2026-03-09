# Plan: image-splitter CLI

## Status

- [x] Core split pipeline (load → split → save)
- [x] Auto seam detection (`--auto`)
- [x] Auto grid size detection (count separator gaps)
- [x] Border trim: pre-trim iteratif + per-cell TrimBorderOnce dengan 15% depth cap
- [x] JPEG artifact handling (edgeSkip=8, tolerance=60)
- [x] Bilateral trim requirement (mencegah over-trim di satu sisi)
- [x] Chroma check (mencegah trim warna foto yang bukan separator)
- [x] `reassemble` subcommand stub
- [x] Comprehensive test suite (semua package)
- [ ] `reassemble` subcommand fully implemented

---

## Architecture

```
main.go → cmd.Execute()
cmd/root.go → load → (pre-trim) → split → (seam snap) → per-cell trim → (upscale) → save
```

Package dependency tree (no cycles):
```
cmd      → config, imageio, splitter, trimmer, upscaler
imageio  → stdlib only
splitter → stdlib only
trimmer  → stdlib only
upscaler → golang.org/x/image/draw
config   → no imports
```

---

## Files

| File | Purpose |
|---|---|
| `main.go` | `cmd.Execute()` |
| `cmd/root.go` | Cobra command, flags, pipeline |
| `cmd/reassemble.go` | Reassemble subcommand (stub) |
| `internal/config/config.go` | Config struct |
| `internal/imageio/reader.go` | `Load(path) → (image.Image, Format, error)` |
| `internal/imageio/writer.go` | `Save(img, SaveOptions) error` |
| `internal/splitter/splitter.go` | `Split(src, rows, cols) → ([]Cell, error)` |
| `internal/splitter/seams.go` | `DetectHorizSeams`, `DetectVertSeams`, `snapToGapCenter` |
| `internal/trimmer/trimmer.go` | `TrimBorder` (iterative), `TrimBorderOnce` (single-pass + 15% cap) |
| `internal/upscaler/upscaler.go` | `Scale(src, factor) → (image.Image, error)` |

---

## Config Struct

```go
type Config struct {
    InputPath     string   // positional arg
    OutputDir     string   // --output, default "./output"
    Rows          int      // --rows (0 = auto-detect)
    Cols          int      // --cols (0 = auto-detect)
    Quality       int      // --quality 1-100; 0 = PNG
    Scale         float64  // --scale, 1.0 = no upscaling
    AutoDetect    bool     // --auto
    Trim          bool     // --trim
    TrimTolerance int      // --trim-tolerance, default 60
}
```

---

## Pipeline Detail (`cmd/root.go → run()`)

1. `validate(cfg)` — rows/cols ≥ 1 (jika tidak auto), quality 0 or 1–100, scale ≥ 1.0
2. `imageio.Load(cfg.InputPath)` → `image.Image`
3. **Pre-trim** (jika `--trim`): `trimmer.TrimBorder(img, tol)` — iteratif, hapus border luar
4. Auto grid size detection (jika `--auto` dan rows/cols = 0): hitung jumlah separator gap
5. `splitter.DetectHorizSeams` / `DetectVertSeams` (jika `--auto`) → posisi seam di tengah gap
6. `splitter.Split(img, rows, cols)` → `[]Cell`
7. Per-cell: `trimmer.TrimBorderOnce(cell, tol)` (jika `--trim`) → hapus separator residue
8. Per-cell: `upscaler.Scale(cell, factor)` (jika scale > 1.0)
9. `imageio.Save(cell, opts)` → file output

---

## Seam Detection Algorithm

```
DetectHorizSeams(src, rows):
  1. Hitung energy tiap row: sum |brightness[y] - brightness[y+1]| across all x
  2. Box filter 5px untuk smooth noise
  3. Untuk tiap seam i ∈ [1..rows-1]:
     - Expected position: H * i / rows
     - Search window: ±25% dari expected
     - Pilih row dengan energy tertinggi di window → seam candidate
  4. snapToGapCenter: geser seam ke tengah gap uniform terdekat (separator)
```

---

## Trim Algorithm

### TrimBorder (pre-trim, iteratif)
```
loop:
  result = trimBorderOnce(src, tol, maxDepthFraction=-1)  // unlimited
  if result == src: break
  src = result
```

### TrimBorderOnce (per-cell, single-pass)
```
trimBorderOnce(src, tol, maxDepthFraction=0.15)
```

### trimBorderOnce detail
```
1. detectBorderColor: coba 8 kandidat warna (4 sudut + 4 midpoint tepi)
   - Terima kandidat pertama yang: memenuhi minimal 1 edge + chromaSum ≤ 30
2. Walk top/bottom/left/right inward → hitung trim depth per sisi
3. Depth cap (jika maxDepthFraction > 0):
   - Jika trimLeft > 15% lebar → trimLeft = false
   - Idem untuk right, top, bottom
4. Bilateral requirement: trim sumbu hanya jika KEDUA sisi positif
5. Minimum result 10×10 px
```

---

## Edge Cases

| Case | Behaviour |
|---|---|
| Non-divisible dimensions | Last cell mendapat pixel lebih |
| `--quality 0` | Write PNG (zero-value sentinel) |
| `--scale 1.0` | No upscaling, no allocation |
| rows/cols > image dimensions | Error sebelum split |
| Paletted PNG input | Konversi ke RGBA sebelum JPEG encode |
| Kolase edge-to-edge (tanpa separator) | `--auto` bisa deteksi seam dari energy transition |
| Separator color = photo background | Bilateral check mencegah over-trim; separator residue mungkin tersisa |
| Corner cell di boundary gambar | 15% depth cap mencegah trim background artistik |

---

## Known Limitations

1. **Separator warna sama dengan background foto**: jika foto berlatar belakang putih dan separator juga putih, bilateral check akan mencegah trim (aman tapi residue tersisa ±14px).
2. **Auto grid size**: hanya akurat jika separator jelas dan uniform. Kolase edge-to-edge tanpa separator tidak bisa auto-detect grid size — harus specify `--rows`/`--cols` manual.
3. **`reassemble` subcommand**: belum diimplementasi, masih stub.

---

## Verified Test Results

```
image-1.png  3×3  --auto --trim  → widths 437–445px  heights 659–667px  ✓
image-2.png  2×2  --auto --trim  → widths 659–667px  heights 991–998px  ✓
image-3.png  2×3  --auto --trim  → konsisten per-kolom/baris (layout non-uniform) ✓
image-4.png  4×4  --auto --trim  → widths 321–334px  heights 485–507px  ✓
sample.jpeg  4×2  --auto --trim  → widths 515–516px  heights 318–322px  ✓
```
