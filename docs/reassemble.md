# image-splitter — Panduan Reassemble

Command `reassemble` menyusun kembali file-file sel hasil split menjadi satu gambar kolase.

```bash
bin/image-splitter reassemble [flags]
```

**Kegunaan:**
- Rebuild kolase asli setelah sel-sel diedit satu per satu
- **Ganti layout grid** — mis. dari 4×2 jadi 2×4
- **Reorder sel** — susun ulang urutan foto sesuai keinginan

---

## Daftar Isi

- [Semua Flag](#semua-flag)
- [Flag `--order`](#flag---order)
- [Contoh](#contoh)
- [Pesan Output](#pesan-output)
- [Error Umum](#error-umum)

---

## Semua Flag

| Flag | Default | Keterangan |
|---|---|---|
| `--input` | `./output` | Direktori berisi file sel (`cell_rowNN_colNN.{png,jpg}`) |
| `--rows` | `0` (auto) | Baris di kolase output (0 = deteksi dari nama file) |
| `--cols` | `0` (auto) | Kolom di kolase output (0 = deteksi dari nama file) |
| `--order` | _(kosong)_ | Urutan sel kustom — lihat penjelasan di bawah |
| `--output` | `collage.png` | Path file output kolase |
| `--quality` | `0` | Format output: 0 = PNG, 1–100 = JPEG |

**Aturan `--rows` dan `--cols`:**
- Dikosongkan (0): grid dideteksi otomatis dari nama file sel
- Diisi: wajib `rows × cols = jumlah file sel` di direktori input
  - Ada 8 file sel → boleh: `2×4`, `4×2`, `8×1`, `1×8`

**Aturan `--quality`:** sama persis dengan flag `--quality` di split.
Tambahan: jika `--quality 0` tapi `--output` diakhiri `.jpg`, tool otomatis pakai quality 85.

---

## Flag `--order`

`--order` menentukan urutan sel dari **input** yang ditempatkan ke posisi **output** secara berurutan kiri-ke-kanan, atas-ke-bawah.

### Dasar: bagaimana indeks dihitung

Setelah split, file di folder output diurutkan **alfabetis** (otomatis sesuai nama `cell_rowNN_colNN`). Urutan inilah yang menjadi indeks untuk `--order`:

```
Hasil split 4×2 (8 file):

indeks 0 → cell_row00_col00.png  (baris 0, kolom 0 = foto kiri atas)
indeks 1 → cell_row00_col01.png  (baris 0, kolom 1 = foto kanan atas)
indeks 2 → cell_row01_col00.png
indeks 3 → cell_row01_col01.png
indeks 4 → cell_row02_col00.png
indeks 5 → cell_row02_col01.png
indeks 6 → cell_row03_col00.png
indeks 7 → cell_row03_col01.png  (baris 3, kolom 1 = foto kanan bawah)
```

Visualnya:

```
Grid asli 4×2:
┌──────┬──────┐
│  0   │  1   │
├──────┼──────┤
│  2   │  3   │
├──────┼──────┤
│  4   │  5   │
├──────┼──────┤
│  6   │  7   │
└──────┴──────┘
```

### Cara baca `--order`

`--order A,B,C,D,...` artinya:
- Posisi pertama di grid output → isi dengan sel input nomor A
- Posisi kedua → isi dengan sel input nomor B
- dst, kiri-ke-kanan, atas-ke-bawah

Posisi grid output untuk `--rows 4 --cols 2`:
```
posisi 0  posisi 1
posisi 2  posisi 3
posisi 4  posisi 5
posisi 6  posisi 7
```

### Contoh konkret untuk 4×2

**Balik urutan** (`--order 7,6,5,4,3,2,1,0`):
```
Asli:          Hasil:
┌────┬────┐    ┌────┬────┐
│ 0  │ 1  │    │ 7  │ 6  │
├────┼────┤    ├────┼────┤
│ 2  │ 3  │    │ 5  │ 4  │
├────┼────┤    ├────┼────┤
│ 4  │ 5  │    │ 3  │ 2  │
├────┼────┤    ├────┼────┤
│ 6  │ 7  │    │ 1  │ 0  │
└────┴────┘    └────┴────┘
```

**Tukar kolom kiri-kanan** (`--order 1,0,3,2,5,4,7,6`):
```
Asli:          Hasil:
┌────┬────┐    ┌────┬────┐
│ 0  │ 1  │    │ 1  │ 0  │
├────┼────┤    ├────┼────┤
│ 2  │ 3  │    │ 3  │ 2  │
├────┼────┤    ├────┼────┤
│ 4  │ 5  │    │ 5  │ 4  │
├────┼────┤    ├────┼────┤
│ 6  │ 7  │    │ 7  │ 6  │
└────┴────┘    └────┴────┘
```

**Ganti layout 4×2 → 2×4 sekaligus reorder** (`--rows 2 --cols 4 --order 0,2,4,6,1,3,5,7`):
```
Asli (4×2):     Hasil (2×4):
┌───┬───┐       ┌───┬───┬───┬───┐
│ 0 │ 1 │       │ 0 │ 2 │ 4 │ 6 │
├───┼───┤   →   ├───┼───┼───┼───┤
│ 2 │ 3 │       │ 1 │ 3 │ 5 │ 7 │
├───┼───┤       └───┴───┴───┴───┘
│ 4 │ 5 │
├───┼───┤
│ 6 │ 7 │
└───┴───┘
```

### Aturan validasi `--order`

- Jumlah indeks harus = `rows × cols` (total sel)
- Setiap indeks harus unik (tidak boleh duplikat)
- Setiap indeks harus dalam rentang `0` sampai `(jumlah_sel - 1)`

---

## Contoh

### 1. Rebuild kolase dengan layout yang sama

```bash
bin/image-splitter reassemble --input ./output
```

Grid dideteksi otomatis dari nama file, sel disusun dengan urutan asli.

---

### 2. Ganti layout grid dari 2×3 menjadi 3×2

```bash
bin/image-splitter reassemble --input ./output --rows 3 --cols 2
```

6 sel yang sama, disusun dalam 3 baris × 2 kolom.

---

### 3. Ganti layout dari 4×2 menjadi 2×4

```bash
bin/image-splitter reassemble --input ./output --rows 2 --cols 4
```

---

### 4. Balik urutan semua sel (4×2)

```bash
bin/image-splitter reassemble --input ./output --rows 4 --cols 2 --order 7,6,5,4,3,2,1,0
```

---

### 5. Tukar kolom kiri dan kanan (4×2)

```bash
bin/image-splitter reassemble --input ./output --rows 4 --cols 2 --order 1,0,3,2,5,4,7,6
```

---

### 6. Output JPEG kualitas 90

```bash
bin/image-splitter reassemble --input ./output --output kolase-baru.jpg --quality 90
```

Atau pakai ekstensi `.jpg` di `--output` saja (auto quality 85):

```bash
bin/image-splitter reassemble --input ./output --output kolase-baru.jpg
```

---

### 7. Output ke path tertentu

```bash
bin/image-splitter reassemble --input ./output --output ./hasil/kolase-final.png
```

Direktori `./hasil/` dibuat otomatis.

---

## Pesan Output

### Sukses

```
Reassembling 8 cells from "./output" into 2×4 grid...
Done. Collage saved to collage.png
```

---

## Error Umum

| Situasi | Pesan Error |
|---|---|
| Direktori input tidak ada | `read directory "./output": no such file or directory` |
| Tidak ada file sel | `no cell files found in "./output"` |
| Jumlah sel tidak cocok dengan grid | `grid 2×4 needs 8 cells, found 6 in "./output"` |
| `--order` jumlah indeks salah | `--order must have exactly 8 indices, got 6` |
| Indeks `--order` out of range | `--order: index 9 out of range [0, 8)` |
| Indeks `--order` duplikat | `--order: duplicate index 2` |
