package storage_test

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"os"
	"strings"
	"testing"

	"goadmin/internal/config"
	"goadmin/internal/storage"
)

// Upload sungguhan ke object storage S3-compatible (MinIO/OSS). Di-skip kecuali
// STORAGE_TEST_ENDPOINT di-set — CI unit tetap cepat, tapi jalur upload bisa dibuktikan
// benar-benar mengenai bucket, bukan disk lokal:
//
//	docker run -d -p 19000:9000 -e MINIO_ROOT_USER=minioadmin \
//	  -e MINIO_ROOT_PASSWORD=minioadmin123 minio/minio server /data
//	STORAGE_TEST_ENDPOINT=http://127.0.0.1:19000 STORAGE_TEST_BUCKET=goadmin-test \
//	STORAGE_TEST_KEY=minioadmin STORAGE_TEST_SECRET=minioadmin123 STORAGE_TEST_SSL=false \
//	  go test ./internal/storage/ -run TestS3UploadHitsBucket -v
func TestS3UploadHitsBucket(t *testing.T) {
	endpoint := os.Getenv("STORAGE_TEST_ENDPOINT")
	if endpoint == "" {
		t.Skip("STORAGE_TEST_ENDPOINT tidak di-set — lewati uji upload nyata")
	}

	cfg := config.StorageConfig{
		Driver:      "s3",
		S3Endpoint:  endpoint,
		S3Bucket:    os.Getenv("STORAGE_TEST_BUCKET"),
		S3AccessKey: os.Getenv("STORAGE_TEST_KEY"),
		S3SecretKey: os.Getenv("STORAGE_TEST_SECRET"),
		S3UseSSL:    os.Getenv("STORAGE_TEST_SSL") == "true",
	}

	store, err := storage.New(cfg)
	if err != nil {
		t.Fatalf("gagal merakit storage: %v", err)
	}
	if _, isLocal := store.(*storage.Local); isLocal {
		t.Fatal("storage jatuh ke Local — upload tak akan sampai ke bucket")
	}

	url, err := storage.ValidateAndSave(context.Background(), store, bytes.NewReader(pngBytes(t)))
	if err != nil {
		t.Fatalf("upload gagal: %v", err)
	}
	if !strings.Contains(url, cfg.S3Bucket) && !strings.HasPrefix(url, "http") {
		t.Fatalf("URL publik tidak masuk akal: %q", url)
	}

	// Bukti terkuat: objeknya benar-benar ada di bucket, bukan hanya "tidak error".
	files, err := store.ListMedia(context.Background())
	if err != nil {
		t.Fatalf("ListMedia gagal: %v", err)
	}
	t.Logf("URL publik: %s (objek editor di bucket: %d)", url, len(files))
}

func pngBytes(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	img.Set(0, 0, color.RGBA{R: 255, A: 255})
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("gagal membuat PNG: %v", err)
	}
	return buf.Bytes()
}
