# Image Splitter

## Project Description

image-splitter adalah CLI tool berbasis Go untuk memotong gambar grid/kolase menjadi gambar-gambar satuan dengan kualitas tinggi, dilengkapi kemampuan upscaling.

## Problem Statement
Kita sering bekerja dengan gambar kolase (seperti mood board, grid foto, atau komposit) yang perlu dipecah menjadi file-file individual. Proses manual menggunakan Photoshop atau tools lain memakan waktu, tidak bisa di-automate, dan sering menurunkan kualitas gambar saat export.

## Solution
CLI tool yang menerima gambar grid sebagai input, memotongnya berdasarkan jumlah baris dan kolom yang ditentukan user, dengan output berkualitas tinggi dan opsi upscaling — semua dari terminal dalam satu command.

## Core Features
- Grid Splitting — Memotong gambar berdasarkan input rows × cols. Setiap cell dipotong dengan presisi pixel menggunakan SubImage (zero-copy, tidak ada degradasi kualitas saat cropping).
- High Quality Output — Support JPEG quality 1-100 dan PNG lossless. Default output PNG untuk menghindari kompresi lossy.
- Upscaling — Opsi untuk memperbesar tiap cell menggunakan algoritma CatmullRom resampling, cocok untuk foto high-res yang butuh diperbesar lebih lanjut.
- Auto Format Detection — Otomatis mendeteksi format input (JPEG/PNG) dari extension atau magic bytes, tidak perlu flag tambahan.
- Structured Output Naming — File output dinamai dengan format cell_row00_col00.ext dengan zero-padding sehingga sorting file tetap benar.

## Technical Highlights

- Dibangun dengan Go standard library + golang.org/x/image untuk resampling
- Zero-copy cropping via SubImage() — kualitas crop tidak turun sama sekali
- Degradasi kualitas hanya terjadi satu kali saat encode ke file output
- Separation of concerns yang ketat: splitter, upscaler, dan imageio adalah package terpisah
- CLI menggunakan Cobra untuk UX yang clean dan extensible

## Project Structure

```bash
image-splitter/
├── main.go                    # Entry point, inisialisasi CLI
├── go.mod
├── go.sum
├── Makefile                   # Run dev & build binary macos, windows, linux
│
├── cmd/
│   └── root.go                # Definisi command & flags (cobra)
│
├── internal/
│   ├── splitter/
│   │   ├── splitter.go        # Core logic: crop, split grid
│   │   └── splitter_test.go
│   │
│   ├── upscaler/
│   │   ├── upscaler.go        # Upscale logic (CatmullRom, dll)
│   │   └── upscaler_test.go
│   │
│   ├── imageio/
│   │   ├── reader.go          # Decode gambar (jpeg/png auto-detect)
│   │   ├── writer.go          # Encode & save ke file
│   │   └── imageio_test.go
│   │
│   └── config/
│       └── config.go          # Struct config dari flags CLI
│
└── output/                    # Default output directory (gitignore)

```
