package service

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"goadmin/internal/auth"
	apperr "goadmin/internal/errors"
	"goadmin/internal/helpers"
	"goadmin/internal/modules/access/model"
)

// IAuthService = kontrak autentikasi (login/logout API + verifikasi kredensial web).
type IAuthService interface {
	// Login memverifikasi kredensial & menerbitkan JWT (untuk API).
	Login(ctx context.Context, email, password string) (token string, user *model.User, err error)
	// Logout mencabut token (blacklist) hingga kedaluwarsa.
	Logout(ctx context.Context, jti string, expiresAt time.Time) error
	// Authenticate memverifikasi email+password & mengembalikan user (untuk sesi web).
	Authenticate(ctx context.Context, email, password string) (*model.User, error)
	// FindByID memuat user + relasi (dipakai middleware untuk otorisasi).
	FindByID(ctx context.Context, id string) (*model.User, error)
}

// AuthService mengimplementasi IAuthService.
type AuthService struct {
	db        *gorm.DB
	jwt       *auth.JWTManager
	blacklist auth.TokenBlacklist
}

var _ IAuthService = (*AuthService)(nil)

// NewAuthService merakit service auth.
func NewAuthService(db *gorm.DB, jwt *auth.JWTManager, blacklist auth.TokenBlacklist) *AuthService {
	return &AuthService{db: db, jwt: jwt, blacklist: blacklist}
}

func (s *AuthService) Authenticate(ctx context.Context, email, password string) (*model.User, error) {
	var user model.User
	err := s.db.WithContext(ctx).
		Preload("Roles").Preload("Roles.Permissions").
		First(&user, "email = ?", email).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Pesan generik (jangan bocorkan email terdaftar/tidak).
			return nil, apperr.Unauthorized("Email atau password salah")
		}
		return nil, apperr.Internal(err.Error())
	}
	if user.Blocked {
		return nil, apperr.Forbidden("Akun diblokir")
	}
	if !auth.CheckPassword(user.Password, password) {
		return nil, apperr.Unauthorized("Email atau password salah")
	}
	return &user, nil
}

func (s *AuthService) Login(ctx context.Context, email, password string) (string, *model.User, error) {
	user, err := s.Authenticate(ctx, email, password)
	if err != nil {
		return "", nil, err
	}
	jti := helpers.NewID()
	token, _, err := s.jwt.Generate(user.ID, user.Email, jti)
	if err != nil {
		return "", nil, apperr.Internal("gagal menerbitkan token: " + err.Error())
	}
	return token, user, nil
}

func (s *AuthService) Logout(ctx context.Context, jti string, expiresAt time.Time) error {
	ttl := time.Until(expiresAt)
	if ttl <= 0 {
		ttl = time.Minute
	}
	if err := s.blacklist.Revoke(ctx, jti, ttl); err != nil {
		return apperr.Internal("gagal mencabut token: " + err.Error())
	}
	return nil
}

func (s *AuthService) FindByID(ctx context.Context, id string) (*model.User, error) {
	var user model.User
	err := s.db.WithContext(ctx).
		Preload("Roles").Preload("Roles.Permissions").
		First(&user, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperr.Unauthorized("Sesi tidak valid")
		}
		return nil, apperr.Internal(err.Error())
	}
	return &user, nil
}
