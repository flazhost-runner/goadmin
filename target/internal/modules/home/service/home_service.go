package service

import (
	"context"

	"goadmin/internal/modules/home/fetemplate"
	settingsvc "goadmin/internal/modules/setting/service"
	"goadmin/internal/modules/setting/theme"
)

// Fallback default landing bila Setting kosong (atau service setting belum siap).
const (
	defaultAppName     = "GoAdmin"
	defaultDescription = "Panel admin siap pakai — kelola pengguna, peran, dan pengaturan."
	defaultCopyright   = "© GoAdmin"
)

// HomeService membangun view-model landing dari Setting global.
//
// Setting di-resolve LAZY lewat provider-func (bukan di-inject saat konstruksi):
// modul home terdaftar SEBELUM setting (urut nama), sehingga setting service
// belum tersedia saat Register. Provider dipanggil saat request (setelah semua
// modul terdaftar) → selalu mendapat instance ber-cache yang sama.
type HomeService struct {
	settings func() settingsvc.ISettingService
}

// Pastikan kontrak terpenuhi saat compile.
var _ IHomeService = (*HomeService)(nil)

// NewHomeService merakit service dengan provider setting (lazy).
func NewHomeService(settings func() settingsvc.ISettingService) *HomeService {
	return &HomeService{settings: settings}
}

// Landing membangun view-model landing. Bila setting service belum siap / kosong,
// memakai nilai default (landing tetap tampil — fallback aman).
func (s *HomeService) Landing(ctx context.Context) (LandingData, error) {
	data := LandingData{
		AppName:     defaultAppName,
		Description: defaultDescription,
		Copyright:   defaultCopyright,
		ThemeName:   theme.Default,
		Template:    fetemplate.DefaultSlug,
	}
	pal := theme.ByName(theme.Default)
	data.Primary, data.Accent = pal.Primary, pal.Dark

	svc := s.settings()
	if svc == nil {
		return data, nil // fallback: setting belum siap
	}

	setting, err := svc.Get(ctx)
	if err != nil {
		return LandingData{}, err
	}

	if setting.Name != "" {
		data.AppName = setting.Name
	}
	if setting.Description != "" {
		data.Description = setting.Description
	}
	if setting.Copyright != "" {
		data.Copyright = setting.Copyright
	}
	data.Logo = setting.Logo
	data.Email = setting.Email
	data.Phone = setting.Phone
	data.Address = setting.Address

	if setting.Theme != "" {
		data.ThemeName = setting.Theme
		pal = theme.ByName(setting.Theme)
		data.Primary, data.Accent = pal.Primary, pal.Dark
	}
	data.Template = fetemplate.ResolveActive(setting.FeTemplate)
	return data, nil
}
