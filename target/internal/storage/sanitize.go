package storage

import (
	"bytes"
	"image/jpeg"
	"image/png"

	apperr "goadmin/internal/errors"
)

// SanitizeImage memvalidasi (magic-byte) LALU me-RE-ENCODE jpeg/png: decode →
// encode ulang membuang metadata/payload tersembunyi (anti polyglot/eksploit)
// dan memastikan byte benar-benar gambar yang dapat di-render. gif & webp
// dikembalikan apa adanya (jaga animasi; tak ada encoder webp di stdlib).
// Mengembalikan byte bersih + ekstensi kanonik.
func SanitizeImage(data []byte, maxBytes int) ([]byte, string, error) {
	ext, err := DetectImage(data, maxBytes)
	if err != nil {
		return nil, "", err
	}
	switch ext {
	case ".jpg":
		img, derr := jpeg.Decode(bytes.NewReader(data))
		if derr != nil {
			return nil, "", apperr.Validation("Gambar JPEG tidak valid", nil)
		}
		var buf bytes.Buffer
		if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 90}); err != nil {
			return nil, "", apperr.Internal("re-encode jpeg gagal: " + err.Error())
		}
		return buf.Bytes(), ext, nil
	case ".png":
		img, derr := png.Decode(bytes.NewReader(data))
		if derr != nil {
			return nil, "", apperr.Validation("Gambar PNG tidak valid", nil)
		}
		var buf bytes.Buffer
		enc := png.Encoder{CompressionLevel: png.DefaultCompression}
		if err := enc.Encode(&buf, img); err != nil {
			return nil, "", apperr.Internal("re-encode png gagal: " + err.Error())
		}
		return buf.Bytes(), ext, nil
	default:
		return data, ext, nil // gif/webp: tervalidasi magic-byte
	}
}
