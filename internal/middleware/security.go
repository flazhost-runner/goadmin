package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/unrolled/secure"

	"goadmin/internal/config"
)

// SecurityHeaders memasang header keamanan (padanan helmet di NodeAdmin):
// HSTS, X-Frame-Options, X-Content-Type-Options, Referrer-Policy, dll.
// HSTS hanya diaktifkan di production (hindari memaksa HTTPS di dev lokal).
func SecurityHeaders(cfg *config.Config) gin.HandlerFunc {
	sec := secure.New(secure.Options{
		FrameDeny:             true,
		ContentTypeNosniff:    true,
		BrowserXssFilter:      true,
		ReferrerPolicy:        "strict-origin-when-cross-origin",
		STSSeconds:            stsSeconds(cfg.IsProd),
		STSIncludeSubdomains:  cfg.IsProd,
		STSPreload:            cfg.IsProd,
		ForceSTSHeader:        cfg.IsProd,
		IsDevelopment:         !cfg.IsProd,
	})
	return func(c *gin.Context) {
		if err := sec.Process(c.Writer, c.Request); err != nil {
			c.Abort()
			return
		}
		c.Next()
	}
}

func stsSeconds(isProd bool) int64 {
	if isProd {
		return 31536000 // 1 tahun
	}
	return 0
}
