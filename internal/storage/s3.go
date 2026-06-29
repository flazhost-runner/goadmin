package storage

import (
	"bytes"
	"context"
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

// NewS3 merakit storage S3 dari config. Tidak menyentuh jaringan (klien lazy;
// koneksi terjadi saat PutObject pertama).
func NewS3(cfg config.StorageConfig) (*S3, error) {
	cli, err := minio.New(cfg.S3Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.S3AccessKey, cfg.S3SecretKey, ""),
		Secure: cfg.S3UseSSL,
		Region: cfg.S3Region,
	})
	if err != nil {
		return nil, err
	}
	return &S3{client: cli, bucket: cfg.S3Bucket, publicURL: cfg.S3PublicURL}, nil
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
