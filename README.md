# Image Splitter

## Project Description

image-splitter adalah CLI tool berbasis Go untuk memotong gambar grid/kolase menjadi gambar-gambar satuan dengan kualitas tinggi, dilengkapi kemampuan deteksi seam otomatis, trim border, dan upscaling.

## Problem Statement

Kita sering bekerja dengan gambar kolase (seperti mood board, grid foto, atau komposit) yang perlu dipecah menjadi file-file individual. Proses manual menggunakan Photoshop atau tools lain memakan waktu, tidak bisa di-automate, dan sering menurunkan kualitas gambar saat export.

## Solution

CLI tool yang menerima gambar grid sebagai input, memotongnya berdasarkan jumlah baris dan kolom yang ditentukan user (atau otomatis terdeteksi), dengan output berkualitas tinggi dan opsi upscaling — semua dari terminal dalam satu command.

## Core Features

- **Grid Splitting** — Memotong gambar berdasarkan input `--rows × --cols`. Setiap cell dipotong dengan presisi pixel menggunakan SubImage (zero-copy, tidak ada degradasi kualitas saat cropping).
- **Auto Seam Detection** — Flag `--auto` mendeteksi posisi seam tepat di tengah separator gap, bukan sekadar membagi gambar secara merata. Cocok untuk kolase dengan separator putih/abu.
- **Auto Grid Size Detection** — Jika `--rows`/`--cols` tidak ditentukan, `--auto` menghitung jumlah baris dan kolom secara otomatis dari jumlah separator gap yang ditemukan.
- **Border Trim** — Flag `--trim` menghapus sisa piksel separator di tepi tiap cell setelah splitting. Menggunakan algoritma dua tahap: pre-trim iteratif pada gambar utuh, lalu single-pass trim per-cell dengan depth cap 15% untuk mencegah penghapusan background artistik.
- **High Quality Output** — Support JPEG quality 1–100 dan PNG lossless. Default output PNG.
- **Upscaling** — Opsi untuk memperbesar tiap cell menggunakan algoritma CatmullRom resampling.
- **Auto Format Detection** — Otomatis mendeteksi format input (JPEG/PNG) dari magic bytes.
- **Structured Output Naming** — File output dinamai `cell_row00_col00.ext` dengan zero-padding.

## Usage

```bash
# Split 3×3 grid dengan seam detection dan trim otomatis
image-splitter image.png --rows 3 --cols 3 --auto --trim

# Auto-detect jumlah grid sekaligus (tidak perlu --rows/--cols)
image-splitter image.png --auto --trim

# Edge-to-edge kolase (tanpa separator) — tetap butuh --rows/--cols
image-splitter photo.jpeg --rows 4 --cols 2 --auto --trim

# JPEG output + upscaling 2x
image-splitter image.png --rows 3 --cols 3 --quality 90 --scale 2.0

# Custom output dir
image-splitter image.png --rows 3 --cols 3 --output ./hasil
```

## Flags

| Flag | Short | Default | Keterangan |
|------|-------|---------|------------|
| `--rows` | `-r` | — | Jumlah baris grid |
| `--cols` | `-c` | — | Jumlah kolom grid |
| `--auto` | `-a` | false | Deteksi seam otomatis; juga auto-deteksi grid size jika rows/cols tidak ditentukan |
| `--trim` | `-t` | false | Hapus sisa piksel separator di tepi tiap cell |
| `--trim-tolerance` | — | 60 | Toleransi deteksi warna border (max channel diff RGB) |
| `--output` | `-o` | `./output` | Direktori output |
| `--quality` | `-q` | 0 (PNG) | JPEG quality 1–100; 0 = PNG |
| `--scale` | `-s` | 1.0 | Faktor upscale per cell (CatmullRom) |

## Hasil Test pada Gambar Nyata

| Gambar | Ukuran | Grid | Separator | Akurasi Split |
|--------|--------|------|-----------|---------------|
| `image-1.png` | 1333×1999 | 3×3 | Abu ~216 | ✓ widths 437–445px, heights 659–667px |
| `image-2.png` | 1333×1999 | 2×2 | Abu ~241 | ✓ widths 659–667px, heights 991–998px |
| `image-3.png` | 1333×2000 | 2×3 | Tidak ada (edge-to-edge) | ✓ Konsisten per-kolom/baris |
| `image-4.png` | 1333×1999 | 4×4 | Putih 255 | ✓ widths 321–334px, heights 485–507px |
| `sample.jpeg` | 1031×1280 | 4×2 | Tidak ada (edge-to-edge) | ✓ widths 515–516px, heights 318–322px |

> **Catatan:** Variasi beberapa pixel antar cell adalah normal karena ketebalan separator yang sedikit berbeda. Variasi besar (>20px) tidak boleh terjadi pada gambar yang grid-nya ditentukan benar.

## Technical Highlights

- Dibangun dengan Go standard library + `golang.org/x/image` untuk resampling
- Zero-copy cropping via `SubImage()` — kualitas crop tidak turun sama sekali
- Degradasi kualitas hanya terjadi satu kali saat encode ke file output
- Seam detection: energy profile + box filter + snap-to-gap-center
- Trim: deteksi warna corner, bilateral check, 15% depth cap, iterative untuk pre-trim
- Separation of concerns yang ketat: splitter, trimmer, upscaler, dan imageio adalah package terpisah
- CLI menggunakan Cobra untuk UX yang clean dan extensible

## Project Structure

```
image-splitter/
├── main.go                    # Entry point
├── Makefile                   # build, test, build-all, clean
├── cmd/
│   ├── root.go                # Command utama & flags (cobra)
│   └── reassemble.go          # Subcommand reassemble (stub)
└── internal/
    ├── config/
    │   └── config.go          # Struct Config dari flags CLI
    ├── imageio/
    │   ├── reader.go          # Decode gambar (jpeg/png auto-detect)
    │   └── writer.go          # Encode & save ke file
    ├── splitter/
    │   ├── splitter.go        # Core split logic (SubImage zero-copy)
    │   └── seams.go           # DetectHorizSeams / DetectVertSeams
    ├── trimmer/
    │   └── trimmer.go         # TrimBorder / TrimBorderOnce
    └── upscaler/
        └── upscaler.go        # Scale (CatmullRom)
```

## Build

```bash
make build       # compile binary → bin/image-splitter
make test        # go test ./...
make build-all   # cross-compile linux/mac/windows
make clean       # hapus binary dan output/
```
