package service

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	apperr "goadmin/internal/errors"
	"goadmin/internal/helpers"
	"goadmin/internal/modules/setting/dto"
	"goadmin/internal/modules/setting/model"
	"goadmin/internal/modules/setting/theme"
)

// cacheTTL = masa berlaku cache setting (dibaca tiap request). Pendek + ada
// invalidasi saat update → perubahan tampil instan, beban query minimal.
const cacheTTL = 60 * time.Second

// SettingService mengimplementasi ISettingService. DB di-inject lewat konstruktor;
// cache dirakit internal (per-instance).
type SettingService struct {
	db    *gorm.DB
	cache *settingCache
}

// Pastikan kontrak terpenuhi saat compile.
var _ ISettingService = (*SettingService)(nil)

// NewSettingService merakit service + cache.
func NewSettingService(db *gorm.DB) *SettingService {
	return &SettingService{db: db, cache: newSettingCache(cacheTTL)}
}

// Get mengembalikan setting singleton (dari cache; muat/buat default bila perlu).
func (s *SettingService) Get(ctx context.Context) (*model.Setting, error) {
	if cached := s.cache.get(); cached != nil {
		return cached, nil
	}
	setting, err := s.loadOrCreate(ctx)
	if err != nil {
		return nil, err
	}
	s.cache.set(setting)
	return setting, nil
}

// Update menimpa field non-kosong dari input, simpan, lalu invalidasi & hangatkan cache.
func (s *SettingService) Update(ctx context.Context, in dto.UpdateSettingInput, actorID string) (*model.Setting, error) {
	// Validasi tema (domain) sebelum menyentuh DB.
	if in.Theme != "" && !theme.IsValid(in.Theme) {
		return nil, apperr.Validation("Tema tidak dikenal", map[string]string{"theme": "pilih dari katalog tema"})
	}

	setting, err := s.loadOrCreate(ctx)
	if err != nil {
		return nil, err
	}

	applyUpdate(setting, in)
	setting.UpdatedBy = actorID

	if err := s.db.WithContext(ctx).Save(setting).Error; err != nil {
		return nil, apperr.Internal(err.Error())
	}

	// Invalidasi lalu hangatkan ulang dari DB (state kanonik + cache segar).
	s.cache.invalidate()
	fresh, err := s.loadOrCreate(ctx)
	if err != nil {
		return nil, err
	}
	s.cache.set(fresh)
	return fresh, nil
}

// Themes mengembalikan katalog palet (theme switcher).
func (s *SettingService) Themes() []theme.Theme { return theme.All() }

// loadOrCreate mengambil baris setting tunggal; membuat default bila tabel kosong.
func (s *SettingService) loadOrCreate(ctx context.Context) (*model.Setting, error) {
	var setting model.Setting
	err := s.db.WithContext(ctx).First(&setting).Error
	if err == nil {
		return &setting, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, apperr.Internal(err.Error())
	}

	// Belum ada → buat singleton default.
	setting = model.Setting{
		ID:         helpers.NewID(),
		Initial:    "GA",
		Name:       "GoAdmin",
		Copyright:  "© GoAdmin",
		Theme:      theme.Default,
		FeTemplate: "",
	}
	if err := s.db.WithContext(ctx).Create(&setting).Error; err != nil {
		return nil, apperr.Internal(err.Error())
	}
	return &setting, nil
}

// applyUpdate menimpa hanya field non-kosong (update parsial).
func applyUpdate(s *model.Setting, in dto.UpdateSettingInput) {
	if in.Initial != "" {
		s.Initial = in.Initial
	}
	if in.Name != "" {
		s.Name = in.Name
	}
	if in.Description != "" {
		// Deskripsi = HTML rich-text (Trumbowyg) → sanitasi sebelum simpan (anti XSS),
		// padanan cleanRichText NodeAdmin. Disimpan & dirender mentah di landing.
		s.Description = helpers.CleanRichText(in.Description)
	}
	if in.Phone != "" {
		s.Phone = in.Phone
	}
	if in.Address != "" {
		s.Address = in.Address
	}
	if in.Email != "" {
		s.Email = in.Email
	}
	if in.Copyright != "" {
		s.Copyright = in.Copyright
	}
	if in.Icon != "" {
		s.Icon = in.Icon
	}
	if in.Logo != "" {
		s.Logo = in.Logo
	}
	if in.LoginImage != "" {
		s.LoginImage = in.LoginImage
	}
	if in.Theme != "" {
		s.Theme = in.Theme
	}
	if in.FeTemplate != "" {
		s.FeTemplate = in.FeTemplate
	}
}
