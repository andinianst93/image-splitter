# image-splitter — User Guide

`image-splitter` adalah CLI tool untuk memotong gambar grid/kolase menjadi sel-sel terpisah. Cocok untuk memisahkan Instagram carousel, sprite sheet, atau kolase foto menjadi file individual.

---

## Daftar Isi

- [Instalasi](#instalasi)
- [Cara Kerja](#cara-kerja)
- [Penggunaan Dasar](#penggunaan-dasar)
- [Semua Flag](#semua-flag)
- [Contoh Penggunaan](#contoh-penggunaan)
- [Format Output](#format-output)
- [Format yang Didukung](#format-yang-didukung)
- [Catatan & Edge Case](#catatan--edge-case)

---

## Instalasi

### Build dari source

```bash
git clone https://github.com/andinianst93/image-splitter
cd image-splitter
make build
```

Binary `image-splitter` akan muncul di direktori saat ini.

### Cross-compile (semua platform sekaligus)

```bash
make build-all
```

Menghasilkan binary di folder `dist/`:

```
dist/image-splitter-darwin-arm64
dist/image-splitter-darwin-amd64
dist/image-splitter-linux-amd64
dist/image-splitter-windows-amd64.exe
```

---

## Cara Kerja

Tool ini membagi gambar sumber menjadi grid `rows × cols`. Setiap sel disimpan sebagai file terpisah di direktori output.

```
┌─────────────────────────────────────┐
│  Gambar asli (mis. 1200 × 600 px)  │
│                                     │
│  col0      col1      col2           │
│ ┌────────┬────────┬────────┐        │
│ │ [0,0]  │ [0,1]  │ [0,2]  │ row0  │
│ ├────────┼────────┼────────┤        │
│ │ [1,0]  │ [1,1]  │ [1,2]  │ row1  │
│ └────────┴────────┴────────┘        │
└─────────────────────────────────────┘

Hasil: 6 file → cell_row00_col00.png … cell_row01_col02.png
```

---

## Penggunaan Dasar

```bash
image-splitter <path-gambar> --rows <N> --cols <N> [opsi lain]
```

`--rows` dan `--cols` **wajib** diisi. Semua flag lainnya opsional.

### Contoh tercepat

```bash
# Split sample-img.jpeg menjadi grid 2 baris × 3 kolom, output PNG
./image-splitter sample-img.jpeg --rows 2 --cols 3
```

Output akan masuk ke folder `./output/` secara default.

---

## Semua Flag

| Flag | Shorthand | Default | Keterangan |
|---|---|---|---|
| `--rows` | `-r` | *(wajib)* | Jumlah baris dalam grid |
| `--cols` | `-c` | *(wajib)* | Jumlah kolom dalam grid |
| `--output` | `-o` | `./output` | Direktori tempat menyimpan hasil |
| `--quality` | `-q` | `0` | Kualitas JPEG 1–100; `0` = output PNG |
| `--scale` | `-s` | `1.0` | Faktor upscale tiap sel (≥ 1.0) |
| `--auto` | `-a` | `false` | Auto-detect posisi seam (direkomendasikan untuk foto kolase nyata) |

### Aturan `--quality`

| Nilai | Perilaku |
|---|---|
| `0` (default) | Simpan sebagai **PNG** (lossless) |
| `1` – `100` | Simpan sebagai **JPEG** dengan kualitas tersebut |

### Aturan `--scale`

- `1.0` (default) → tidak ada perubahan ukuran
- `2.0` → tiap sel di-upscale 2× menggunakan CatmullRom resampling
- Nilai di bawah `1.0` akan ditolak (tidak mendukung downscale)

---

## Contoh Penggunaan

### 1. Split dasar ke PNG

```bash
./image-splitter sample-img.jpeg --rows 2 --cols 3
```

---

### 2. Split kolase nyata dengan auto-detect seam

Gunakan `--auto` jika foto kolase kamu dibuat oleh aplikasi (Instagram, Canva, dll) karena tinggi/lebar tiap sel bisa sedikit tidak sama persis.

```bash
./image-splitter foto-kolase.jpg --rows 4 --cols 2 --auto
```

Output saat `--auto` aktif:
```
Detecting seams in "foto-kolase.jpg"...
  horizontal seams: [318 640 961]
  vertical seams:   [515]
Splitting "foto-kolase.jpg" into 4×2 cells → ./output
  wrote cell_row00_col00.png
  ...
```

Tanpa `--auto`, tool membagi rata (`H/rows`, `W/cols`). Dengan `--auto`, tool mencari batas antar foto yang sebenarnya menggunakan analisis energi piksel.

```
Splitting "sample-img.jpeg" into 2×3 cells → ./output
  wrote cell_row00_col00.png
  wrote cell_row00_col01.png
  wrote cell_row00_col02.png
  wrote cell_row01_col00.png
  wrote cell_row01_col01.png
  wrote cell_row01_col02.png
Done. 6 cells written to ./output
```

---

### 3. Output JPEG dengan kualitas tertentu

```bash
./image-splitter sample-img.jpeg --rows 2 --cols 3 --quality 85
```

Menghasilkan `cell_row00_col00.jpg` … `cell_row01_col02.jpg`.

---

### 4. Output ke direktori kustom

```bash
./image-splitter sample-img.jpeg --rows 2 --cols 3 --output ./hasil
```

Direktori `./hasil/` dibuat otomatis jika belum ada.

---

### 5. Upscale tiap sel 2×

```bash
./image-splitter sample-img.jpeg --rows 2 --cols 3 --scale 2.0
```

Gambar asli 1200×600 → tiap sel 400×300 → setelah upscale menjadi **800×600**.

---

### 6. JPEG + upscale sekaligus

```bash
./image-splitter sample-img.jpeg --rows 2 --cols 3 --quality 90 --scale 2.0 --output ./tiles
```

---

### 7. Grid 1 kolom (potong vertikal)

```bash
./image-splitter sample-img.jpeg --rows 4 --cols 1
```

Membagi gambar menjadi 4 strip horizontal.

---

### 8. Grid 1 baris (potong horizontal)

```bash
./image-splitter sample-img.jpeg --rows 1 --cols 4
```

Membagi gambar menjadi 4 strip vertikal.

---

### 9. Menggunakan shorthand flag

```bash
./image-splitter sample-img.jpeg -r 2 -c 3 -q 80 -s 1.5 -o ./out
```

---

## Format Output

### Penamaan file

```
cell_row{R}_col{C}.{ext}
```

- `{R}` dan `{C}` adalah nomor baris/kolom dengan zero-padding minimal 2 digit
- Padding melebar otomatis untuk grid besar (mis. grid 100×100 → `cell_row00_col00` s/d `cell_row99_col99`)

### Contoh nama file untuk grid 2×3

```
output/
├── cell_row00_col00.png   ← baris 0, kolom 0 (kiri atas)
├── cell_row00_col01.png   ← baris 0, kolom 1
├── cell_row00_col02.png   ← baris 0, kolom 2 (kanan atas)
├── cell_row01_col00.png   ← baris 1, kolom 0 (kiri bawah)
├── cell_row01_col01.png
└── cell_row01_col02.png   ← baris 1, kolom 2 (kanan bawah)
```

---

## Format yang Didukung

| Format | Baca | Tulis |
|---|---|---|
| JPEG (`.jpg`, `.jpeg`) | ✓ | ✓ (dengan `--quality`) |
| PNG (`.png`) | ✓ | ✓ (default) |

Format input dideteksi dari **magic bytes** file, bukan ekstensi nama file.

---

## Catatan & Edge Case

### Dimensi tidak habis dibagi

Jika ukuran gambar tidak habis dibagi jumlah baris/kolom, **sel terakhir** di setiap baris/kolom mendapat sisa piksel.

Contoh: gambar 601×401 dibagi 2×3:
- Kolom 0 & 1 → lebar 200 px; Kolom 2 → lebar **201 px**
- Baris 0 → tinggi 200 px; Baris 1 → tinggi **201 px**

### Batas jumlah baris/kolom

```bash
# Error: --rows melebihi tinggi gambar
./image-splitter sample-img.jpeg --rows 700 --cols 1
# → Error: --rows 700 exceeds image height 600

# Error: --cols melebihi lebar gambar
./image-splitter sample-img.jpeg --rows 1 --cols 1300
# → Error: --cols 1300 exceeds image width 1200
```

### Input PNG paletted

PNG dengan mode warna paletted (indexed color) otomatis dikonversi ke RGBA sebelum disimpan sebagai JPEG.

---

## Menjalankan Test

```bash
make test
# atau
go test ./...
```

Untuk verbose:

```bash
go test ./... -v
```

Untuk package tertentu:

```bash
go test ./internal/splitter/... -v -run TestSplit_PixelCorrectness
```
