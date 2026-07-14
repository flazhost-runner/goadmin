package storage

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"goadmin/internal/config"
	"goadmin/internal/helpers"
)

// S3 menyimpan ke object storage S3-compatible (AWS S3 / Aliyun OSS / MinIO)
// lewat minio-go. URL publik = S3PublicURL + key.
type S3 struct {
	client    *minio.Client
	bucket    string
	publicURL string
}

// NewS3 merakit storage S3-compatible (driver "s3" atau "oss") dari config. Tidak
// menyentuh jaringan (klien lazy; koneksi terjadi saat PutObject pertama).
func NewS3(cfg config.StorageConfig) (*S3, error) {
	if err := validateObjectConfig(cfg); err != nil {
		return nil, err
	}

	host := stripScheme(cfg.S3Endpoint)
	isOSS := strings.EqualFold(cfg.Driver, "oss")

	opts := &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.S3AccessKey, cfg.S3SecretKey, ""),
		Secure: cfg.S3UseSSL,
		Region: cfg.S3Region,
	}
	if isOSS {
		// Aliyun OSS hanya melayani virtual-hosted (bucket.endpoint/key). Auto-detect
		// minio-go memilih path-style untuk endpoint tak dikenal → OSS menolaknya.
		opts.BucketLookup = minio.BucketLookupDNS
	}

	cli, err := minio.New(host, opts)
	if err != nil {
		return nil, fmt.Errorf("storage: gagal init driver %s: %w", cfg.Driver, err)
	}
	return &S3{
		client:    cli,
		bucket:    cfg.S3Bucket,
		publicURL: publicBaseURL(cfg, host, isOSS),
	}, nil
}

// validateObjectConfig menolak config object storage yang tak lengkap. Tanpa ini,
// kredensial kosong baru ketahuan saat upload pertama (403 dari bucket) — atau lebih
// buruk: dulu ditelan lalu jatuh ke disk lokal.
func validateObjectConfig(cfg config.StorageConfig) error {
	missing := make([]string, 0, 4)
	if cfg.S3Endpoint == "" {
		missing = append(missing, "STORAGE_ENDPOINT")
	}
	if cfg.S3Bucket == "" {
		missing = append(missing, "STORAGE_BUCKET")
	}
	if cfg.S3AccessKey == "" {
		missing = append(missing, "STORAGE_ACCESS_KEY_ID")
	}
	if cfg.S3SecretKey == "" {
		missing = append(missing, "STORAGE_SECRET_ACCESS_KEY")
	}
	if len(missing) > 0 {
		return fmt.Errorf("storage: driver %q butuh %s", cfg.Driver, strings.Join(missing, ", "))
	}
	return nil
}

// publicBaseURL menentukan base URL objek. STORAGE_PUBLIC_URL menang bila diisi (mis. CDN).
// Bila kosong, diturunkan dari driver — dulu base URL kosong menghasilkan URL relatif
// "/images/x.webp" yang tak pernah bisa dibuka.
func publicBaseURL(cfg config.StorageConfig, host string, isOSS bool) string {
	if cfg.S3PublicURL != "" {
		return strings.TrimRight(cfg.S3PublicURL, "/")
	}
	scheme := "http"
	if cfg.S3UseSSL {
		scheme = "https"
	}
	if isOSS {
		// OSS virtual-hosted: bucket.endpoint/key (paritas NodeAdmin fileService).
		return fmt.Sprintf("%s://%s.%s", scheme, cfg.S3Bucket, host)
	}
	// S3-compatible dengan endpoint eksplisit (MinIO/R2): path-style endpoint/bucket/key.
	return fmt.Sprintf("%s://%s/%s", scheme, host, cfg.S3Bucket)
}

func stripScheme(endpoint string) string {
	e := strings.TrimPrefix(endpoint, "https://")
	return strings.TrimPrefix(e, "http://")
}

// SaveImage meng-upload byte gambar (key acak) → URL publik.
func (s *S3) SaveImage(ctx context.Context, data []byte, ext string) (string, error) {
	key := "images/" + helpers.NewID() + ext
	_, err := s.client.PutObject(ctx, s.bucket, key, bytes.NewReader(data), int64(len(data)),
		minio.PutObjectOptions{ContentType: contentType(ext)})
	if err != nil {
		return "", err
	}
	return strings.TrimRight(s.publicURL, "/") + "/" + key, nil
}

// SaveMedia meng-upload gambar editor ke prefix "editor/" → MediaFile.
func (s *S3) SaveMedia(ctx context.Context, data []byte, ext string) (MediaFile, error) {
	name := helpers.NewID() + ext
	key := editorSubdir + "/" + name
	if _, err := s.client.PutObject(ctx, s.bucket, key, bytes.NewReader(data), int64(len(data)),
		minio.PutObjectOptions{ContentType: contentType(ext)}); err != nil {
		return MediaFile{}, err
	}
	return MediaFile{Name: name, Key: key, URL: strings.TrimRight(s.publicURL, "/") + "/" + key}, nil
}

// ListMedia mendaftar objek di prefix "editor/".
func (s *S3) ListMedia(ctx context.Context) ([]MediaFile, error) {
	var out []MediaFile
	for obj := range s.client.ListObjects(ctx, s.bucket, minio.ListObjectsOptions{Prefix: editorSubdir + "/", Recursive: true}) {
		if obj.Err != nil {
			return nil, obj.Err
		}
		name := obj.Key[strings.LastIndex(obj.Key, "/")+1:]
		out = append(out, MediaFile{Name: name, Key: obj.Key, URL: strings.TrimRight(s.publicURL, "/") + "/" + obj.Key})
	}
	return out, nil
}

// DeleteMedia menghapus objek editor by key (divalidasi).
func (s *S3) DeleteMedia(ctx context.Context, key string) error {
	if _, ok := safeMediaName(key); !ok {
		return ErrInvalidMediaKey
	}
	return s.client.RemoveObject(ctx, s.bucket, key, minio.RemoveObjectOptions{})
}

func contentType(ext string) string {
	switch ext {
	case ".jpg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	default:
		return "application/octet-stream"
	}
}
