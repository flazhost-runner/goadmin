package api_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"goadmin/internal/app"
	"goadmin/internal/config"
	"goadmin/tests/testutil"

	// Side-effect: daftarkan modul access ke registry (init()).
	_ "goadmin/internal/modules/access"
)

// TestJWTBlacklist_RealFlow menguji blacklist token SECARA NYATA lewat HTTP:
// login → akses 200 → logout → akses 401. Memakai MemoryBlacklist yang
// BERPERILAKU seperti runtime (TTL), bukan mock yang selalu mulus.
//
// Ini menutup pelajaran NodeAdmin: blacklist bisa lolos test tapi gagal di
// produksi bila store-test tak setia perilaku. Di sini store yang dipakai app
// = store yang diuji (API identik).
func TestJWTBlacklist_RealFlow(t *testing.T) {
	c := testutil.NewContainer(t, config.ModeAPI)
	email, password := testutil.SeedAdmin(t, c)
	engine := app.Build(c)

	// 1. Login → dapat token.
	token := doLogin(t, engine, email, password)

	// 2. Akses terproteksi dengan token → 200.
	if code := doMe(engine, token); code != http.StatusOK {
		t.Fatalf("akses pra-logout harus 200, dapat %d", code)
	}

	// 3. Logout (cabut token).
	logoutCode := doLogout(engine, token)
	if logoutCode != http.StatusOK {
		t.Fatalf("logout harus 200, dapat %d", logoutCode)
	}

	// 4. Akses lagi dengan token yang sama → HARUS 401 (token tercabut).
	if code := doMe(engine, token); code != http.StatusUnauthorized {
		t.Fatalf("akses pasca-logout harus 401 (blacklist), dapat %d", code)
	}
}

func doLogin(t *testing.T, engine http.Handler, email, password string) string {
	t.Helper()
	body, _ := json.Marshal(map[string]string{"email": email, "password": password})
	req := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("login gagal: %d — %s", w.Code, w.Body.String())
	}
	var resp struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode login: %v", err)
	}
	if resp.Data.Token == "" {
		t.Fatal("token kosong")
	}
	return resp.Data.Token
}

func doMe(engine http.Handler, token string) int {
	req := httptest.NewRequest("GET", "/api/v1/auth/me", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)
	return w.Code
}

func doLogout(engine http.Handler, token string) int {
	req := httptest.NewRequest("POST", "/api/v1/auth/logout", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)
	return w.Code
}
