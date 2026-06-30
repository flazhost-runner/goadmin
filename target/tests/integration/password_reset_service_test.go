package integration

import (
	"context"
	"regexp"
	"testing"

	"goadmin/internal/auth"
	"goadmin/internal/config"
	"goadmin/internal/container"
	apperr "goadmin/internal/errors"
	"goadmin/internal/mail"
	accessdto "goadmin/internal/modules/access/dto"
	accessmodel "goadmin/internal/modules/access/model"
	accesssvc "goadmin/internal/modules/access/service"
	"goadmin/tests/testutil"
)

const resetEmail = "budi@example.com"

// captureMailer = Mailer nyata (fidelity) yang menangkap email terkirim → test
// bisa mengambil OTP plaintext.
type captureMailer struct {
	sent []mail.Message
}

func (m *captureMailer) Send(_ context.Context, msg mail.Message) error {
	m.sent = append(m.sent, msg)
	return nil
}

var otpRe = regexp.MustCompile(`\d{6}`)

func newResetEnv(t *testing.T) (*accesssvc.PasswordResetService, *captureMailer, *container.Container, context.Context) {
	t.Helper()
	c := testutil.NewContainer(t, config.ModeFull)
	ctx := context.Background()

	userSvc := accesssvc.NewUserService(c.DB, c.Config.Security.BcryptRounds)
	if _, err := userSvc.Store(ctx, accessdto.CreateUserInput{
		Name: "Budi", Email: resetEmail, Password: "oldpassword1",
	}, ""); err != nil {
		t.Fatalf("seed user: %v", err)
	}
	mailer := &captureMailer{}
	svc := accesssvc.NewPasswordResetService(c.DB, mailer, c.Config.Security.BcryptRounds, "GoAdmin", c.Config.Security.OTPExpiryMinutes)
	return svc, mailer, c, ctx
}

// RequestReset (email terdaftar) → email memuat OTP 6 digit.
func TestPasswordReset_RequestSendsOTP(t *testing.T) {
	svc, mailer, _, ctx := newResetEnv(t)

	if err := svc.RequestReset(ctx, resetEmail); err != nil {
		t.Fatalf("request: %v", err)
	}
	if len(mailer.sent) != 1 || mailer.sent[0].To != resetEmail {
		t.Fatalf("email OTP tak terkirim: %+v", mailer.sent)
	}
	if otpRe.FindString(mailer.sent[0].Body) == "" {
		t.Fatalf("body tak memuat OTP 6 digit: %q", mailer.sent[0].Body)
	}
}

// Email tak terdaftar → nil + TANPA kirim email (anti user-enumeration).
func TestPasswordReset_UnknownEmailSilent(t *testing.T) {
	svc, mailer, _, ctx := newResetEnv(t)

	if err := svc.RequestReset(ctx, "tidakada@example.com"); err != nil {
		t.Fatalf("harus nil, dapat: %v", err)
	}
	if len(mailer.sent) != 0 {
		t.Fatal("tak boleh kirim email untuk akun tak terdaftar")
	}
}

// Alur lengkap: request → OTP salah ditolak → OTP benar → password berganti & OTP dibersihkan.
func TestPasswordReset_FullFlow(t *testing.T) {
	svc, mailer, c, ctx := newResetEnv(t)

	if err := svc.RequestReset(ctx, resetEmail); err != nil {
		t.Fatalf("request: %v", err)
	}
	otp := otpRe.FindString(mailer.sent[0].Body)

	if err := svc.Reset(ctx, resetEmail, "000000", "newpassword1"); err == nil {
		t.Fatal("OTP salah harus ditolak")
	}
	if err := svc.Reset(ctx, resetEmail, otp, "newpassword1"); err != nil {
		t.Fatalf("reset OTP benar: %v", err)
	}

	var user accessmodel.User
	if err := c.DB.First(&user, "email = ?", resetEmail).Error; err != nil {
		t.Fatalf("reload user: %v", err)
	}
	if !auth.CheckPassword(user.Password, "newpassword1") {
		t.Fatal("password baru tak aktif")
	}
	if auth.CheckPassword(user.Password, "oldpassword1") {
		t.Fatal("password lama masih berlaku")
	}
	if user.PasswordOTP != "" || user.PasswordOTPExpires != nil {
		t.Fatal("OTP harus dibersihkan setelah reset")
	}

	// OTP sekali pakai: dipakai ulang → ditolak.
	if err := svc.Reset(ctx, resetEmail, otp, "another12345"); err == nil {
		t.Fatal("OTP bekas harus tak bisa dipakai lagi")
	}
}

// OTP kedaluwarsa → 401.
func TestPasswordReset_Expired(t *testing.T) {
	svc, mailer, c, ctx := newResetEnv(t)

	if err := svc.RequestReset(ctx, resetEmail); err != nil {
		t.Fatalf("request: %v", err)
	}
	otp := otpRe.FindString(mailer.sent[0].Body)

	// Paksa expiry ke masa lalu.
	var user accessmodel.User
	if err := c.DB.First(&user, "email = ?", resetEmail).Error; err != nil {
		t.Fatalf("reload: %v", err)
	}
	past := int64(1)
	user.PasswordOTPExpires = &past
	if err := c.DB.Model(&user).Select("PasswordOTPExpires").Updates(&user).Error; err != nil {
		t.Fatalf("set expiry: %v", err)
	}

	err := svc.Reset(ctx, resetEmail, otp, "newpassword1")
	ae, ok := apperr.As(err)
	if !ok || ae.Status != 401 {
		t.Fatalf("harus 401 kedaluwarsa, dapat: %v", err)
	}
}
