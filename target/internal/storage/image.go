package storage

import (
	"net/http"

	apperr "goadmin/internal/errors"
)

// MaxImageBytes = batas ukuran gambar upload (2 MB).
const MaxImageBytes = 2 << 20

// imageExt memetakan content-type ter-sniff → ekstensi kanonik (whitelist).
var imageExt = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/gif":  ".gif",
	"image/webp": ".webp",
}

// DetectImage memvalidasi byte sebagai gambar berdasar MAGIC-BYTE (bukan MIME
// klien yang bisa dipalsukan) + batas ukuran. Mengembalikan ekstensi kanonik.
func DetectImage(data []byte, maxBytes int) (string, error) {
	if len(data) == 0 {
		return "", apperr.Validation("File kosong", nil)
	}
	if len(data) > maxBytes {
		return "", apperr.Validation("Ukuran gambar melebihi batas (maks 2MB)", nil)
	}

	ct := http.DetectContentType(data) // membaca ~512 byte awal (magic byte)
	if ct == "application/octet-stream" && isWebP(data) {
		ct = "image/webp"
	}
	ext, ok := imageExt[ct]
	if !ok {
		return "", apperr.Validation("Tipe file tidak didukung (hanya jpg/png/gif/webp)", map[string]string{"file": "harus gambar"})
	}
	return ext, nil
}

// isWebP mendeteksi kontainer WebP (RIFF....WEBP) bila sniffer bawaan tak yakin.
func isWebP(b []byte) bool {
	return len(b) >= 12 && string(b[0:4]) == "RIFF" && string(b[8:12]) == "WEBP"
}
