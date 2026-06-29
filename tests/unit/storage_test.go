package unit

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"goadmin/internal/config"
	apperr "goadmin/internal/errors"
	"goadmin/internal/storage"
)

// realPNG menghasilkan PNG kecil yang BENAR-BENAR dapat di-decode.
func realPNG(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	img.Set(0, 0, color.RGBA{R: 10, G: 20, B: 30, A: 255})
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode png: %v", err)
	}
	return buf.Bytes()
}

// Magic-byte minimal tiap format.
var (
	pngMagic  = []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, 0, 0, 0, 0}
	jpegMagic = []byte{0xFF, 0xD8, 0xFF, 0xE0, 0, 0, 0, 0}
	gifMagic  = []byte("GIF89a______")
	webpMagic = append([]byte("RIFF\x00\x00\x00\x00WEBP"), make([]byte, 8)...)
)

// DetectImage menerima gambar nyata (magic-byte), menolak non-gambar & oversize.
func TestStorage_DetectImage(t *testing.T) {
	cases := []struct {
		name string
		data []byte
		ext  string
	}{
		{"png", pngMagic, ".png"},
		{"jpeg", jpegMagic, ".jpg"},
		{"gif", gifMagic, ".gif"},
		{"webp", webpMagic, ".webp"},
	}
	for _, tc := range cases {
		ext, err := storage.DetectImage(tc.data, storage.MaxImageBytes)
		if err != nil || ext != tc.ext {
			t.Fatalf("%s: ext=%q err=%v (harap %s)", tc.name, ext, err, tc.ext)
		}
	}

	// Bukan gambar (teks) → 422, walau "MIME klien" mengaku gambar.
	if _, err := storage.DetectImage([]byte("<html>halo</html>"), storage.MaxImageBytes); err == nil {
		t.Fatal("non-gambar harus ditolak (magic-byte, bukan MIME klien)")
	}
	// Oversize → 422.
	big := make([]byte, storage.MaxImageBytes+1)
	copy(big, pngMagic)
	if _, err := storage.DetectImage(big, storage.MaxImageBytes); err == nil {
		t.Fatal("oversize harus ditolak")
	}
	// Kosong → 422.
	if _, err := storage.DetectImage(nil, storage.MaxImageBytes); err == nil {
		t.Fatal("kosong harus ditolak")
	}

	// Pastikan status AppError = 422 (Validation).
	_, err := storage.DetectImage([]byte("nope"), storage.MaxImageBytes)
	if ae, ok := apperr.As(err); !ok || ae.Status != 422 {
		t.Fatalf("harus 422, dapat: %v", err)
	}
}

// SanitizeImage me-RE-ENCODE PNG asli (hasil tetap PNG valid); menolak byte
// yang lolos magic-byte tapi TIDAK dapat di-decode (anti payload tersembunyi).
func TestStorage_SanitizeImage(t *testing.T) {
	clean, ext, err := storage.SanitizeImage(realPNG(t), storage.MaxImageBytes)
	if err != nil || ext != ".png" {
		t.Fatalf("png asli: ext=%q err=%v", ext, err)
	}
	if _, derr := png.Decode(bytes.NewReader(clean)); derr != nil {
		t.Fatalf("hasil re-encode tak dapat di-decode: %v", derr)
	}

	// Hanya signature PNG + sampah (lolos DetectImage, gagal decode) → ditolak.
	fake := append([]byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}, []byte("garbage-not-an-image")...)
	if _, _, err := storage.SanitizeImage(fake, storage.MaxImageBytes); err == nil {
		t.Fatal("PNG palsu (tak dapat di-decode) harus ditolak saat re-encode")
	}
}

// New memilih driver dari config (tanpa jaringan): local → *Local, s3 → *S3.
func TestStorage_DriverSelection(t *testing.T) {
	if _, ok := storage.New(config.StorageConfig{Driver: "local"}).(*storage.Local); !ok {
		t.Fatal("driver local harus → *Local")
	}
	s3cfg := config.StorageConfig{
		Driver: "s3", S3Endpoint: "s3.example.com", S3Bucket: "bucket",
		S3AccessKey: "k", S3SecretKey: "s", S3PublicURL: "https://cdn.example.com",
	}
	if _, ok := storage.New(s3cfg).(*storage.S3); !ok {
		t.Fatal("driver s3 harus → *S3")
	}
}

// Local.SaveImage menulis file + mengembalikan URL publik yang benar.
func TestStorage_LocalSaveImage(t *testing.T) {
	dir := t.TempDir()
	st := storage.NewLocal(config.StorageConfig{Dir: dir, URLBase: "/uploads"})

	url, err := st.SaveImage(context.Background(), pngMagic, ".png")
	if err != nil {
		t.Fatalf("save: %v", err)
	}
	if !strings.HasPrefix(url, "/uploads/") || !strings.HasSuffix(url, ".png") {
		t.Fatalf("URL tak sesuai: %s", url)
	}
	// File benar-benar ada di disk.
	name := strings.TrimPrefix(url, "/uploads/")
	if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
		t.Fatalf("file tak tersimpan: %v", err)
	}
}
