# image-splitter — Dokumentasi

`image-splitter` adalah CLI tool untuk memotong gambar grid/kolase menjadi sel-sel terpisah, lalu (opsional) menyusunnya kembali menjadi kolase baru dengan layout berbeda.

---

## Dokumentasi Lengkap

| Dokumen | Isi |
|---|---|
| [split.md](split.md) | Panduan lengkap command split: semua flag, contoh, pesan output, error |
| [reassemble.md](reassemble.md) | Panduan lengkap command reassemble: flag `--order`, ganti layout, contoh |
| [developer.md](developer.md) | Arsitektur, struktur package, build, test, konvensi, CI/CD |

---

## Instalasi Cepat

```bash
git clone https://github.com/andinianst93/image-splitter
cd image-splitter
make build
bin/image-splitter --help
```

---

## Cara Kerja Singkat

```
[split]       gambar kolase → dipotong → N×M file sel
[reassemble]  N×M file sel  → disusun  → 1 gambar kolase baru
```

Contoh alur lengkap:

```
┌──────────────────────────────────────────┐
│   kolase.jpg (1200 × 600 px, grid 2×3)   │
│                                          │
│   col0      col1      col2               │
│  ┌────────┬────────┬────────┐            │
│  │ [0,0]  │ [0,1]  │ [0,2]  │  row0     │
│  ├────────┼────────┼────────┤            │
│  │ [1,0]  │ [1,1]  │ [1,2]  │  row1     │
│  └────────┴────────┴────────┘            │
└──────────────────────────────────────────┘

         ↓  bin/image-splitter kolase.jpg --rows 2 --cols 3

output/
├── cell_row00_col00.png   (400×300 px)
├── cell_row00_col01.png
├── cell_row00_col02.png
├── cell_row01_col00.png
├── cell_row01_col01.png
└── cell_row01_col02.png

         ↓  bin/image-splitter reassemble --input ./output --rows 3 --cols 2

kolase-baru.png (layout berbeda: 3 baris × 2 kolom)
```

---

## Contoh Cepat

### Split

```bash
# Manual
bin/image-splitter kolase.jpg --rows 2 --cols 3

# Auto-detect grid size + seam positions
bin/image-splitter kolase.jpg --auto

# Auto-detect + hapus border seragam dari setiap sel
bin/image-splitter kolase.jpg --auto --trim

# Manual grid size + presisi seam
bin/image-splitter kolase.jpg --rows 2 --cols 3 --auto

# JPEG output
bin/image-splitter kolase.jpg --rows 2 --cols 3 --quality 85

# Upscale 2× per sel
bin/image-splitter kolase.jpg --rows 2 --cols 3 --scale 2.0
```

### Reassemble

```bash
# Rebuild layout yang sama
bin/image-splitter reassemble --input ./output

# Ganti layout 2×3 jadi 3×2
bin/image-splitter reassemble --input ./output --rows 3 --cols 2

# Balik urutan (8 sel)
bin/image-splitter reassemble --input ./output --order 7,6,5,4,3,2,1,0
```

---

## Format Gambar

| Format | Baca | Tulis |
|---|---|---|
| JPEG (`.jpg`, `.jpeg`) | ✓ | ✓ dengan `--quality 1-100` |
| PNG (`.png`) | ✓ | ✓ default, lossless |

Format input dideteksi dari **magic bytes** file, bukan ekstensi nama file.
