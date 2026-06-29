package service

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"goadmin/internal/auth"
	apperr "goadmin/internal/errors"
	"goadmin/internal/helpers"
	"goadmin/internal/mail"
	"goadmin/internal/modules/access/model"
)

// PasswordResetService menangani reset password via OTP email. OTP disimpan
// TER-HASH (bcrypt) + expiry; plaintext hanya dikirim via email.
type PasswordResetService struct {
	db               *gorm.DB
	mailer           mail.Mailer
	bcryptRounds     int
	appName          string
	otpExpiryMinutes int
}

// Pastikan kontrak terpenuhi saat compile.
var _ IPasswordResetService = (*PasswordResetService)(nil)

// NewPasswordResetService merakit service.
// otpExpiryMinutes dibaca dari OTP_EXPIRY_MINUTES env (default 10).
func NewPasswordResetService(db *gorm.DB, mailer mail.Mailer, bcryptRounds int, appName string, otpExpiryMinutes int) *PasswordResetService {
	if otpExpiryMinutes <= 0 {
		otpExpiryMinutes = 10
	}
	return &PasswordResetService{db: db, mailer: mailer, bcryptRounds: bcryptRounds, appName: appName, otpExpiryMinutes: otpExpiryMinutes}
}

// RequestReset membuat OTP, menyimpan hash+expiry, dan mengirimkannya via email.
// Bila email tak terdaftar → tetap return nil (jangan bocorkan keberadaan akun).
func (s *PasswordResetService) RequestReset(ctx context.Context, email string) error {
	var user model.User
	err := s.db.WithContext(ctx).First(&user, "email = ?", email).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil // diam-diam (anti user-enumeration)
		}
		return apperr.Internal(err.Error())
	}

	otp := helpers.NewNumericOTP(6)
	hash, herr := auth.HashPassword(otp, s.bcryptRounds)
	if herr != nil {
		return apperr.Internal("gagal hash OTP: " + herr.Error())
	}
	expires := time.Now().Add(time.Duration(s.otpExpiryMinutes) * time.Minute).UnixMilli()
	user.PasswordOTP = hash
	user.PasswordOTPExpires = &expires
	if err := s.db.WithContext(ctx).Model(&user).
		Select("PasswordOTP", "PasswordOTPExpires").Updates(&user).Error; err != nil {
		return apperr.Internal(err.Error())
	}

	msg := mail.Message{
		To:      email,
		Subject: "Reset Password " + s.appName,
		Body: "Kode OTP reset password Anda: " + otp +
			"\nBerlaku 15 menit. Abaikan email ini bila Anda tidak memintanya.",
	}
	if err := s.mailer.Send(ctx, msg); err != nil {
		return apperr.Internal("gagal kirim email: " + err.Error())
	}
	return nil
}

// Reset memverifikasi OTP lalu menyetel password baru (dan menghapus OTP).
func (s *PasswordResetService) Reset(ctx context.Context, email, otp, newPassword string) error {
	if len(newPassword) < 8 {
		return apperr.Validation("Password minimum 8 characters.", map[string]string{"password": "minimum 8 characters"})
	}

	var user model.User
	err := s.db.WithContext(ctx).First(&user, "email = ?", email).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apperr.Unauthorized("Email atau OTP salah")
		}
		return apperr.Internal(err.Error())
	}

	if user.PasswordOTP == "" || user.PasswordOTPExpires == nil {
		return apperr.Unauthorized("OTP is invalid.")
	}
	if time.Now().UnixMilli() > *user.PasswordOTPExpires {
		return apperr.Unauthorized("OTP has expired.")
	}
	if !auth.CheckPassword(user.PasswordOTP, otp) {
		return apperr.Unauthorized("OTP is invalid.")
	}

	hash, herr := auth.HashPassword(newPassword, s.bcryptRounds)
	if herr != nil {
		return apperr.Internal("gagal hash password: " + herr.Error())
	}
	user.Password = hash
	user.PasswordOTP = ""
	user.PasswordOTPExpires = nil
	if err := s.db.WithContext(ctx).Model(&user).
		Select("Password", "PasswordOTP", "PasswordOTPExpires").Updates(&user).Error; err != nil {
		return apperr.Internal(err.Error())
	}
	return nil
}
