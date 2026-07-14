// Package storage menyediakan penyimpanan file upload yang dapat ditukar
// (interface), dengan implementasi lokal (disk). Validasi gambar berbasis
// MAGIC-BYTE (bukan MIME klien) — lihat image.go.
package storage

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"goadmin/internal/config"
	"goadmin/internal/helpers"
)

// editorSubdir = sub-folder khusus berkas gambar yang diunggah lewat rich text
// editor (file manager Trumbowyg) — dipisah dari upload lain agar bisa di-list.
const editorSubdir = "editor"

// MediaFile = satu berkas media editor (untuk file manager rich text).
type MediaFile struct {
	Name string `json:"name"`
	URL  string `json:"url"`
	Key  string `json:"key"`
}

// editorKeyRe membatasi key media ke "editor/<nama-aman>" (anti path-traversal).
var editorKeyRe = regexp.MustCompile(`^editor/([A-Za-z0-9._-]+)$`)

// ErrInvalidMediaKey = key media tak valid (di luar folder editor / berbahaya).
var ErrInvalidMediaKey = errors.New("storage: key media tidak valid")

// safeMediaName mengembalikan nama file dari key "editor/<nama>" bila aman.
func safeMediaName(key string) (string, bool) {
	m := editorKeyRe.FindStringSubmatch(key)
	if m == nil {
		return "", false
	}
	return m[1], true
}

// Storage = kontrak penyimpanan (mudah diganti S3/OSS kemudian).
type Storage interface {
	// SaveImage menyimpan byte gambar (sudah tervalidasi) → mengembalikan URL publik.
	SaveImage(ctx context.Context, data []byte, ext string) (string, error)
	// SaveMedia menyimpan gambar editor di folder "editor/" → MediaFile (name/url/key).
	SaveMedia(ctx context.Context, data []byte, ext string) (MediaFile, error)
	// ListMedia mengembalikan daftar berkas editor (untuk file manager).
	ListMedia(ctx context.Context) ([]MediaFile, error)
	// DeleteMedia menghapus berkas editor by key (divalidasi anti path-traversal).
	DeleteMedia(ctx context.Context, key string) error
}

// Local menyimpan ke folder disk yang disajikan sebagai static di URLBase.
type Local struct {
	dir     string
	urlBase string
}

// IsObjectDriver melaporkan apakah Driver menunjuk object storage ("s3" atau "oss"),
// bukan disk lokal. Dipakai juga oleh app untuk memutuskan mount static /uploads.
func IsObjectDriver(driver string) bool {
	return strings.EqualFold(driver, "s3") || strings.EqualFold(driver, "oss")
}

// New memilih implementasi berdasar Driver: "s3"/"oss" → S3-compatible, "local" → disk.
//
// Mengembalikan error bila driver object storage gagal dirakit. DULU fungsi ini diam-diam
// fallback ke Local saat init gagal — dan "oss" bahkan tak dikenali sama sekali, sehingga
// STORAGE_DRIVER=oss menulis file ke disk container: upload seolah sukses tapi tak pernah
// sampai ke bucket, lalu hilang saat container restart / tak terlihat oleh replika lain.
// Salah konfigurasi harus berisik, bukan menyamar jadi "berhasil".
func New(cfg config.StorageConfig) (Storage, error) {
	if IsObjectDriver(cfg.Driver) {
		return NewS3(cfg)
	}
	return NewLocal(cfg), nil
}

// NewLocal merakit storage lokal dari config.
func NewLocal(cfg config.StorageConfig) *Local {
	return &Local{dir: cfg.Dir, urlBase: cfg.URLBase}
}

// ValidateAndSave membaca reader (mis. file upload), MEMVALIDASI + RE-ENCODE
// gambar (magic-byte + sanitasi), lalu menyimpan → URL publik. Helper DRY.
func ValidateAndSave(ctx context.Context, store Storage, r io.Reader) (string, error) {
	data, err := io.ReadAll(io.LimitReader(r, MaxImageBytes+1))
	if err != nil {
		return "", err
	}
	clean, ext, verr := SanitizeImage(data, MaxImageBytes)
	if verr != nil {
		return "", verr
	}
	return store.SaveImage(ctx, clean, ext)
}

// ValidateAndSaveMedia = ValidateAndSave varian editor: validasi+re-encode lalu
// simpan ke folder editor → MediaFile (dipakai file manager rich text).
func ValidateAndSaveMedia(ctx context.Context, store Storage, r io.Reader) (MediaFile, error) {
	data, err := io.ReadAll(io.LimitReader(r, MaxImageBytes+1))
	if err != nil {
		return MediaFile{}, err
	}
	clean, ext, verr := SanitizeImage(data, MaxImageBytes)
	if verr != nil {
		return MediaFile{}, verr
	}
	return store.SaveMedia(ctx, clean, ext)
}

// SaveImage menulis file bernama acak (UUID + ext) lalu mengembalikan URL publik.
func (s *Local) SaveImage(_ context.Context, data []byte, ext string) (string, error) {
	if err := os.MkdirAll(s.dir, 0o755); err != nil {
		return "", err
	}
	name := helpers.NewID() + ext
	if err := os.WriteFile(filepath.Join(s.dir, name), data, 0o644); err != nil {
		return "", err
	}
	return strings.TrimRight(s.urlBase, "/") + "/" + name, nil
}

// mediaFile merakit MediaFile dari nama berkas di folder editor.
func (s *Local) mediaFile(name string) MediaFile {
	return MediaFile{
		Name: name,
		Key:  editorSubdir + "/" + name,
		URL:  strings.TrimRight(s.urlBase, "/") + "/" + editorSubdir + "/" + name,
	}
}

// SaveMedia menyimpan gambar editor ke <dir>/editor/<uuid><ext>.
func (s *Local) SaveMedia(_ context.Context, data []byte, ext string) (MediaFile, error) {
	sub := filepath.Join(s.dir, editorSubdir)
	if err := os.MkdirAll(sub, 0o755); err != nil {
		return MediaFile{}, err
	}
	name := helpers.NewID() + ext
	if err := os.WriteFile(filepath.Join(sub, name), data, 0o644); err != nil {
		return MediaFile{}, err
	}
	return s.mediaFile(name), nil
}

// ListMedia mengembalikan seluruh berkas di folder editor (terbaru tak diurut).
func (s *Local) ListMedia(_ context.Context) ([]MediaFile, error) {
	entries, err := os.ReadDir(filepath.Join(s.dir, editorSubdir))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	out := make([]MediaFile, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() {
			out = append(out, s.mediaFile(e.Name()))
		}
	}
	return out, nil
}

// DeleteMedia menghapus berkas editor by key ("editor/<nama>", divalidasi).
func (s *Local) DeleteMedia(_ context.Context, key string) error {
	name, ok := safeMediaName(key)
	if !ok {
		return ErrInvalidMediaKey
	}
	if err := os.Remove(filepath.Join(s.dir, editorSubdir, name)); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
