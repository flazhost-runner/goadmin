package integration

import (
	"context"
	"testing"

	"goadmin/internal/config"
	homesvc "goadmin/internal/modules/home/service"
	settingdto "goadmin/internal/modules/setting/dto"
	settingmig "goadmin/internal/modules/setting/migration"
	settingsvc "goadmin/internal/modules/setting/service"
	"goadmin/tests/testutil"
)

// newHomeSvc menyiapkan HomeService dengan provider setting nyata (DB in-memory).
func newHomeSvc(t *testing.T) (*homesvc.HomeService, *settingsvc.SettingService, context.Context) {
	t.Helper()
	c := testutil.NewContainer(t, config.ModeFull)
	if err := settingmig.AutoMigrate(c.DB); err != nil {
		t.Fatalf("migrate setting: %v", err)
	}
	ssvc := settingsvc.NewSettingService(c.DB)
	home := homesvc.NewHomeService(func() settingsvc.ISettingService { return ssvc })
	return home, ssvc, context.Background()
}

// Tanpa setting tersimpan → default singleton (GoAdmin / tema Blue) terikat ke landing.
func TestHomeService_LandingDefault(t *testing.T) {
	home, _, ctx := newHomeSvc(t)

	ld, err := home.Landing(ctx)
	if err != nil {
		t.Fatalf("landing: %v", err)
	}
	if ld.AppName != "GoAdmin" {
		t.Fatalf("app name: harap GoAdmin, dapat %s", ld.AppName)
	}
	if ld.ThemeName != "Blue" || ld.Primary != "#3B82F6" {
		t.Fatalf("tema default salah: %s / %s", ld.ThemeName, ld.Primary)
	}
}

// Setting di-update → landing mengikuti (nama + warna tema).
func TestHomeService_LandingBindsSetting(t *testing.T) {
	home, ssvc, ctx := newHomeSvc(t)

	if _, err := ssvc.Update(ctx, settingdto.UpdateSettingInput{
		Name: "Toko Saya", Theme: "Green", Email: "halo@toko.com",
	}, ""); err != nil {
		t.Fatalf("update setting: %v", err)
	}

	ld, err := home.Landing(ctx)
	if err != nil {
		t.Fatalf("landing: %v", err)
	}
	if ld.AppName != "Toko Saya" {
		t.Fatalf("nama tak terikat: %s", ld.AppName)
	}
	if ld.ThemeName != "Green" || ld.Primary != "#10B981" {
		t.Fatalf("tema tak terikat: %s / %s", ld.ThemeName, ld.Primary)
	}
	if ld.Email != "halo@toko.com" {
		t.Fatalf("email tak terikat: %s", ld.Email)
	}
}

// Provider mengembalikan nil (setting belum siap) → fallback default, tanpa error.
func TestHomeService_LandingFallbackWhenSettingNil(t *testing.T) {
	home := homesvc.NewHomeService(func() settingsvc.ISettingService { return nil })

	ld, err := home.Landing(context.Background())
	if err != nil {
		t.Fatalf("landing fallback error: %v", err)
	}
	if ld.AppName != "GoAdmin" || ld.Primary == "" {
		t.Fatalf("fallback salah: %+v", ld)
	}
}
