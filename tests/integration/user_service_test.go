package integration

import (
	"context"
	"testing"

	apperr "goadmin/internal/errors"
	"goadmin/internal/config"
	"goadmin/internal/modules/access/dto"
	"goadmin/internal/modules/access/service"
	"goadmin/tests/testutil"
)

// Integration: UserService ↔ DB (SQLite in-memory) — membuktikan portabilitas.
func TestUserService_StoreAndShow(t *testing.T) {
	c := testutil.NewContainer(t, config.ModeFull)
	svc := service.NewUserService(c.DB, c.Config.Security.BcryptRounds)
	ctx := context.Background()

	user, err := svc.Store(ctx, dto.CreateUserInput{
		Name:     "Budi",
		Email:    "budi@example.com",
		Password: "password123",
	}, "tester")
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	if user.ID == "" || user.Code == "" {
		t.Fatalf("id/code kosong: %+v", user)
	}

	got, err := svc.Show(ctx, user.ID)
	if err != nil {
		t.Fatalf("show: %v", err)
	}
	if got.Email != "budi@example.com" {
		t.Fatalf("email mismatch: %s", got.Email)
	}
	// Password harus ter-hash (bukan plaintext).
	if got.Password == "password123" {
		t.Fatal("password tersimpan plaintext — wajib bcrypt")
	}
}

func TestUserService_DuplicateEmailConflict(t *testing.T) {
	c := testutil.NewContainer(t, config.ModeFull)
	svc := service.NewUserService(c.DB, c.Config.Security.BcryptRounds)
	ctx := context.Background()

	in := dto.CreateUserInput{Name: "A", Email: "dup@example.com", Password: "password123"}
	if _, err := svc.Store(ctx, in, ""); err != nil {
		t.Fatalf("store pertama: %v", err)
	}
	_, err := svc.Store(ctx, in, "")
	if err == nil {
		t.Fatal("email duplikat seharusnya ditolak")
	}
	ae, ok := apperr.As(err)
	if !ok || ae.Status != 409 {
		t.Fatalf("harusnya AppError 409, dapat: %v", err)
	}
}

func TestUserService_ShowNotFound(t *testing.T) {
	c := testutil.NewContainer(t, config.ModeFull)
	svc := service.NewUserService(c.DB, c.Config.Security.BcryptRounds)

	_, err := svc.Show(context.Background(), "tidak-ada")
	ae, ok := apperr.As(err)
	if !ok || ae.Status != 404 {
		t.Fatalf("harusnya AppError 404, dapat: %v", err)
	}
}

// Search case-insensitive portabel (ciLike) — bekerja di SQLite.
func TestUserService_SearchCaseInsensitive(t *testing.T) {
	c := testutil.NewContainer(t, config.ModeFull)
	svc := service.NewUserService(c.DB, c.Config.Security.BcryptRounds)
	ctx := context.Background()

	_, _ = svc.Store(ctx, dto.CreateUserInput{Name: "Charlie", Email: "charlie@example.com", Password: "password123"}, "")

	res, err := svc.Index(ctx, dto.ListQuery{Search: "CHAR"})
	if err != nil {
		t.Fatalf("index: %v", err)
	}
	if len(res.Data) != 1 {
		t.Fatalf("ciLike gagal: harap 1, dapat %d", len(res.Data))
	}
}
