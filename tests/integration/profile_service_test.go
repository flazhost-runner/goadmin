package integration

import (
	"context"
	"testing"

	"goadmin/internal/config"
	apperr "goadmin/internal/errors"
	accessdto "goadmin/internal/modules/access/dto"
	accesssvc "goadmin/internal/modules/access/service"
	"goadmin/internal/modules/profile/dto"
	"goadmin/internal/modules/profile/service"
	"goadmin/tests/testutil"
)

// profileEnv mengelompokkan dependency test profil di atas SATU DB in-memory.
type profileEnv struct {
	profiles *service.ProfileService
	users    accesssvc.IUserService
	userID   string
	ctx      context.Context
}

// newProfileEnv menyiapkan user awal + ProfileService (skema access sudah
// di-migrate testutil), semua di DB yang sama.
func newProfileEnv(t *testing.T) profileEnv {
	t.Helper()
	c := testutil.NewContainer(t, config.ModeFull)
	ctx := context.Background()

	userSvc := accesssvc.NewUserService(c.DB, c.Config.Security.BcryptRounds)
	u, err := userSvc.Store(ctx, accessdto.CreateUserInput{
		Name: "Budi", Email: "budi@example.com", Password: "password123",
	}, "")
	if err != nil {
		t.Fatalf("seed user: %v", err)
	}
	return profileEnv{
		profiles: service.NewProfileService(c.DB, c.Config.Security.BcryptRounds),
		users:    userSvc,
		userID:   u.ID,
		ctx:      ctx,
	}
}

func TestProfileService_GetOwn(t *testing.T) {
	e := newProfileEnv(t)

	p, err := e.profiles.Get(e.ctx, e.userID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if p.Email != "budi@example.com" {
		t.Fatalf("email mismatch: %s", p.Email)
	}
}

func TestProfileService_UpdateNameAndPassword(t *testing.T) {
	e := newProfileEnv(t)

	before, _ := e.profiles.Get(e.ctx, e.userID)
	oldHash := before.Password

	updated, err := e.profiles.Update(e.ctx, e.userID, dto.UpdateProfileInput{
		Name:                 "Budi Baru",
		Email:                "budi@example.com",
		Password:             "newpassword1",
		PasswordConfirmation: "newpassword1",
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if updated.Name != "Budi Baru" {
		t.Fatalf("nama tak berubah: %s", updated.Name)
	}
	if updated.Password == oldHash || updated.Password == "newpassword1" {
		t.Fatal("password harus di-hash ulang (bukan lama / plaintext)")
	}
}

// Ganti password tanpa konfirmasi yang cocok → 422.
func TestProfileService_PasswordMismatch(t *testing.T) {
	e := newProfileEnv(t)

	_, err := e.profiles.Update(e.ctx, e.userID, dto.UpdateProfileInput{
		Name:                 "Budi",
		Email:                "budi@example.com",
		Password:             "abcdefgh",
		PasswordConfirmation: "zzzzzzzz",
	})
	ae, ok := apperr.As(err)
	if !ok || ae.Status != 422 {
		t.Fatalf("harusnya AppError 422, dapat: %v", err)
	}
}

// Ambil email user lain → 409.
func TestProfileService_EmailConflict(t *testing.T) {
	e := newProfileEnv(t)

	// User kedua di DB yang sama.
	if _, err := e.users.Store(e.ctx, accessdto.CreateUserInput{
		Name: "Ani", Email: "ani@example.com", Password: "password123",
	}, ""); err != nil {
		t.Fatalf("seed user2: %v", err)
	}

	_, err := e.profiles.Update(e.ctx, e.userID, dto.UpdateProfileInput{
		Name: "Budi", Email: "ani@example.com",
	})
	ae, ok := apperr.As(err)
	if !ok || ae.Status != 409 {
		t.Fatalf("harusnya AppError 409, dapat: %v", err)
	}
}

// Avatar (Picture) tersimpan saat di-update.
func TestProfileService_UpdatePicture(t *testing.T) {
	e := newProfileEnv(t)

	out, err := e.profiles.Update(e.ctx, e.userID, dto.UpdateProfileInput{
		Name: "Budi", Email: "budi@example.com", Picture: "/uploads/abc.png",
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if out.Picture != "/uploads/abc.png" {
		t.Fatalf("avatar tak tersimpan: %q", out.Picture)
	}
}

func TestProfileService_NotFound(t *testing.T) {
	e := newProfileEnv(t)

	_, err := e.profiles.Get(e.ctx, "tidak-ada")
	ae, ok := apperr.As(err)
	if !ok || ae.Status != 404 {
		t.Fatalf("harusnya AppError 404, dapat: %v", err)
	}
}
