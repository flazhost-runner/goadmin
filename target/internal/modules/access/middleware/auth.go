// Package middleware (access) menyediakan guard autentikasi & otorisasi.
// Urutan WAJIB: Authenticated → Authorize (auth dulu, baru cek izin).
package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"

	"goadmin/internal/auth"
	apperr "goadmin/internal/errors"
	"goadmin/internal/modules/access/model"
	"goadmin/internal/modules/access/service"
)

// ctxUserKey adalah kunci penyimpanan user terotentikasi di gin.Context.
const ctxUserKey = "auth_user"

// Guard merangkum dependency middleware auth.
type Guard struct {
	jwt       *auth.JWTManager
	blacklist auth.TokenBlacklist
	auths     service.IAuthService
}

// NewGuard merakit guard.
func NewGuard(jwt *auth.JWTManager, blacklist auth.TokenBlacklist, auths service.IAuthService) *Guard {
	return &Guard{jwt: jwt, blacklist: blacklist, auths: auths}
}

// NewGuardWebOnly merakit guard untuk jalur sesi web (tanpa JWT/blacklist).
func NewGuardWebOnly(auths service.IAuthService) *Guard {
	return &Guard{auths: auths}
}

// AuthenticatedJWT memverifikasi Bearer token: tanda tangan + algoritma + blacklist.
// Inilah jalur yang membuat blacklist BEKERJA NYATA: token yang sudah logout
// (jti tercabut) ditolak 401 walau tanda tangannya masih sah.
func (g *Guard) AuthenticatedJWT() gin.HandlerFunc {
	return func(c *gin.Context) {
		raw := extractBearer(c)
		if raw == "" {
			c.Error(apperr.Unauthorized("Token tidak ada"))
			c.Abort()
			return
		}
		claims, err := g.jwt.Verify(raw)
		if err != nil {
			c.Error(apperr.Unauthorized("Token tidak valid"))
			c.Abort()
			return
		}
		// Cek blacklist (token dicabut saat logout).
		revoked, err := g.blacklist.IsRevoked(c.Request.Context(), claims.ID)
		if err != nil {
			c.Error(apperr.Internal("gagal cek blacklist: " + err.Error()))
			c.Abort()
			return
		}
		if revoked {
			c.Error(apperr.Unauthorized("Token sudah dicabut"))
			c.Abort()
			return
		}
		user, err := g.auths.FindByID(c.Request.Context(), claims.UserID)
		if err != nil {
			c.Error(err)
			c.Abort()
			return
		}
		// Simpan user + claims (jti & exp dipakai endpoint logout).
		c.Set(ctxUserKey, user)
		c.Set("jwt_claims", claims)
		c.Next()
	}
}

func extractBearer(c *gin.Context) string {
	h := c.GetHeader("Authorization")
	if h == "" {
		return ""
	}
	parts := strings.SplitN(h, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return strings.TrimSpace(parts[1])
}

// UserFrom mengambil user terotentikasi dari context (nil bila tak ada).
func UserFrom(c *gin.Context) *model.User {
	v, ok := c.Get(ctxUserKey)
	if !ok {
		return nil
	}
	u, _ := v.(*model.User)
	return u
}
