package imageio

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"os"
)

// Format represents the detected image format.
type Format string

const (
	FormatJPEG Format = "jpeg"
	FormatPNG  Format = "png"
)

// Load opens the file at path, decodes it, and returns the image along with
// the detected format. Supported formats: JPEG, PNG.
func Load(path string) (image.Image, Format, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, "", fmt.Errorf("open %q: %w", path, err)
	}
	defer f.Close()

	img, fmtName, err := image.Decode(f)
	if err != nil {
		return nil, "", fmt.Errorf("decode %q: %w", path, err)
	}

	switch fmtName {
	case "jpeg":
		return img, FormatJPEG, nil
	case "png":
		return img, FormatPNG, nil
	default:
		return img, Format(fmtName), nil
	}
}

// ReadPHYs reads the pHYs chunk from a PNG file and returns the
// pixels-per-unit values (always in meters when unit==1). ok is false if
// the file is not a PNG or has no pHYs chunk.
func ReadPHYs(path string) (xppu, yppu uint32, ok bool) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()

	// Verify PNG signature.
	var sig [8]byte
	if _, err := io.ReadFull(f, sig[:]); err != nil {
		return
	}
	if !bytes.Equal(sig[:], []byte{137, 80, 78, 71, 13, 10, 26, 10}) {
		return
	}

	// Scan chunks until pHYs is found or IDAT is reached (pHYs must precede IDAT).
	for {
		var length uint32
		if err := binary.Read(f, binary.BigEndian, &length); err != nil {
			return
		}
		var typ [4]byte
		if _, err := io.ReadFull(f, typ[:]); err != nil {
			return
		}
		data := make([]byte, length)
		if _, err := io.ReadFull(f, data); err != nil {
			return
		}
		if _, err := io.ReadFull(f, make([]byte, 4)); err != nil { // skip CRC
			return
		}
		switch string(typ[:]) {
		case "pHYs":
			if len(data) >= 9 {
				xppu = binary.BigEndian.Uint32(data[0:4])
				yppu = binary.BigEndian.Uint32(data[4:8])
				ok = true
			}
			return
		case "IDAT":
			return
		}
	}
}
