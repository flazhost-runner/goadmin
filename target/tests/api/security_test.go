package api_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"

	"goadmin/internal/middleware"
)

// Rate limiter: maks N per jendela per-key; hit ke-(N+1) ditolak, key lain bebas.
func TestRateLimiter_Allow(t *testing.T) {
	rl := middleware.NewRateLimiter(2, time.Minute)

	if !rl.Allow("1.1.1.1") || !rl.Allow("1.1.1.1") {
		t.Fatal("2 hit pertama harus lolos")
	}
	if rl.Allow("1.1.1.1") {
		t.Fatal("hit ke-3 harus ditolak")
	}
	if !rl.Allow("2.2.2.2") {
		t.Fatal("IP berbeda tak boleh terpengaruh")
	}
}

func csrfEngine() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(middleware.ErrorHandler(false, false)) // JSON (tanpa template) → 403
	store := cookie.NewStore([]byte("test-secret"))
	r.Use(sessions.Sessions("s", store))
	r.Use(middleware.CSRF())
	r.GET("/form", func(c *gin.Context) { c.String(http.StatusOK, c.GetString("csrf_token")) })
	r.POST("/submit", func(c *gin.Context) { c.String(http.StatusOK, "ok") })
	return r
}

// POST tanpa token → 403.
func TestCSRF_BlocksWithoutToken(t *testing.T) {
	r := csrfEngine()
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/submit", nil))
	if w.Code != http.StatusForbidden {
		t.Fatalf("POST tanpa token harus 403, dapat %d", w.Code)
	}
}

// GET menyetel token+cookie; POST dengan token valid → 200; token salah → 403.
func TestCSRF_TokenFlow(t *testing.T) {
	r := csrfEngine()

	wGet := httptest.NewRecorder()
	r.ServeHTTP(wGet, httptest.NewRequest(http.MethodGet, "/form", nil))
	token := strings.TrimSpace(wGet.Body.String())
	cookieHdr := wGet.Header().Get("Set-Cookie")
	if token == "" || cookieHdr == "" {
		t.Fatalf("token/cookie kosong: %q / %q", token, cookieHdr)
	}

	post := func(csrf string) int {
		form := url.Values{"_csrf": {csrf}}
		req := httptest.NewRequest(http.MethodPost, "/submit", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Cookie", cookieHdr)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w.Code
	}

	if code := post(token); code != http.StatusOK {
		t.Fatalf("POST token valid harus 200, dapat %d", code)
	}
	if code := post("token-salah"); code != http.StatusForbidden {
		t.Fatalf("POST token salah harus 403, dapat %d", code)
	}
}
