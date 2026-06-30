package integration

import (
	"context"
	"testing"

	"goadmin/internal/config"
	apperr "goadmin/internal/errors"
	"goadmin/internal/modules/setting/dto"
	settingmig "goadmin/internal/modules/setting/migration"
	"goadmin/internal/modules/setting/service"
	"goadmin/tests/testutil"
)

func newSettingSvc(t *testing.T) (*service.SettingService, context.Context) {
	t.Helper()
	c := testutil.NewContainer(t, config.ModeFull)
	if err := settingmig.AutoMigrate(c.DB); err != nil {
		t.Fatalf("migrate setting: %v", err)
	}
	return service.NewSettingService(c.DB), context.Background()
}

// Get pada tabel kosong → membuat default singleton (Name GoAdmin, Theme Blue).
func TestSettingService_GetCreatesDefault(t *testing.T) {
	svc, ctx := newSettingSvc(t)

	s, err := svc.Get(ctx)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if s.ID == "" {
		t.Fatal("id default kosong")
	}
	if s.Name != "GoAdmin" || s.Theme != "Blue" {
		t.Fatalf("default tak sesuai: name=%s theme=%s", s.Name, s.Theme)
	}
}

// Get berulang TIDAK membuat baris baru (singleton).
func TestSettingService_GetIsSingleton(t *testing.T) {
	svc, ctx := newSettingSvc(t)

	first, _ := svc.Get(ctx)
	second, _ := svc.Get(ctx)
	if first.ID != second.ID {
		t.Fatalf("singleton dilanggar: %s != %s", first.ID, second.ID)
	}
}

// Update menimpa field non-kosong & invalidasi cache → Get berikutnya tampak baru.
func TestSettingService_UpdateInvalidatesCache(t *testing.T) {
	svc, ctx := newSettingSvc(t)

	// Hangatkan cache dengan Get awal.
	if _, err := svc.Get(ctx); err != nil {
		t.Fatalf("get awal: %v", err)
	}

	updated, err := svc.Update(ctx, dto.UpdateSettingInput{Name: "Toko Saya", Theme: "Green"}, "tester")
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if updated.Name != "Toko Saya" || updated.Theme != "Green" {
		t.Fatalf("update tak diterapkan: %+v", updated)
	}

	// Get setelah update HARUS mencerminkan nilai baru (cache ter-invalidasi).
	got, err := svc.Get(ctx)
	if err != nil {
		t.Fatalf("get setelah update: %v", err)
	}
	if got.Name != "Toko Saya" || got.Theme != "Green" {
		t.Fatalf("cache basi: %+v", got)
	}
	if got.UpdatedBy != "tester" {
		t.Fatalf("updated_by tak terisi: %s", got.UpdatedBy)
	}
}

// Update parsial: field kosong tidak menghapus nilai lama.
func TestSettingService_PartialUpdateKeepsExisting(t *testing.T) {
	svc, ctx := newSettingSvc(t)

	if _, err := svc.Update(ctx, dto.UpdateSettingInput{Name: "Awal", Email: "a@b.com"}, ""); err != nil {
		t.Fatalf("update1: %v", err)
	}
	// Update kedua hanya theme — name & email harus tetap.
	out, err := svc.Update(ctx, dto.UpdateSettingInput{Theme: "Purple"}, "")
	if err != nil {
		t.Fatalf("update2: %v", err)
	}
	if out.Name != "Awal" || out.Email != "a@b.com" || out.Theme != "Purple" {
		t.Fatalf("partial update salah: %+v", out)
	}
}

// Tema di luar katalog ditolak 422.
func TestSettingService_InvalidThemeRejected(t *testing.T) {
	svc, ctx := newSettingSvc(t)

	_, err := svc.Update(ctx, dto.UpdateSettingInput{Theme: "Neon"}, "")
	ae, ok := apperr.As(err)
	if !ok || ae.Status != 422 {
		t.Fatalf("harusnya AppError 422, dapat: %v", err)
	}
}
