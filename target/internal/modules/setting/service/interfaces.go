// Package service berisi logika bisnis modul setting. Service mengembalikan
// *errors.AppError saat gagal (BUKAN error telanjang) & mengimplementasi
// ISettingService (Dependency Inversion).
package service

import (
	"context"

	"goadmin/internal/modules/setting/dto"
	"goadmin/internal/modules/setting/model"
	"goadmin/internal/modules/setting/theme"
)

// ISettingService = kontrak setting global (singleton + ber-cache).
type ISettingService interface {
	// Get mengembalikan setting tunggal (dibuat default bila belum ada), dari cache.
	Get(ctx context.Context) (*model.Setting, error)
	// Update menimpa field non-kosong + invalidasi cache.
	Update(ctx context.Context, in dto.UpdateSettingInput, actorID string) (*model.Setting, error)
	// Themes mengembalikan katalog palet (theme switcher).
	Themes() []theme.Theme
}
