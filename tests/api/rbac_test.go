package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"goadmin/internal/app"
	"goadmin/internal/config"
	"goadmin/internal/modules/access/dto"
	"goadmin/internal/modules/access/service"
	"goadmin/tests/testutil"

	_ "goadmin/internal/modules/access"
)

// RBAC: user tanpa permission → 403; admin → 200. Urutan middleware
// authenticated → authorize dibuktikan (401 bila tanpa token, 403 bila tanpa izin).
func TestRBAC_ForbiddenWithoutPermission(t *testing.T) {
	c := testutil.NewContainer(t, config.ModeAPI)
	adminEmail, adminPass := testutil.SeedAdmin(t, c)

	// Buat user biasa TANPA role (tak punya izin apa pun).
	userSvc := service.NewUserService(c.DB, c.Config.Security.BcryptRounds)
	_, err := userSvc.Store(context.Background(), dto.CreateUserInput{
		Name: "Plain", Email: "plain@example.com", Password: "password123",
	}, "")
	if err != nil {
		t.Fatalf("buat user: %v", err)
	}

	engine := app.Build(c)

	// 1. Tanpa token → 401.
	if code := getUsers(engine, ""); code != http.StatusUnauthorized {
		t.Fatalf("tanpa token harus 401, dapat %d", code)
	}

	// 2. User biasa (login) → akses /users → 403 (tak punya user.view).
	plainToken := doLogin(t, engine, "plain@example.com", "password123")
	if code := getUsers(engine, plainToken); code != http.StatusForbidden {
		t.Fatalf("user tanpa izin harus 403, dapat %d", code)
	}

	// 3. Admin → 200 (bypass / punya semua izin).
	adminToken := doLogin(t, engine, adminEmail, adminPass)
	if code := getUsers(engine, adminToken); code != http.StatusOK {
		t.Fatalf("admin harus 200, dapat %d", code)
	}
}

// Mass-assignment: field tak dikenal di body diabaikan (DTO whitelist).
func TestMassAssignment_UnknownFieldIgnored(t *testing.T) {
	c := testutil.NewContainer(t, config.ModeAPI)
	adminEmail, adminPass := testutil.SeedAdmin(t, c)
	engine := app.Build(c)
	token := doLogin(t, engine, adminEmail, adminPass)

	// Sertakan field "code" & "created_by" yang TIDAK ada di DTO → harus diabaikan
	// (code di-generate server; created_by diisi dari actor, bukan body).
	body, _ := json.Marshal(map[string]interface{}{
		"name": "Eve", "email": "eve@example.com", "password": "password123",
		"code": "HACK-001", "created_by": "HACK-USER",
	})
	req := httptest.NewRequest("POST", "/api/v1/access/user/store", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("buat user harus 201, dapat %d — %s", w.Code, w.Body.String())
	}
	var resp struct {
		Data struct {
			Code      string `json:"code"`
			CreatedBy string `json:"created_by"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Data.Code == "HACK-001" {
		t.Fatal("field 'code' tak boleh di-inject lewat mass-assignment")
	}
	if resp.Data.CreatedBy == "HACK-USER" {
		t.Fatal("field 'created_by' tak boleh di-inject lewat mass-assignment")
	}
}

func getUsers(engine http.Handler, token string) int {
	req := httptest.NewRequest("GET", "/api/v1/access/user", nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)
	return w.Code
}
