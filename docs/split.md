# image-splitter — Panduan Split

Command `split` (default) memotong gambar grid/kolase menjadi sel-sel terpisah.

```bash
bin/image-splitter <path-gambar> [flags]
```

---

## Daftar Isi

- [Mode Split](#mode-split)
- [Semua Flag](#semua-flag)
- [Flag `--quality`](#flag---quality)
- [Flag `--scale`](#flag---scale)
- [Perbedaan `--quality` dan `--scale`](#perbedaan---quality-dan---scale)
- [Flag `--auto`](#flag---auto)
- [Flag `--trim`](#flag---trim)
- [Format Output File](#format-output-file)
- [Contoh](#contoh)
- [Pesan Output](#pesan-output)
- [Error Umum](#error-umum)

---

## Mode Split

Ada tiga cara pakai:

| Mode | Command | Kapan dipakai |
|---|---|---|
| Manual | `--rows N --cols N` | Kamu tahu persis berapa baris & kolomnya |
| Manual + presisi | `--rows N --cols N --auto` | Tahu jumlah sel, tapi batas antar foto tidak presisi |
| Full auto | `--auto` | Tidak tahu jumlah baris/kolom — biarkan tool deteksi sendiri |

---

## Semua Flag

| Flag | Shorthand | Default | Keterangan |
|---|---|---|---|
| `--rows` | `-r` | `0` | Jumlah baris dalam grid |
| `--cols` | `-c` | `0` | Jumlah kolom dalam grid |
| `--auto` | `-a` | `false` | Auto-detect seam & ukuran grid |
| `--trim` | `-t` | `false` | Auto-hapus border seragam dari sumber dan setiap sel |
| `--trim-tolerance` | _(none)_ | `40` | Max perbedaan RGB untuk deteksi warna border |
| `--output` | `-o` | `./output` | Direktori tempat menyimpan hasil |
| `--quality` | `-q` | `0` | Format & kualitas output (0 = PNG, 1–100 = JPEG) |
| `--scale` | `-s` | `1.0` | Faktor perbesar ukuran tiap sel (≥ 1.0) |

**Aturan `--rows` dan `--cols`:**
- Tanpa `--auto` → wajib diisi, harus ≥ 1
- Dengan `--auto` → boleh dikosongkan, tool deteksi sendiri
- Boleh isi salah satu: `--rows 3 --auto` → baris diset 3, kolom dideteksi otomatis

---

## Flag `--quality`

Flag ini mengontrol **format dan tingkat kompresi** output:

| Nilai | Format output | Penjelasan |
|---|---|---|
| `0` (default) | **PNG** | Lossless — piksel disimpan 100% persis, tidak ada degradasi sama sekali. File lebih besar dari JPEG. |
| `95` – `100` | **JPEG** | Kualitas sangat tinggi. Hampir tidak bisa dibedakan dari aslinya. File cukup besar. |
| `80` – `94` | **JPEG** | Kualitas tinggi. Artefak hampir tidak terlihat. **Pilihan umum untuk produksi.** |
| `60` – `79` | **JPEG** | Kualitas sedang. Mulai muncul artefak di detail halus. File kecil. |
| `1` – `59` | **JPEG** | Kualitas rendah. Artefak jelas terlihat: blur, kotak-kotak, warna pudar. File sangat kecil. |

**Kapan pakai apa:**
- Hasil akan diedit lagi → `0` (PNG), tidak ada degradasi antar-edit
- Upload ke web / sharing → `--quality 85` atau `--quality 90`
- Preview cepat → `--quality 60`

**Perkiraan ukuran file** (sel 400×300 px dari foto):

```
PNG  (--quality 0)   → ~300 KB
JPEG --quality 90    → ~50 KB
JPEG --quality 85    → ~35 KB
JPEG --quality 70    → ~20 KB
```

---

## Flag `--scale`

Flag ini mengubah **ukuran piksel** tiap sel setelah dipotong, menggunakan algoritma **CatmullRom resampling**.

| Nilai | Efek |
|---|---|
| `1.0` (default) | Tidak ada perubahan ukuran |
| `1.5` | Sel diperbesar 1.5× (mis. 400×300 → 600×450) |
| `2.0` | Sel diperbesar 2× (mis. 400×300 → 800×600) |
| `3.0` | Sel diperbesar 3× (mis. 400×300 → 1200×900) |
| `< 1.0` | **Error** — downscale tidak didukung |

**Kapan pakai:**
- Gambar sumber resolusi rendah, hasil potongan butuh dicetak besar → `--scale 2.0`
- Resolusi sumber sudah cukup → tidak perlu `--scale`

**Catatan:** `--scale` memperbesar piksel dengan interpolasi. Ini menambah resolusi tapi tidak menambah detail asli yang memang tidak ada di gambar sumber.

---

## Perbedaan `--quality` dan `--scale`

| | `--quality` | `--scale` |
|---|---|---|
| **Mengubah apa** | Seberapa lossy kompresi saat simpan | Ukuran piksel gambar (resolusi) |
| **Efek pada piksel** | Tidak mengubah jumlah piksel, hanya presisinya | Menambah jumlah piksel |
| **Efek pada ukuran file** | Makin kecil nilai → file makin kecil | Makin besar scale → file makin besar |
| **Contoh** | sel 400×300 tetap 400×300, detail bisa berkurang | sel 400×300 jadi 800×600 |

Analogi:
```
--quality = seberapa keras kamu "kompres" saat menyimpan foto
--scale   = seberapa besar kamu "perbesar" foto sebelum menyimpan
```

Contoh keduanya dipakai bersamaan:
```bash
# Perbesar 2× lalu simpan sebagai JPEG kualitas 90
bin/image-splitter kolase.jpg --rows 2 --cols 3 --scale 2.0 --quality 90
# sel 400×300 → diperbesar jadi 800×600 → disimpan JPEG 90
```

---

## Flag `--auto`

`--auto` mengaktifkan deteksi otomatis posisi batas antar foto menggunakan **analisis energi piksel**.

**Tanpa `--auto`:** tool membagi gambar secara matematis rata:
```
cellW = totalWidth  / cols
cellH = totalHeight / rows
```
Gambar 1200×600 dibagi 2×3 → setiap sel pasti 400×300 px.

**Dengan `--auto`:** tool mencari batas antar foto yang sebenarnya:
1. Hitung perbedaan piksel antar setiap baris berurutan (energy profile)
2. Haluskan dengan box filter untuk mengurangi noise
3. Cari baris/kolom dengan perbedaan tertinggi → itulah batas seam
4. Potong di posisi seam tersebut

Hasilnya tiap sel bisa **tidak sama persis ukurannya** — tapi batas potongan tepat di garis antar foto.

**Kapan perlu `--auto`:**
- Kolase dibuat oleh Instagram, Canva, atau app lain yang menambahkan padding/gap antar foto
- Tinggi atau lebar tiap foto dalam kolase tidak persis sama
- Terlihat garis putih/hitam tipis di antara foto dalam kolase

**Kapan tidak perlu `--auto`:**
- Sprite sheet / grid programatik → semua sel persis sama ukurannya
- Hanya mau potong rata tanpa mempedulikan batas sebenarnya

**`--auto` tanpa `--rows`/`--cols`:** tool juga mendeteksi sendiri berapa baris dan kolomnya.

**Penting — `--auto` tidak menghapus border:**
`--auto` hanya mendeteksi POSISI seam (batas potong) yang lebih presisi, bukan menghapus warna border. Jika gambar punya background gelap/terang, seam masih bisa jatuh di dalam area border tersebut, sehingga hasil potongan masih menyisakan tepi berwarna.

**Untuk hasil bersih (tanpa border), selalu gunakan `--auto` bersama `--trim`:**
```bash
bin/image-splitter kolase.jpg --auto --trim
```

`--trim` akan menghapus border luar sebelum split, lalu membersihkan sisa border dari setiap sel.

---

## Flag `--trim`

`--trim` menghapus border/background berwarna seragam secara otomatis.

**Cara kerja — dua tahap:**

**Tahap 1 — Pre-trim (sebelum split):** Border luar gambar sumber dideteksi dan dihapus terlebih dahulu, kemudian barulah gambar dipotong. Ini menyelesaikan kasus paling umum: kolase dengan background gelap/terang yang membingkai seluruh gambar.

**Tahap 2 — Post-trim (setelah split):** Setiap sel hasil potongan juga di-trim untuk membersihkan sisa gap/border antar foto jika ada.

**Algoritma deteksi border:**
1. Sampel 4 piksel sudut (kiri-atas, kanan-atas, kiri-bawah, kanan-bawah)
2. Jika keempat sudut warnanya mirip (dalam `--trim-tolerance`) → itulah warna border
3. Jalan dari setiap tepi ke dalam sampai menemukan baris/kolom yang bukan border
4. Crop ke area interior
5. Jika hasil crop < 10×10 px → dikembalikan tanpa perubahan

**Kapan pakai:**
- Kolase punya background/border gelap atau terang seragam di tepinya (kasus paling umum)
- Hasil potongan masih menyisakan tepi hitam/putih/abu-abu

**Kapan tidak perlu:**
- Konten foto menyentuh tepi tanpa border
- Warna di sudut foto bervariasi / tidak seragam

```bash
# Rekomendasi: pakai trim bersama auto-detect
bin/image-splitter kolase.jpg --auto --trim

# Atau dengan ukuran grid manual
bin/image-splitter kolase.jpg --rows 2 --cols 3 --trim
```

---

## Flag `--trim-tolerance`

Mengontrol seberapa "ketat" deteksi warna border. Default: **40**.

| Nilai | Kapan pakai |
|---|---|
| `10`–`20` | Gambar PNG lossless — border warnanya sangat seragam |
| `40` (default) | JPEG — menoleransi artifact kompresi di area border |
| `50`–`80` | Border warnanya tidak seragam / ada gradasi tipis |

JPEG compression menyebabkan piksel-piksel dalam area warna seragam bisa berbeda 20–30 unit. Nilai default 40 menangani ini.

```bash
# Ketat (PNG atau border yang sangat bersih)
bin/image-splitter kolase.png --trim --trim-tolerance 15

# Longgar (JPEG dengan border yang sedikit bervariasi)
bin/image-splitter kolase.jpg --auto --trim --trim-tolerance 60
```

---

## Format Output File

### Penamaan file sel

```
cell_row{R}_col{C}.{ext}
```

- `{R}` = nomor baris, zero-based, zero-padded minimal 2 digit
- `{C}` = nomor kolom, zero-based, zero-padded minimal 2 digit
- Padding melebar otomatis: grid 100×100 → `cell_row00_col00` s/d `cell_row99_col99`
- `{ext}` = `png` jika `--quality 0`, `jpg` jika `--quality 1-100`

### Contoh nama file — grid 4×2

```
output/
├── cell_row00_col00.png   ← baris 0, kolom 0  (indeks 0 untuk --order)
├── cell_row00_col01.png   ← baris 0, kolom 1  (indeks 1)
├── cell_row01_col00.png   ← baris 1, kolom 0  (indeks 2)
├── cell_row01_col01.png   ← baris 1, kolom 1  (indeks 3)
├── cell_row02_col00.png   ← baris 2, kolom 0  (indeks 4)
├── cell_row02_col01.png   ← baris 2, kolom 1  (indeks 5)
├── cell_row03_col00.png   ← baris 3, kolom 0  (indeks 6)
└── cell_row03_col01.png   ← baris 3, kolom 1  (indeks 7)
```

### Dimensi tidak habis dibagi

Jika ukuran gambar tidak habis dibagi, **sel terakhir** di setiap baris/kolom mendapat sisa piksel.

Contoh: gambar 601×401 dibagi 2×3:
- Kolom 0 & 1 → lebar 200 px; Kolom 2 → lebar **201 px**
- Baris 0 → tinggi 200 px; Baris 1 → tinggi **201 px**

Total area semua sel = total area gambar asli — tidak ada piksel yang hilang.

---

## Contoh

### 1. Split dasar, output PNG

```bash
bin/image-splitter kolase.jpg --rows 2 --cols 3
```

Output default ke `./output/`, format PNG (lossless).

---

### 2. Auto-detect segalanya

```bash
bin/image-splitter kolase.jpg --auto
```

Output contoh:

```
Auto-detected grid size: 2 rows × 3 cols
Detecting seams in "kolase.jpg"...
  horizontal seams: [319]
  vertical seams:   [514 1028]
Splitting "kolase.jpg" into 2×3 cells → ./output
  wrote cell_row00_col00.png
  ...
Done. 6 cells written to ./output

To rebuild a collage from these cells:
  ImageMagick:  montage ./output/cell_*.png -tile 3x2 -geometry +0+0 collage.png
  Reassemble:   bin/image-splitter reassemble --input ./output --rows 2 --cols 3
```

---

### 3. Auto-detect + trim border (rekomendasi untuk kolase nyata)

```bash
bin/image-splitter kolase.jpg --auto --trim
```

Urutan yang terjadi:
1. `--trim` mendeteksi dan menghapus border/background luar dari gambar sumber
2. `--auto` mendeteksi seam di gambar yang sudah bersih
3. Split dilakukan di posisi seam yang tepat
4. `--trim` lagi membersihkan sisa border dari setiap sel hasil potong

---

### 4. Tahu jumlah sel, tapi batas tidak presisi

```bash
bin/image-splitter kolase.jpg --rows 4 --cols 2 --auto
```

---

### 5. Output JPEG kualitas 85

```bash
bin/image-splitter kolase.jpg --rows 2 --cols 3 --quality 85
```

Menghasilkan `cell_row00_col00.jpg` … `cell_row01_col02.jpg`.

---

### 6. Output ke direktori kustom

```bash
bin/image-splitter kolase.jpg --rows 2 --cols 3 --output ./hasil
```

Direktori `./hasil/` dibuat otomatis jika belum ada.

---

### 7. Upscale tiap sel 2×

```bash
bin/image-splitter kolase.jpg --rows 2 --cols 3 --scale 2.0
```

Gambar 1200×600 → tiap sel 400×300 → setelah upscale: **800×600 per sel**.

---

### 8. JPEG + upscale + direktori kustom sekaligus

```bash
bin/image-splitter kolase.jpg --rows 2 --cols 3 --quality 90 --scale 2.0 --output ./tiles
```

---

### 9. Grid 1 kolom — potong jadi strip horizontal

```bash
bin/image-splitter kolase.jpg --rows 4 --cols 1
```

---

### 10. Shorthand flag

```bash
bin/image-splitter kolase.jpg -r 2 -c 3 -q 80 -s 1.5 -t -o ./out
```

---

## Pesan Output

### Sukses

```
Splitting "kolase.jpg" into 4×2 cells → ./output
  wrote cell_row00_col00.png
  wrote cell_row00_col01.png
  ...
Done. 8 cells written to ./output

To rebuild a collage from these cells:
  ImageMagick:  montage ./output/cell_*.png -tile 2x4 -geometry +0+0 collage.png
  Reassemble:   bin/image-splitter reassemble --input ./output --rows 4 --cols 2
```

Dua pilihan rebuild yang ditampilkan:
- **ImageMagick** — tool eksternal (`sudo apt install imagemagick`), pakai jika sudah terinstall
- **Reassemble** — command bawaan tool ini, langsung bisa dijalankan

---

## Error Umum

| Situasi | Pesan Error |
|---|---|
| Tidak ada `--rows`/`--cols` dan tidak ada `--auto` | `--rows must be >= 1 (or use --auto to detect grid size)` |
| `--rows` melebihi tinggi gambar | `--rows 700 exceeds image height 600` |
| `--cols` melebihi lebar gambar | `--cols 1300 exceeds image width 1200` |
| `--quality` di luar range | `--quality must be 0 (PNG) or between 1 and 100, got 150` |
| `--scale` kurang dari 1.0 | `--scale must be >= 1.0, got 0.50` |
| File input tidak ditemukan | `open "file.jpg": no such file or directory` |
