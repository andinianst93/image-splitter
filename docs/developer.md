# image-splitter — Panduan Developer

Dokumentasi arsitektur, build, test, dan konvensi kode.

---

## Daftar Isi

- [Instalasi & Build](#instalasi--build)
- [Struktur Package](#struktur-package)
- [Arsitektur Pipeline](#arsitektur-pipeline)
- [Package: internal/splitter](#package-internalsplitter)
- [Package: internal/trimmer](#package-internaltrimmer)
- [Package: internal/upscaler](#package-internalupscaler)
- [Package: internal/imageio](#package-internalimageio)
- [Package: cmd](#package-cmd)
- [Menjalankan Test](#menjalankan-test)
- [Konvensi Kode](#konvensi-kode)
- [CI/CD](#cicd)

---

## Instalasi & Build

```bash
git clone https://github.com/andinianst93/image-splitter
cd image-splitter
make build
```

Binary tersimpan di `bin/image-splitter`. Jalankan dengan:

```bash
bin/image-splitter --help
```

### Cross-compile semua platform

```bash
make build-all
```

Menghasilkan binary di folder `dist/`:

```
dist/image-splitter-darwin-arm64      ← macOS Apple Silicon
dist/image-splitter-darwin-amd64      ← macOS Intel
dist/image-splitter-linux-amd64       ← Linux
dist/image-splitter-windows-amd64.exe ← Windows
```

### Perintah Make

```bash
make build       # compile ke bin/image-splitter
make test        # go test ./...
make build-all   # cross-compile semua platform
make clean       # hapus bin/ dan output/
```

### Struktur folder build

```
bin/    ← hasil make build  (untuk pakai lokal, masuk .gitignore)
dist/   ← hasil make build-all (untuk distribusi, masuk .gitignore)
```

---

## Struktur Package

```
main.go                        ← calls cmd.Execute()
cmd/root.go                    ← cobra command, flags, pipeline orchestration (split)
cmd/reassemble.go              ← cobra subcommand reassemble
internal/config/config.go      ← Config struct (no imports)
internal/imageio/reader.go     ← Load(path) → (image.Image, Format, error)
internal/imageio/writer.go     ← Save(img, SaveOptions) error
internal/splitter/splitter.go  ← Split(src, rows, cols) → ([]Cell, error)
internal/splitter/seams.go     ← DetectHorizSeams, DetectVertSeams, DetectGridSize
internal/trimmer/trimmer.go    ← TrimBorder(src, tolerance) → image.Image
internal/upscaler/upscaler.go  ← Scale(src, factor) → (image.Image, error)
```

**Dependency rule:** tidak ada package yang boleh import `cmd`. Package di bawah `internal/` tidak boleh saling import satu sama lain.

---

## Arsitektur Pipeline

### Split pipeline

```
cmd/root.go: run()
  │
  ├─ validate(cfg)
  ├─ imageio.Load(path)                → image.Image
  ├─ trimmer.TrimBorder(img, tol)      → image.Image  [jika --trim: pre-trim sumber]
  ├─ splitter.DetectGridSize(img)      → rows, cols   [jika --auto tanpa rows/cols]
  ├─ splitter.DetectHorizSeams(img, n) → []int        [jika --auto]
  ├─ splitter.DetectVertSeams(img, n)  → []int        [jika --auto]
  ├─ splitter.SplitAt(img, hSeams, vSeams) → []Cell   [jika --auto]
  │  atau splitter.Split(img, rows, cols) → []Cell    [jika manual]
  │
  └─ for each cell:
       ├─ trimmer.TrimBorder(cell, tol)   [jika --trim: post-trim per sel]
       ├─ upscaler.Scale(cell, factor)    [jika --scale > 1.0]
       └─ imageio.Save(cell, opts)
```

**Pre-trim vs post-trim:**
- Pre-trim: menghapus border/background luar dari seluruh gambar sumber sebelum split. Ini menyelesaikan kasus utama (kolase dengan outer border). Seam detection kemudian berjalan pada gambar yang sudah bersih.
- Post-trim: membersihkan sisa gap/border dari setiap sel setelah split. Berguna jika ada gap antar foto di dalam kolase (bukan hanya outer border).

### Reassemble pipeline

```
cmd/reassemble.go: runReassemble()
  │
  ├─ discoverCells(dir)           → []cellFile
  ├─ sort by (row, col)
  ├─ applyOrder(cells, order, n)  [jika --order]
  ├─ imageio.Load(each cell)      → []image.Image
  ├─ compute canvas size          (per-col widths, per-row heights)
  ├─ draw.Draw onto canvas
  └─ imageio.Save(canvas, opts)
```

---

## Package: internal/splitter

### splitter.go

```go
type Cell struct {
    Image image.Image
    Row   int
    Col   int
}

// Split membagi src menjadi rows×cols sel dengan ukuran rata.
// Sel terakhir menyerap sisa piksel jika dimensi tidak habis dibagi.
func Split(src image.Image, rows, cols int) ([]Cell, error)

// SplitAt membagi src di posisi seam yang sudah dideteksi.
func SplitAt(src image.Image, hSeams, vSeams []int) ([]Cell, error)
```

**Zero-copy:** `Split` dan `SplitAt` menggunakan `SubImage()` — tidak ada alokasi memori baru. Jika image type tidak mendukung `SubImage`, fallback ke `draw.Draw`.

### seams.go

```go
// DetectHorizSeams mendeteksi n-1 seam horizontal (batas baris) di src.
func DetectHorizSeams(src image.Image, n int) []int

// DetectVertSeams mendeteksi n-1 seam vertikal (batas kolom) di src.
func DetectVertSeams(src image.Image, n int) []int

// DetectGridSize mendeteksi jumlah baris dan kolom secara otomatis.
func DetectGridSize(src image.Image) (rows, cols int)
```

**Algoritma auto-detect:**
1. Hitung **energy profile** — rata-rata selisih absolut warna antar baris/kolom berurutan
2. Haluskan dengan **box filter** radius 7 untuk mengurangi noise
3. Cari puncak energi menggunakan `findSignificantPeaks`:
   - Urutkan nilai energi
   - Cari **gap terbesar** di 25% nilai tertinggi
   - Threshold = nilai tepat di batas bawah gap → semua puncak di atas threshold diterima
4. Jumlah puncak + 1 = jumlah baris/kolom

---

## Package: internal/trimmer

```go
// TrimBorder mendeteksi dan menghapus border berwarna seragam dari src.
// tolerance = max perbedaan channel RGB yang masih dianggap "sama warna border"
// Default dari cmd: 40 (menangani JPEG compression artifacts)
// Dikontrol via flag --trim-tolerance
func TrimBorder(src image.Image, tolerance int) image.Image
```

**Algoritma:**
1. Sampel 4 sudut (`tl`, `tr`, `bl`, `br`)
2. Jika max-channel-diff antar sudut mana pun > tolerance → return src tanpa perubahan
3. Gunakan `tl` sebagai warna border
4. Walk dari tepi atas → bawah → kiri → kanan, cari baris/kolom pertama yang bukan border
5. Jika hasil crop < 10×10 px → return src tanpa perubahan
6. Crop menggunakan `SubImage()` (zero-copy) jika image type mendukung; fallback ke `draw.Draw`

**Alpha diabaikan** dalam perbandingan warna border — hanya RGB yang dibandingkan.

---

## Package: internal/upscaler

```go
// Scale memperbesar src sebesar factor menggunakan CatmullRom resampling.
// factor harus >= 1.0.
func Scale(src image.Image, factor float64) (image.Image, error)
```

Menggunakan `golang.org/x/image/draw` dengan kernel `draw.CatmullRom` — salah satu algoritma upscaling berkualitas tinggi yang menjaga ketajaman tepi.

---

## Package: internal/imageio

```go
// Load membaca file gambar dari path.
// Format dideteksi dari magic bytes, bukan ekstensi.
// Mengembalikan image.Image, string format ("jpeg"/"png"), dan error.
func Load(path string) (image.Image, string, error)

type SaveOptions struct {
    OutputDir string
    Filename  string  // tanpa ekstensi
    Quality   int     // 0 = PNG, 1-100 = JPEG
}

// Save menyimpan img ke file.
// Membuat OutputDir jika belum ada.
func Save(img image.Image, opts SaveOptions) error
```

**PNG paletted:** PNG dengan mode indexed color otomatis dikonversi ke RGBA sebelum disimpan sebagai JPEG.

---

## Package: cmd

### root.go (split command)

- Mendaftarkan semua flag split
- `validate()`: cek constraint sebelum I/O
- `run()`: pipeline split lengkap
- Mencetak pesan sukses + instruksi rebuild

### reassemble.go

- Subcommand `reassemble`
- `discoverCells(dir)`: cari file `cell_rowNN_colNN.{png,jpg}` via regex
- `applyOrder(cells, orderStr, n)`: reorder sel sesuai `--order`
- `runReassemble()`: pipeline reassemble lengkap

---

## Menjalankan Test

```bash
make test
# atau
go test ./...
```

Verbose:

```bash
go test ./... -v
```

Per package:

```bash
go test ./internal/splitter/... -v
go test ./internal/trimmer/... -v
go test ./cmd/... -v -run TestReassemble
go test ./internal/splitter/... -v -run TestDetectGridSize
```

Test spesifik:

```bash
go test ./internal/splitter/... -run TestSplit_PixelCorrectness
go test ./cmd/... -run TestRun_PNG_2x3
go test ./internal/trimmer/... -run TestTrimBorder_AsymmetricBorder
```

**Cakupan test:**

| Package | Test yang ada |
|---|---|
| `internal/splitter` | Split, SplitAt, DetectGridSize, horizEnergy/vertEnergy |
| `internal/trimmer` | TrimBorder (10 kasus: solid border, no border, corners disagree, tiny result, tolerance, SubImage path, fallback path, non-zero bounds, asymmetric, zero size) |
| `internal/imageio` | Load (JPEG, PNG, not-found, invalid), Save (PNG, JPEG, mkdir) |
| `internal/upscaler` | Scale (2×, 3×, factor=1, invalid) |
| `cmd` | run (PNG 2×3, JPEG, auto, trim, scale, invalid args), reassemble (same layout, swap layout, custom order, error cases) |

---

## Konvensi Kode

- **Table-driven tests** dengan `t.Run()` untuk semua skenario multi-kasus
- `t.TempDir()` untuk semua file sementara di test — tidak meninggalkan file temp
- **Error messages:** lowercase, wrap dengan `%w` untuk unwrapping
- Tidak ada global state di luar package `cmd/`
- `Quality == 0` adalah sentinel untuk PNG output (bukan error)
- **Sel terakhir** menyerap sisa piksel saat dimensi tidak habis dibagi
- **SubImage** adalah zero-copy; fallback ke `draw.Draw` untuk image type yang tidak implement `SubImage`

---

## CI/CD

### CI (`.github/workflows/ci.yml`)

Trigger: push/PR ke branch `master`

Steps:
1. Checkout kode
2. Setup Go (versi dari `go.mod`)
3. `go test ./...`

### Release (`.github/workflows/release.yml`)

Trigger: push tag `v*` (mis. `git tag v1.0.0 && git push --tags`)

Steps:
1. Checkout kode
2. Setup Go
3. `go test ./...`
4. Build semua platform:
   - `darwin/arm64`, `darwin/amd64`, `linux/amd64`, `windows/amd64`
5. Upload binary ke GitHub Release (dengan release notes otomatis)

### Cara release versi baru

```bash
git tag v1.2.3
git push origin v1.2.3
```

GitHub Actions akan otomatis build dan publish release.

### Dependencies

```
github.com/spf13/cobra v1.8.1     ← CLI framework
golang.org/x/image v0.24.0        ← CatmullRom resampling
```

Jangan tambah dependency baru tanpa diskusi terlebih dahulu.
