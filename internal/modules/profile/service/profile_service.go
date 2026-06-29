package service

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"goadmin/internal/auth"
	apperr "goadmin/internal/errors"
	model "goadmin/internal/modules/access/model"
	"goadmin/internal/modules/profile/dto"
)

// ProfileService mengimplementasi IProfileService. DB + bcrypt cost di-inject.
type ProfileService struct {
	db           *gorm.DB
	bcryptRounds int
}

// Pastikan kontrak terpenuhi saat compile.
var _ IProfileService = (*ProfileService)(nil)

// NewProfileService merakit service.
func NewProfileService(db *gorm.DB, bcryptRounds int) *ProfileService {
	return &ProfileService{db: db, bcryptRounds: bcryptRounds}
}

// Get mengambil profil user (dengan rolenya, untuk tampilan).
func (s *ProfileService) Get(ctx context.Context, userID string) (*model.User, error) {
	var user model.User
	if err := s.db.WithContext(ctx).Preload("Roles").First(&user, "id = ?", userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperr.NotFound("Profil tidak ditemukan")
		}
		return nil, apperr.Internal(err.Error())
	}
	return &user, nil
}

// Update menimpa field profil + password opsional. Status & role TIDAK disentuh
// (least-privilege: user tak bisa mempromosikan dirinya). Relasi Roles di-Omit
// agar tak ikut ter-upsert saat Save.
func (s *ProfileService) Update(ctx context.Context, userID string, in dto.UpdateProfileInput) (*model.User, error) {
	// Konfirmasi password bila user mengganti password.
	if in.Password != "" && in.Password != in.PasswordConfirmation {
		return nil, apperr.Validation("Konfirmasi password tidak cocok",
			map[string]string{"password_confirmation": "harus sama dengan password"})
	}

	user, err := s.Get(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Email unik (selain milik sendiri).
	if in.Email != user.Email {
		var count int64
		if err := s.db.WithContext(ctx).Model(&model.User{}).
			Where("email = ? AND id <> ?", in.Email, userID).Count(&count).Error; err != nil {
			return nil, apperr.Internal(err.Error())
		}
		if count > 0 {
			return nil, apperr.Conflict("Email sudah terpakai")
		}
	}

	user.Name = in.Name
	user.Email = in.Email
	user.Phone = in.Phone
	if in.Timezone != "" {
		user.Timezone = in.Timezone
	}
	if in.Picture != "" {
		user.Picture = in.Picture
	}
	user.UpdatedBy = userID

	// Password hanya diubah bila diisi.
	if in.Password != "" {
		hash, herr := auth.HashPassword(in.Password, s.bcryptRounds)
		if herr != nil {
			return nil, apperr.Internal("gagal hash password: " + herr.Error())
		}
		user.Password = hash
	}

	if err := s.db.WithContext(ctx).Omit("Roles").Save(user).Error; err != nil {
		return nil, apperr.Internal(err.Error())
	}
	return s.Get(ctx, userID)
}
