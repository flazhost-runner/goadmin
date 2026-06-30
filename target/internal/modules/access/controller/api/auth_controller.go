// Package api berisi controller REST modul access. Controller TIPIS: parse
// input + panggil service + format respons. Error diteruskan via c.Error()
// ke middleware terpusat (tanpa try/catch manual / mapping status di sini).
package api

import (
	"github.com/gin-gonic/gin"

	"goadmin/internal/auth"
	apperr "goadmin/internal/errors"
	"goadmin/internal/helpers"
	"goadmin/internal/modules/access/dto"
	accessmw "goadmin/internal/modules/access/middleware"
	"goadmin/internal/modules/access/service"
)

// AuthController menangani auth API (JWT): login/logout/me + register + reset
// password OTP (paritas penuh dengan web — konsumen API-only bisa daftar & reset).
type AuthController struct {
	auths service.IAuthService
	users service.IUserService
	reset service.IPasswordResetService
}

// NewAuthController merakit controller (service di-inject, bukan di-new di sini).
func NewAuthController(auths service.IAuthService, users service.IUserService, reset service.IPasswordResetService) *AuthController {
	return &AuthController{auths: auths, users: users, reset: reset}
}

type loginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// Login → POST /api/v1/auth/login.
func (ctl *AuthController) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperr.Validation("Input tidak valid", nil))
		return
	}
	token, user, err := ctl.auths.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		c.Error(err)
		return
	}
	helpers.OK(c, "Login berhasil", gin.H{"token": token, "user": user})
}

// Logout → POST /api/v1/auth/logout (butuh AuthenticatedJWT).
// Mencabut token saat ini (blacklist) → akses berikutnya 401.
func (ctl *AuthController) Logout(c *gin.Context) {
	v, ok := c.Get("jwt_claims")
	if !ok {
		c.Error(apperr.Unauthorized("Token tidak ada"))
		return
	}
	claims, ok := v.(*auth.Claims)
	if !ok || claims.ExpiresAt == nil {
		c.Error(apperr.Unauthorized("Token tidak valid"))
		return
	}
	if err := ctl.auths.Logout(c.Request.Context(), claims.ID, claims.ExpiresAt.Time); err != nil {
		c.Error(err)
		return
	}
	helpers.OK(c, "Logout berhasil", nil)
}

// Me → GET /api/v1/auth/me (profil user terautentikasi).
func (ctl *AuthController) Me(c *gin.Context) {
	user := accessmw.UserFrom(c)
	if user == nil {
		c.Error(apperr.Unauthorized("Belum terautentikasi"))
		return
	}
	helpers.OK(c, "OK", user)
}

// --- Register + Reset password OTP (publik; kembar dgn jalur web) ---

type registerRequest struct {
	Name     string `json:"name" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
}

// Register → POST /api/v1/auth/register (publik; rate-limited).
func (ctl *AuthController) Register(c *gin.Context) {
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperr.Validation("Input tidak valid", nil))
		return
	}
	user, err := ctl.users.Store(c.Request.Context(), dto.CreateUserInput{
		Name: req.Name, Email: req.Email, Password: req.Password, Status: "Active",
	}, "")
	if err != nil {
		c.Error(err)
		return
	}
	helpers.Created(c, "Akun berhasil dibuat", user)
}

type resetRequestRequest struct {
	Email string `json:"email" binding:"required,email"`
}

// ResetRequest → POST /api/v1/auth/reset/request (publik; kirim OTP ke email).
// Tak membocorkan keberadaan email — selalu "sukses" dari sisi pemanggil.
func (ctl *AuthController) ResetRequest(c *gin.Context) {
	var req resetRequestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperr.Validation("Input tidak valid", nil))
		return
	}
	if err := ctl.reset.RequestReset(c.Request.Context(), req.Email); err != nil {
		c.Error(err)
		return
	}
	helpers.OK(c, "Jika email terdaftar, kode OTP telah dikirim.", nil)
}

type resetProcessRequest struct {
	Email    string `json:"email" binding:"required,email"`
	OTP      string `json:"otp" binding:"required"`
	Password string `json:"password" binding:"required,min=8"`
}

// ResetProcess → POST /api/v1/auth/reset/process (publik; verifikasi OTP + set password).
func (ctl *AuthController) ResetProcess(c *gin.Context) {
	var req resetProcessRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperr.Validation("Input tidak valid", nil))
		return
	}
	if err := ctl.reset.Reset(c.Request.Context(), req.Email, req.OTP, req.Password); err != nil {
		c.Error(err)
		return
	}
	helpers.OK(c, "Password berhasil direset.", nil)
}
