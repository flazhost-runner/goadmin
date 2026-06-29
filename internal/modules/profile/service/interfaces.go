// Package service berisi logika bisnis modul profile (self-service user
// terautentikasi). Beroperasi pada entity access.User dengan kontrak SEMPIT:
// hanya field profil + ganti password — bukan status/role (least-privilege).
package service

import (
	"context"

	model "goadmin/internal/modules/access/model"
	"goadmin/internal/modules/profile/dto"
)

// IProfileService = kontrak profil milik-sendiri.
type IProfileService interface {
	// Get mengambil profil user (404 bila tak ada).
	Get(ctx context.Context, userID string) (*model.User, error)
	// Update menimpa field profil + password opsional (status/role tak tersentuh).
	Update(ctx context.Context, userID string, in dto.UpdateProfileInput) (*model.User, error)
}
