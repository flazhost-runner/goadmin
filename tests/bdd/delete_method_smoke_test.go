package bdd

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"goadmin/internal/app"
	"goadmin/internal/bootstrap"
	"goadmin/internal/config"
	"goadmin/internal/container"
	"goadmin/internal/database"
	"goadmin/internal/helpers"
	"goadmin/internal/middleware"
	accessdto "goadmin/internal/modules/access/dto"
	"goadmin/internal/modules/access/model"
	accesssvc "goadmin/internal/modules/access/service"
)

var csrfMetaRe = regexp.MustCompile(`name="csrf-token" content="([^"]+)"`)

// TestDeleteMethodEndToEnd memverifikasi delete user lewat HTTP method DELETE
// (form POST + ?_method=DELETE) MELALUI wrapper MethodOverride + CSRF + RBAC —
// persis jalur runtime (main.go membungkus engine di level http.Server).
func TestDeleteMethodEndToEnd(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Template & aset di-load via path relatif repo-root; test berjalan dari
	// tests/bdd → chdir ke root agar view ter-parse (mode full butuh template web).
	wd, _ := os.Getwd()
	if err := os.Chdir("../.."); err != nil {
		t.Fatalf("chdir root: %v", err)
	}
	defer os.Chdir(wd)

	cfg := &config.Config{
		Env: "test", IsTest: true,
		App:      config.AppConfig{Name: "GoAdmin Test", Mode: config.ModeFull},
		DB:       config.DBConfig{Type: "sqlite"},
		Session:  config.SessionConfig{Secret: "test-session-secret"},
		JWT:      config.JWTConfig{Secret: "test-jwt", Algorithm: "HS256"},
		Security: config.SecurityConfig{BcryptRounds: 4},
	}
	db, err := database.OpenSQLiteMemory(t.Name())
	if err != nil {
		t.Fatalf("db: %v", err)
	}
	if err := bootstrap.MigrateAndSeed(db, "admin@goadmin.test", "secret123", cfg.Security.BcryptRounds); err != nil {
		t.Fatalf("seed: %v", err)
	}
	c := container.MustNew(cfg, db, nil)

	// Bungkus engine seperti runtime: MethodOverride SEBELUM routing.
	handler := middleware.MethodOverride(app.Build(c))

	// User korban (bukan admin sendiri — template sembunyikan delete untuk self).
	victim := model.User{
		ID: helpers.NewID(), Code: helpers.NewCode("U"),
		Name: "Victim", Email: "victim@goadmin.test",
		Password: "x", Status: model.StatusActive, Timezone: "UTC",
	}
	if err := db.Create(&victim).Error; err != nil {
		t.Fatalf("create victim: %v", err)
	}

	jar := map[string]string{}
	do := func(method, path string, form url.Values) *httptest.ResponseRecorder {
		var body *strings.Reader
		if form != nil {
			body = strings.NewReader(form.Encode())
		} else {
			body = strings.NewReader("")
		}
		req := httptest.NewRequest(method, path, body)
		if form != nil {
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		}
		var cookies []string
		for k, v := range jar {
			cookies = append(cookies, k+"="+v)
		}
		if len(cookies) > 0 {
			req.Header.Set("Cookie", strings.Join(cookies, "; "))
		}
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		for _, ck := range rec.Result().Cookies() {
			jar[ck.Name] = ck.Value
		}
		return rec
	}

	csrfOf := func(rec *httptest.ResponseRecorder) string {
		m := csrfMetaRe.FindStringSubmatch(rec.Body.String())
		if len(m) < 2 {
			// Halaman login pakai partial berbeda; ambil dari hidden input.
			re := regexp.MustCompile(`name="_csrf" value="([^"]+)"`)
			if mm := re.FindStringSubmatch(rec.Body.String()); len(mm) >= 2 {
				return mm[1]
			}
			t.Fatalf("csrf token tak ditemukan di body")
		}
		return m[1]
	}

	// 1) GET login → token + cookie sesi.
	loginPage := do(http.MethodGet, "/auth/login", nil)
	if loginPage.Code != http.StatusOK {
		t.Fatalf("GET login = %d", loginPage.Code)
	}
	tok := csrfOf(loginPage)

	// 2) POST login.
	lr := do(http.MethodPost, "/auth/login", url.Values{
		"email": {"admin@goadmin.test"}, "password": {"secret123"}, "_csrf": {tok},
	})
	if lr.Code != http.StatusFound && lr.Code != http.StatusSeeOther {
		t.Fatalf("POST login = %d (body: %s)", lr.Code, lr.Body.String())
	}

	// 3) GET daftar user → token segar. Sidebar (chrome) ter-render via FuncMap
	// hasAccess — admin bypass → menu User tampil.
	list := do(http.MethodGet, "/admin/v1/access/user", nil)
	if list.Code != http.StatusOK {
		t.Fatalf("GET user list = %d", list.Code)
	}
	tok = csrfOf(list)
	if !strings.Contains(list.Body.String(), "/user/"+victim.ID+"/delete?_method=DELETE") {
		t.Fatalf("form delete DELETE tak ada di daftar user")
	}
	if !strings.Contains(list.Body.String(), "/admin/v1/access/user") {
		t.Fatalf("menu sidebar User (hasAccess admin bypass) tak ter-render")
	}

	// 3b) Buka halaman Permission → memicu SyncFromRoutes lazy (route-driven).
	if pg := do(http.MethodGet, "/admin/v1/access/permission", nil); pg.Code != http.StatusOK {
		t.Fatalf("GET permission page = %d", pg.Code)
	}
	var permCnt int64
	db.Model(&model.Permission{}).
		Where("name = ? AND method = ? AND guard_name = ?", "admin.v1.access.user.delete", "DELETE", "web").
		Count(&permCnt)
	if permCnt != 1 {
		t.Fatalf("lazy sync gagal: permission web admin.v1.access.user.delete/DELETE = %d (harap 1)", permCnt)
	}

	// 4) Hapus victim via POST + ?_method=DELETE (jalur override). _csrf di QUERY
	// karena net/http tak mem-parse body form untuk DELETE.
	del := do(http.MethodPost, "/admin/v1/access/user/"+victim.ID+"/delete?_method=DELETE&_csrf="+url.QueryEscape(tok), nil)
	if del.Code != http.StatusFound && del.Code != http.StatusSeeOther {
		t.Fatalf("DELETE user = %d (body: %s)", del.Code, del.Body.String())
	}

	// 5) Verifikasi terhapus.
	var cnt int64
	db.Model(&model.User{}).Where("id = ?", victim.ID).Count(&cnt)
	if cnt != 0 {
		t.Fatalf("victim masih ada (count=%d) — delete via DELETE gagal", cnt)
	}

	// 6) Kontrol negatif: POST tanpa ?_method tetap bukan DELETE → tidak menghapus.
	victim2 := model.User{ID: helpers.NewID(), Code: helpers.NewCode("U"), Name: "V2", Email: "v2@goadmin.test", Password: "x", Status: model.StatusActive, Timezone: "UTC"}
	db.Create(&victim2)
	noOverride := do(http.MethodPost, "/admin/v1/access/user/"+victim2.ID+"/delete", url.Values{"_csrf": {tok}})
	if noOverride.Code == http.StatusFound || noOverride.Code == http.StatusSeeOther {
		t.Fatalf("POST tanpa override seharusnya tidak match route DELETE (dapat %d)", noOverride.Code)
	}
	db.Model(&model.User{}).Where("id = ?", victim2.ID).Count(&cnt)
	if cnt != 1 {
		t.Fatalf("victim2 terhapus tanpa override — route delete bocor ke POST")
	}
}

// TestApiDeleteMethodEndToEnd memverifikasi endpoint API delete = DELETE asli
// (JWT, TANPA ?_method override) pada path simetris web: `/api/v1/access/
// {resource}/:id/delete` (nama api.v1.access.*.delete), BUKAN REST `DELETE /:id`.
func TestApiDeleteMethodEndToEnd(t *testing.T) {
	w, err := newWorld() // mode API (tanpa template web)
	if err != nil {
		t.Fatalf("world: %v", err)
	}
	if err := w.login(adminEmail, adminPass); err != nil {
		t.Fatalf("login: %v", err)
	}

	victim := model.User{
		ID: helpers.NewID(), Code: helpers.NewCode("U"),
		Name: "ApiVictim", Email: "apivictim@goadmin.test",
		Password: "x", Status: model.StatusActive, Timezone: "UTC",
	}
	if err := w.cont.DB.Create(&victim).Error; err != nil {
		t.Fatalf("create victim: %v", err)
	}

	// DELETE asli + Bearer JWT pada path baru.
	w.request(http.MethodDelete, "/api/v1/access/user/"+victim.ID+"/delete", true)
	if w.status != http.StatusOK {
		t.Fatalf("API DELETE = %d (harap 200)", w.status)
	}
	var cnt int64
	w.cont.DB.Model(&model.User{}).Where("id = ?", victim.ID).Count(&cnt)
	if cnt != 0 {
		t.Fatalf("victim API masih ada (count=%d)", cnt)
	}

	// Kontrol negatif: REST lama `DELETE /:id` (.destroy) sudah TIDAK terdaftar → 404.
	victim2 := model.User{ID: helpers.NewID(), Code: helpers.NewCode("U"), Name: "ApiV2", Email: "apiv2@goadmin.test", Password: "x", Status: model.StatusActive, Timezone: "UTC"}
	w.cont.DB.Create(&victim2)
	w.request(http.MethodDelete, "/api/v1/access/user/"+victim2.ID, true)
	if w.status != http.StatusNotFound {
		t.Fatalf("REST lama DELETE /:id seharusnya 404, dapat %d", w.status)
	}
	w.cont.DB.Model(&model.User{}).Where("id = ?", victim2.ID).Count(&cnt)
	if cnt != 1 {
		t.Fatalf("victim2 terhapus via path REST lama — route bocor")
	}
}

// TestApiVerbosePathsEndToEnd memverifikasi seluruh CRUD API access memakai path
// VERBOSE persis NodeAdmin: store `/store`, edit `/:id/edit`, update `/:id/update`,
// delete_selected `/delete_selected`; dan path REST lama (POST “, GET `/:id`) → 404.
func TestApiVerbosePathsEndToEnd(t *testing.T) {
	w, err := newWorld()
	if err != nil {
		t.Fatalf("world: %v", err)
	}
	if err := w.login(adminEmail, adminPass); err != nil {
		t.Fatalf("login: %v", err)
	}

	jsonReq := func(method, path string, body any) *httptest.ResponseRecorder {
		var rdr *bytes.Reader
		if body != nil {
			b, _ := json.Marshal(body)
			rdr = bytes.NewReader(b)
		} else {
			rdr = bytes.NewReader(nil)
		}
		req := httptest.NewRequest(method, path, rdr)
		req.Header.Set("Authorization", "Bearer "+w.token)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		w.engine.ServeHTTP(rec, req)
		return rec
	}

	// store → POST /store (201).
	st := jsonReq(http.MethodPost, "/api/v1/access/user/store", map[string]any{
		"name": "Verbose", "email": "verbose@goadmin.test", "password": "password123",
	})
	if st.Code != http.StatusCreated {
		t.Fatalf("store = %d (harap 201): %s", st.Code, st.Body.String())
	}
	var created struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(st.Body.Bytes(), &created)
	id := created.Data.ID
	if id == "" {
		t.Fatalf("id user kosong")
	}

	// edit → GET /:id/edit (200).
	if ed := jsonReq(http.MethodGet, "/api/v1/access/user/"+id+"/edit", nil); ed.Code != http.StatusOK {
		t.Fatalf("edit = %d (harap 200)", ed.Code)
	}
	// update → PUT /:id/update (200).
	if up := jsonReq(http.MethodPut, "/api/v1/access/user/"+id+"/update", map[string]any{
		"name": "VerboseEdited", "email": "verbose@goadmin.test",
	}); up.Code != http.StatusOK {
		t.Fatalf("update = %d (harap 200): %s", up.Code, up.Body.String())
	}

	// delete_selected → POST /delete_selected (200) + benar-benar terhapus.
	if ds := jsonReq(http.MethodPost, "/api/v1/access/user/delete_selected", map[string]any{
		"selected": []string{id},
	}); ds.Code != http.StatusOK {
		t.Fatalf("delete_selected = %d (harap 200): %s", ds.Code, ds.Body.String())
	}
	var cnt int64
	w.cont.DB.Model(&model.User{}).Where("id = ?", id).Count(&cnt)
	if cnt != 0 {
		t.Fatalf("user tidak terhapus via delete_selected (count=%d)", cnt)
	}

	// Kontrol negatif: path REST lama sudah TIDAK terdaftar → 404.
	if r := jsonReq(http.MethodPost, "/api/v1/access/user", map[string]any{"name": "X"}); r.Code != http.StatusNotFound {
		t.Fatalf("POST REST lama `` seharusnya 404, dapat %d", r.Code)
	}
	if r := jsonReq(http.MethodGet, "/api/v1/access/user/"+id, nil); r.Code != http.StatusNotFound {
		t.Fatalf("GET REST lama `/:id` seharusnya 404, dapat %d", r.Code)
	}
}

// TestRBACGrantByRouteName memverifikasi model RBAC ROUTE-DRIVEN (a la NodeAdmin):
// role non-admin yang DIBERI permission {nama-route, method} tepat bisa mengakses
// route itu (200), tapi DITOLAK (403) untuk route lain yang tak diberikan. Ini
// membuktikan middleware menurunkan nama route dari (method, FullPath) lalu
// mencocokkan name+method — bukan sekadar menolak semua non-admin.
func TestRBACGrantByRouteName(t *testing.T) {
	w, err := newWorld()
	if err != nil {
		t.Fatalf("world: %v", err)
	}
	db := w.cont.DB

	// Permission untuk GET daftar user (api) — nama persis yang diturunkan
	// NameByMethodPath("GET", "/api/v1/access/user").
	perm := model.Permission{ID: helpers.NewID(), Name: "api.v1.access.user.index", Method: "GET", GuardName: "api", Status: model.StatusActive}
	if err := db.Create(&perm).Error; err != nil {
		t.Fatalf("perm: %v", err)
	}
	role := model.Role{ID: helpers.NewID(), Name: "Viewer", GuardName: "api", Status: model.StatusActive}
	if err := db.Create(&role).Error; err != nil {
		t.Fatalf("role: %v", err)
	}
	if err := db.Model(&role).Association("Permissions").Append(&perm); err != nil {
		t.Fatalf("assign perm: %v", err)
	}

	svc := accesssvc.NewUserService(db, w.cont.Config.Security.BcryptRounds)
	u, err := svc.Store(context.Background(), accessdto.CreateUserInput{Name: "Viewer", Email: "viewer@example.com", Password: "password123"}, "")
	if err != nil {
		t.Fatalf("user: %v", err)
	}
	if err := db.Model(u).Association("Roles").Append(&role); err != nil {
		t.Fatalf("assign role: %v", err)
	}

	if err := w.login("viewer@example.com", "password123"); err != nil {
		t.Fatalf("login: %v", err)
	}

	// Diberi izin GET daftar user → 200.
	w.request(http.MethodGet, "/api/v1/access/user", true)
	if w.status != http.StatusOK {
		t.Fatalf("GET user list (granted) = %d, harap 200", w.status)
	}
	// TIDAK diberi izin delete → 403 (membuktikan granularitas per route+method).
	w.request(http.MethodDelete, "/api/v1/access/user/"+u.ID+"/delete", true)
	if w.status != http.StatusForbidden {
		t.Fatalf("DELETE (tanpa izin) = %d, harap 403", w.status)
	}
}

// TestSyncPermissionsFromRoutes membuktikan permission DITURUNKAN OTOMATIS dari
// named-route registry (route-driven, a la NodeAdmin getAllRegisteredRoute):
// nama = nama-route, method = HTTP method, guard = prefix nama (api.→api). Idempoten.
func TestSyncPermissionsFromRoutes(t *testing.T) {
	w, err := newWorld() // app.Build mengisi registry route
	if err != nil {
		t.Fatalf("world: %v", err)
	}
	db := w.cont.DB
	if err := bootstrap.SyncPermissions(db); err != nil {
		t.Fatalf("sync: %v", err)
	}

	// Permission api diturunkan dari route (name+method+guard tepat).
	var cnt int64
	db.Model(&model.Permission{}).
		Where("name = ? AND method = ? AND guard_name = ?", "api.v1.access.user.delete", "DELETE", "api").
		Count(&cnt)
	if cnt != 1 {
		t.Fatalf("permission api.v1.access.user.delete/DELETE/api harusnya 1, dapat %d", cnt)
	}

	// Idempoten: sync ulang TIDAK menggandakan record.
	var before int64
	db.Model(&model.Permission{}).Count(&before)
	if err := bootstrap.SyncPermissions(db); err != nil {
		t.Fatalf("sync ulang: %v", err)
	}
	var after int64
	db.Model(&model.Permission{}).Count(&after)
	if after != before {
		t.Fatalf("sync tak idempoten: %d → %d", before, after)
	}
}

// TestRolePermissionManagement memverifikasi fitur kelola-permission per-role
// (persis NodeAdmin): menu "Permission" di dropdown Role, halaman assign/unassign,
// assign single (GET) + unassign bulk (POST) yang benar-benar mengubah join table.
func TestRolePermissionManagement(t *testing.T) {
	gin.SetMode(gin.TestMode)
	wd, _ := os.Getwd()
	if err := os.Chdir("../.."); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(wd)

	cfg := &config.Config{
		Env: "test", IsTest: true,
		App:      config.AppConfig{Name: "GoAdmin Test", Mode: config.ModeFull},
		DB:       config.DBConfig{Type: "sqlite"},
		Session:  config.SessionConfig{Secret: "test-session-secret"},
		JWT:      config.JWTConfig{Secret: "test-jwt", Algorithm: "HS256"},
		Security: config.SecurityConfig{BcryptRounds: 4},
	}
	db, err := database.OpenSQLiteMemory(t.Name())
	if err != nil {
		t.Fatalf("db: %v", err)
	}
	if err := bootstrap.MigrateAndSeed(db, "admin@goadmin.test", "secret123", cfg.Security.BcryptRounds); err != nil {
		t.Fatalf("seed: %v", err)
	}
	c := container.MustNew(cfg, db, nil)
	handler := middleware.MethodOverride(app.Build(c))

	role := model.Role{ID: helpers.NewID(), Name: "Editor", GuardName: "web", Status: model.StatusActive}
	if err := db.Create(&role).Error; err != nil {
		t.Fatalf("role: %v", err)
	}
	perm := model.Permission{ID: helpers.NewID(), Name: "admin.v1.access.user.index", Method: "GET", GuardName: "web", Status: model.StatusActive}
	if err := db.Create(&perm).Error; err != nil {
		t.Fatalf("perm: %v", err)
	}

	jar := map[string]string{}
	do := func(method, path string, form url.Values) *httptest.ResponseRecorder {
		var body *strings.Reader
		if form != nil {
			body = strings.NewReader(form.Encode())
		} else {
			body = strings.NewReader("")
		}
		req := httptest.NewRequest(method, path, body)
		if form != nil {
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		}
		var cookies []string
		for k, v := range jar {
			cookies = append(cookies, k+"="+v)
		}
		if len(cookies) > 0 {
			req.Header.Set("Cookie", strings.Join(cookies, "; "))
		}
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		for _, ck := range rec.Result().Cookies() {
			jar[ck.Name] = ck.Value
		}
		return rec
	}
	csrfOf := func(rec *httptest.ResponseRecorder) string {
		if m := csrfMetaRe.FindStringSubmatch(rec.Body.String()); len(m) >= 2 {
			return m[1]
		}
		re := regexp.MustCompile(`name="_csrf" value="([^"]+)"`)
		if mm := re.FindStringSubmatch(rec.Body.String()); len(mm) >= 2 {
			return mm[1]
		}
		t.Fatalf("csrf token tak ditemukan")
		return ""
	}

	// Login admin.
	tok := csrfOf(do(http.MethodGet, "/auth/login", nil))
	if lr := do(http.MethodPost, "/auth/login", url.Values{"email": {"admin@goadmin.test"}, "password": {"secret123"}, "_csrf": {tok}}); lr.Code != http.StatusFound {
		t.Fatalf("login = %d", lr.Code)
	}

	// Daftar Role → ada menu "Permission" di dropdown.
	rl := do(http.MethodGet, "/admin/v1/access/role", nil)
	if rl.Code != http.StatusOK {
		t.Fatalf("role list = %d", rl.Code)
	}
	if !strings.Contains(rl.Body.String(), "/role/"+role.ID+"/permission") {
		t.Fatalf("menu Permission tak ada di dropdown Role")
	}

	// Halaman kelola permission → 200 + permission tampil.
	pg := do(http.MethodGet, "/admin/v1/access/role/"+role.ID+"/permission", nil)
	if pg.Code != http.StatusOK {
		t.Fatalf("permission page = %d", pg.Code)
	}
	if !strings.Contains(pg.Body.String(), perm.Name) {
		t.Fatalf("permission %s tak tampil di halaman", perm.Name)
	}
	tok = csrfOf(pg)

	count := func() int64 {
		var n int64
		db.Table("roles_permissions").Where("role_id = ? AND permission_id = ?", role.ID, perm.ID).Count(&n)
		return n
	}

	// Assign single (GET link) → join row dibuat.
	if a := do(http.MethodGet, "/admin/v1/access/role/"+role.ID+"/permission/"+perm.ID+"/assign", nil); a.Code != http.StatusFound {
		t.Fatalf("assign = %d (harap 302)", a.Code)
	}
	if count() != 1 {
		t.Fatalf("assign gagal: join row = %d (harap 1)", count())
	}

	// Unassign bulk (POST + _csrf) → join row dihapus.
	if u := do(http.MethodPost, "/admin/v1/access/role/"+role.ID+"/permission/unassign_selected", url.Values{"selected[]": {perm.ID}, "_csrf": {tok}}); u.Code != http.StatusFound {
		t.Fatalf("unassign_selected = %d (harap 302)", u.Code)
	}
	if count() != 0 {
		t.Fatalf("unassign gagal: join row = %d (harap 0)", count())
	}
}

// TestApiRolePermission memverifikasi endpoint API kelola-permission per-role
// (kembar web, persis NodeAdmin): list + assign/unassign single via JWT.
func TestApiRolePermission(t *testing.T) {
	w, err := newWorld()
	if err != nil {
		t.Fatalf("world: %v", err)
	}
	if err := w.login(adminEmail, adminPass); err != nil {
		t.Fatalf("login: %v", err)
	}
	db := w.cont.DB
	role := model.Role{ID: helpers.NewID(), Name: "ApiEditor", GuardName: "api", Status: model.StatusActive}
	if err := db.Create(&role).Error; err != nil {
		t.Fatalf("role: %v", err)
	}
	perm := model.Permission{ID: helpers.NewID(), Name: "api.v1.access.user.index", Method: "GET", GuardName: "api", Status: model.StatusActive}
	if err := db.Create(&perm).Error; err != nil {
		t.Fatalf("perm: %v", err)
	}
	count := func() int64 {
		var n int64
		db.Table("roles_permissions").Where("role_id = ? AND permission_id = ?", role.ID, perm.ID).Count(&n)
		return n
	}

	w.request(http.MethodGet, "/api/v1/access/role/"+role.ID+"/permission", true)
	if w.status != http.StatusOK {
		t.Fatalf("API list permission role = %d", w.status)
	}
	w.request(http.MethodGet, "/api/v1/access/role/"+role.ID+"/permission/"+perm.ID+"/assign", true)
	if w.status != http.StatusOK || count() != 1 {
		t.Fatalf("API assign gagal: status=%d count=%d", w.status, count())
	}
	w.request(http.MethodGet, "/api/v1/access/role/"+role.ID+"/permission/"+perm.ID+"/unassign", true)
	if w.status != http.StatusOK || count() != 0 {
		t.Fatalf("API unassign gagal: status=%d count=%d", w.status, count())
	}
}

// TestFePreviewFoldedToSetting memverifikasi FE-template switcher FOLDED ke
// Setting (persis NodeAdmin): proxy preview ada di namespace setting
// (`/admin/v1/setting/fe-preview/:slug`), thumbnail halaman Setting menunjuk ke
// situ, dan halaman/route "appearance" terpisah SUDAH TIDAK ADA (404).
func TestFePreviewFoldedToSetting(t *testing.T) {
	gin.SetMode(gin.TestMode)
	wd, _ := os.Getwd()
	if err := os.Chdir("../.."); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(wd)

	cfg := &config.Config{
		Env: "test", IsTest: true,
		App:      config.AppConfig{Name: "GoAdmin Test", Mode: config.ModeFull},
		DB:       config.DBConfig{Type: "sqlite"},
		Session:  config.SessionConfig{Secret: "test-session-secret"},
		JWT:      config.JWTConfig{Secret: "test-jwt", Algorithm: "HS256"},
		Security: config.SecurityConfig{BcryptRounds: 4},
	}
	db, err := database.OpenSQLiteMemory(t.Name())
	if err != nil {
		t.Fatalf("db: %v", err)
	}
	if err := bootstrap.MigrateAndSeed(db, "admin@goadmin.test", "secret123", cfg.Security.BcryptRounds); err != nil {
		t.Fatalf("seed: %v", err)
	}
	handler := middleware.MethodOverride(app.Build(container.MustNew(cfg, db, nil)))

	jar := map[string]string{}
	do := func(method, path string, form url.Values) *httptest.ResponseRecorder {
		var body *strings.Reader
		if form != nil {
			body = strings.NewReader(form.Encode())
		} else {
			body = strings.NewReader("")
		}
		req := httptest.NewRequest(method, path, body)
		if form != nil {
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		}
		req.RemoteAddr = "10.77.0.2:1234" // IP unik → bucket rate-limit terisolasi antar-test
		var ck []string
		for k, v := range jar {
			ck = append(ck, k+"="+v)
		}
		if len(ck) > 0 {
			req.Header.Set("Cookie", strings.Join(ck, "; "))
		}
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		for _, c := range rec.Result().Cookies() {
			jar[c.Name] = c.Value
		}
		return rec
	}

	tok := csrfMetaRe.FindStringSubmatch(do(http.MethodGet, "/auth/login", nil).Body.String())
	if len(tok) < 2 {
		re := regexp.MustCompile(`name="_csrf" value="([^"]+)"`)
		tok = re.FindStringSubmatch(do(http.MethodGet, "/auth/login", nil).Body.String())
	}
	do(http.MethodPost, "/auth/login", url.Values{"email": {"admin@goadmin.test"}, "password": {"secret123"}, "_csrf": {tok[1]}})

	const slug = "agency-consulting-002-creative-agency" // builtin default

	// Preview di namespace SETTING → 200.
	if r := do(http.MethodGet, "/admin/v1/setting/fe-preview/"+slug, nil); r.Code != http.StatusOK {
		t.Fatalf("setting/fe-preview = %d (harap 200)", r.Code)
	}
	// Halaman Setting → thumbnail menunjuk ke /setting/fe-preview (folded).
	if s := do(http.MethodGet, "/admin/v1/setting", nil); s.Code != http.StatusOK || !strings.Contains(s.Body.String(), "/admin/v1/setting/fe-preview/") {
		t.Fatalf("setting page (%d) tak memuat URL preview folded", s.Code)
	}
	// Route appearance lama SUDAH TIDAK ADA → 404.
	if r := do(http.MethodGet, "/admin/v1/appearance", nil); r.Code != http.StatusNotFound {
		t.Fatalf("/admin/v1/appearance harus 404, dapat %d", r.Code)
	}
	if r := do(http.MethodGet, "/admin/v1/appearance/preview/"+slug, nil); r.Code != http.StatusNotFound {
		t.Fatalf("/admin/v1/appearance/preview lama harus 404, dapat %d", r.Code)
	}
}

// TestApiAuthRegisterReset memverifikasi PARITAS endpoint auth API dgn web:
// register (bikin user → bisa login) + reset password OTP (request kirim OTP,
// process verifikasi). Endpoint ini sebelumnya HILANG di GoAdmin API.
func TestApiAuthRegisterReset(t *testing.T) {
	w, err := newWorld()
	if err != nil {
		t.Fatalf("world: %v", err)
	}
	post := func(path string, body any) *httptest.ResponseRecorder {
		b, _ := json.Marshal(body)
		req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(b))
		req.Header.Set("Content-Type", "application/json")
		req.RemoteAddr = "10.77.0.1:1234" // IP unik → bucket rate-limit terisolasi antar-test
		rec := httptest.NewRecorder()
		w.engine.ServeHTTP(rec, req)
		return rec
	}

	// register → 201, lalu login dgn akun baru → bukti user benar dibuat.
	if r := post("/api/v1/auth/register", map[string]any{"name": "New", "email": "new@goadmin.test", "password": "password123"}); r.Code != http.StatusCreated {
		t.Fatalf("register = %d: %s", r.Code, r.Body.String())
	}
	if err := w.login("new@goadmin.test", "password123"); err != nil {
		t.Fatalf("login akun hasil register gagal: %v", err)
	}

	// reset/request → 200 (selalu sukses; anti user-enumeration).
	if r := post("/api/v1/auth/reset/request", map[string]any{"email": "new@goadmin.test"}); r.Code != http.StatusOK {
		t.Fatalf("reset/request = %d: %s", r.Code, r.Body.String())
	}
	// reset/process dgn OTP salah → 4xx (endpoint ada + verifikasi jalan).
	if r := post("/api/v1/auth/reset/process", map[string]any{"email": "new@goadmin.test", "otp": "000000", "password": "newpassword123"}); r.Code < 400 {
		t.Fatalf("reset/process OTP salah seharusnya >=400, dapat %d: %s", r.Code, r.Body.String())
	}
}

// TestAuthResetNamingAndLogout memverifikasi paritas NodeAdmin: flow reset
// password pakai nama/path `/admin/v1/auth/reset/*` (req/proc/request/process),
// path lama `/auth/forgot`/`/auth/reset` HILANG, dan logout web = POST (CSRF).
func TestAuthResetNamingAndLogout(t *testing.T) {
	gin.SetMode(gin.TestMode)
	wd, _ := os.Getwd()
	if err := os.Chdir("../.."); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(wd)

	cfg := &config.Config{
		Env: "test", IsTest: true,
		App:      config.AppConfig{Name: "GoAdmin Test", Mode: config.ModeFull},
		DB:       config.DBConfig{Type: "sqlite"},
		Session:  config.SessionConfig{Secret: "test-session-secret"},
		JWT:      config.JWTConfig{Secret: "test-jwt", Algorithm: "HS256"},
		Security: config.SecurityConfig{BcryptRounds: 4},
	}
	db, err := database.OpenSQLiteMemory(t.Name())
	if err != nil {
		t.Fatalf("db: %v", err)
	}
	if err := bootstrap.MigrateAndSeed(db, "admin@goadmin.test", "secret123", cfg.Security.BcryptRounds); err != nil {
		t.Fatalf("seed: %v", err)
	}
	handler := middleware.MethodOverride(app.Build(container.MustNew(cfg, db, nil)))

	jar := map[string]string{}
	do := func(method, path string, form url.Values) *httptest.ResponseRecorder {
		var body *strings.Reader
		if form != nil {
			body = strings.NewReader(form.Encode())
		} else {
			body = strings.NewReader("")
		}
		req := httptest.NewRequest(method, path, body)
		if form != nil {
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		}
		req.RemoteAddr = "10.77.0.2:1234" // IP unik → bucket rate-limit terisolasi antar-test
		var ck []string
		for k, v := range jar {
			ck = append(ck, k+"="+v)
		}
		if len(ck) > 0 {
			req.Header.Set("Cookie", strings.Join(ck, "; "))
		}
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		for _, c := range rec.Result().Cookies() {
			jar[c.Name] = c.Value
		}
		return rec
	}
	csrfHidden := regexp.MustCompile(`name="_csrf" value="([^"]+)"`)

	// 1) Halaman reset (publik) di path baru → 200.
	reqPage := do(http.MethodGet, "/admin/v1/auth/reset/req", nil)
	if reqPage.Code != http.StatusOK {
		t.Fatalf("GET reset/req = %d (harap 200)", reqPage.Code)
	}
	if do(http.MethodGet, "/admin/v1/auth/reset/proc", nil).Code != http.StatusOK {
		t.Fatalf("GET reset/proc bukan 200")
	}
	// 2) Login page memuat link Forgot ke path baru.
	if lp := do(http.MethodGet, "/auth/login", nil); !strings.Contains(lp.Body.String(), "/admin/v1/auth/reset/req") {
		t.Fatalf("link Forgot di login tak menunjuk reset/req")
	}
	// 3) POST reset/request (publik, butuh CSRF dari halaman) → 302.
	m := csrfHidden.FindStringSubmatch(reqPage.Body.String())
	if len(m) < 2 {
		t.Fatalf("csrf di reset/req tak ada")
	}
	if rr := do(http.MethodPost, "/admin/v1/auth/reset/request", url.Values{"email": {"admin@goadmin.test"}, "_csrf": {m[1]}}); rr.Code != http.StatusFound {
		t.Fatalf("POST reset/request = %d (harap 302)", rr.Code)
	}
	// 4) Path lama HILANG → 404.
	if do(http.MethodGet, "/auth/forgot", nil).Code != http.StatusNotFound {
		t.Fatalf("/auth/forgot lama harus 404")
	}
	if do(http.MethodGet, "/auth/reset", nil).Code != http.StatusNotFound {
		t.Fatalf("/auth/reset lama harus 404")
	}

	// 5) Logout web = POST. Login dulu → topbar punya form logout POST → POST 302.
	tok := csrfHidden.FindStringSubmatch(do(http.MethodGet, "/auth/login", nil).Body.String())
	do(http.MethodPost, "/auth/login", url.Values{"email": {"admin@goadmin.test"}, "password": {"secret123"}, "_csrf": {tok[1]}})
	home := do(http.MethodGet, "/admin/v1/access/user", nil)
	if !strings.Contains(home.Body.String(), `action="/auth/logout" method="post"`) {
		t.Fatalf("topbar tak punya form logout POST")
	}
	ltok := csrfHidden.FindStringSubmatch(home.Body.String())
	if lo := do(http.MethodPost, "/auth/logout", url.Values{"_csrf": {ltok[1]}}); lo.Code != http.StatusFound {
		t.Fatalf("POST logout = %d (harap 302)", lo.Code)
	}
	// GET logout (method lama) tak boleh sukses-logout.
	if do(http.MethodGet, "/auth/logout", nil).Code == http.StatusFound {
		t.Fatalf("GET /auth/logout seharusnya bukan 302 (method kini POST)")
	}
}

// TestSettingInlineValidation memverifikasi error validasi form Setting tampil
// INLINE per-field (padanan getError NodeAdmin), bukan hanya flash generik:
// upload gambar tak valid → field icon `is-invalid` + pesan + form diisi ulang
// (old input), bukan kehilangan isian.
func TestSettingInlineValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	wd, _ := os.Getwd()
	if err := os.Chdir("../.."); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(wd)

	cfg := &config.Config{
		Env: "test", IsTest: true,
		App:      config.AppConfig{Name: "GoAdmin Test", Mode: config.ModeFull},
		DB:       config.DBConfig{Type: "sqlite"},
		Session:  config.SessionConfig{Secret: "test-session-secret"},
		JWT:      config.JWTConfig{Secret: "test-jwt", Algorithm: "HS256"},
		Security: config.SecurityConfig{BcryptRounds: 4},
	}
	db, err := database.OpenSQLiteMemory(t.Name())
	if err != nil {
		t.Fatalf("db: %v", err)
	}
	if err := bootstrap.MigrateAndSeed(db, "admin@goadmin.test", "secret123", cfg.Security.BcryptRounds); err != nil {
		t.Fatalf("seed: %v", err)
	}
	handler := middleware.MethodOverride(app.Build(container.MustNew(cfg, db, nil)))

	jar := map[string]string{}
	send := func(req *http.Request) *httptest.ResponseRecorder {
		req.RemoteAddr = "10.77.0.3:1234"
		var ck []string
		for k, v := range jar {
			ck = append(ck, k+"="+v)
		}
		if len(ck) > 0 {
			req.Header.Set("Cookie", strings.Join(ck, "; "))
		}
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		for _, c := range rec.Result().Cookies() {
			jar[c.Name] = c.Value
		}
		return rec
	}
	form := func(method, path string, vals url.Values) *httptest.ResponseRecorder {
		var body *strings.Reader
		if vals != nil {
			body = strings.NewReader(vals.Encode())
		} else {
			body = strings.NewReader("")
		}
		req := httptest.NewRequest(method, path, body)
		if vals != nil {
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		}
		return send(req)
	}
	csrfRe := regexp.MustCompile(`name="_csrf" value="([^"]+)"`)

	// Login admin.
	tok := csrfRe.FindStringSubmatch(form(http.MethodGet, "/auth/login", nil).Body.String())
	form(http.MethodPost, "/auth/login", url.Values{"email": {"admin@goadmin.test"}, "password": {"secret123"}, "_csrf": {tok[1]}})

	// Token dari halaman Setting.
	stok := csrfRe.FindStringSubmatch(form(http.MethodGet, "/admin/v1/setting", nil).Body.String())[1]

	// Submit multipart: name diubah + icon = file BUKAN gambar (magic-byte gagal).
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	_ = mw.WriteField("_csrf", stok)
	_ = mw.WriteField("name", "EditedName")
	fw, _ := mw.CreateFormFile("icon", "bad.png")
	_, _ = fw.Write([]byte("ini jelas bukan gambar valid"))
	_ = mw.Close()
	req := httptest.NewRequest(http.MethodPost, "/admin/v1/setting/update?_method=PUT", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	if up := send(req); up.Code != http.StatusFound {
		t.Fatalf("update setting = %d (harap 302): %s", up.Code, up.Body.String())
	}

	// GET Setting → error inline icon + form diisi ulang (name=EditedName).
	page := form(http.MethodGet, "/admin/v1/setting", nil).Body.String()
	if !strings.Contains(page, "is-invalid") {
		t.Fatalf("tak ada is-invalid (error inline) di halaman Setting")
	}
	if !strings.Contains(page, "invalid-feedback") {
		t.Fatalf("tak ada invalid-feedback (pesan error)")
	}
	if !strings.Contains(page, `value="EditedName"`) {
		t.Fatalf("input name tak diisi ulang dari old input (EditedName)")
	}
}

// TestProfileStructureAndValidation memverifikasi form Profile GoAdmin BERSTRUKTUR
// persis NodeAdmin (Code, Timezone=select, Status=select, Picture) + validasi
// inline (password confirm tak cocok → is-invalid + old input terisi ulang).
func TestProfileStructureAndValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	wd, _ := os.Getwd()
	if err := os.Chdir("../.."); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(wd)

	cfg := &config.Config{
		Env: "test", IsTest: true,
		App:      config.AppConfig{Name: "GoAdmin Test", Mode: config.ModeFull},
		DB:       config.DBConfig{Type: "sqlite"},
		Session:  config.SessionConfig{Secret: "test-session-secret"},
		JWT:      config.JWTConfig{Secret: "test-jwt", Algorithm: "HS256"},
		Security: config.SecurityConfig{BcryptRounds: 4},
	}
	db, err := database.OpenSQLiteMemory(t.Name())
	if err != nil {
		t.Fatalf("db: %v", err)
	}
	if err := bootstrap.MigrateAndSeed(db, "admin@goadmin.test", "secret123", cfg.Security.BcryptRounds); err != nil {
		t.Fatalf("seed: %v", err)
	}
	handler := middleware.MethodOverride(app.Build(container.MustNew(cfg, db, nil)))

	jar := map[string]string{}
	form := func(method, path string, vals url.Values) *httptest.ResponseRecorder {
		var body *strings.Reader
		if vals != nil {
			body = strings.NewReader(vals.Encode())
		} else {
			body = strings.NewReader("")
		}
		req := httptest.NewRequest(method, path, body)
		if vals != nil {
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		}
		req.RemoteAddr = "10.77.0.4:1234"
		var ck []string
		for k, v := range jar {
			ck = append(ck, k+"="+v)
		}
		if len(ck) > 0 {
			req.Header.Set("Cookie", strings.Join(ck, "; "))
		}
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		for _, c := range rec.Result().Cookies() {
			jar[c.Name] = c.Value
		}
		return rec
	}
	csrfRe := regexp.MustCompile(`name="_csrf" value="([^"]+)"`)

	tok := csrfRe.FindStringSubmatch(form(http.MethodGet, "/auth/login", nil).Body.String())
	form(http.MethodPost, "/auth/login", url.Values{"email": {"admin@goadmin.test"}, "password": {"secret123"}, "_csrf": {tok[1]}})

	// Struktur form Profile = persis NodeAdmin.
	pg := form(http.MethodGet, "/admin/v1/profile", nil)
	if pg.Code != http.StatusOK {
		t.Fatalf("GET profile = %d", pg.Code)
	}
	body := pg.Body.String()
	for _, must := range []string{
		`name="code"`, `name="name"`, `name="phone"`, `name="email"`,
		`<select id="timezone"`, `name="password"`, `name="password_confirmation"`,
		`<select name="status"`, `id="picture"`, `>UTC<`, // option timezone terisi
	} {
		if !strings.Contains(body, must) {
			t.Fatalf("struktur form Profile tak memuat: %s", must)
		}
	}
	stok := csrfRe.FindStringSubmatch(body)[1]

	// Validasi inline: password confirm tak cocok → is-invalid + old input.
	form(http.MethodPost, "/admin/v1/profile/update?_method=PUT", url.Values{
		"name": {"Admin Edited"}, "email": {"admin@goadmin.test"},
		"password": {"password123"}, "password_confirmation": {"beda"}, "_csrf": {stok},
	})
	after := form(http.MethodGet, "/admin/v1/profile", nil).Body.String()
	if !strings.Contains(after, "is-invalid") || !strings.Contains(after, "not match") {
		t.Fatalf("error inline password confirm tak tampil")
	}
	if !strings.Contains(after, `value="Admin Edited"`) {
		t.Fatalf("name tak diisi ulang dari old input")
	}
}

// TestUserBlockedFieldPersists memverifikasi field Blocked + BlockedReason
// (baru ditambahkan ke GoAdmin agar setara NodeAdmin/RustAdmin) benar-benar
// di-bind dari payload + tersimpan.
func TestUserBlockedFieldPersists(t *testing.T) {
	w, err := newWorld()
	if err != nil {
		t.Fatalf("world: %v", err)
	}
	if err := w.login(adminEmail, adminPass); err != nil {
		t.Fatalf("login: %v", err)
	}
	b, _ := json.Marshal(map[string]any{
		"name": "Blk", "email": "blk@goadmin.test", "password": "password123",
		"blocked": true, "blocked_reason": "spam",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/access/user/store", bytes.NewReader(b))
	req.Header.Set("Authorization", "Bearer "+w.token)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	w.engine.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create = %d: %s", rec.Code, rec.Body.String())
	}
	var u model.User
	if err := w.cont.DB.First(&u, "email = ?", "blk@goadmin.test").Error; err != nil {
		t.Fatalf("user tak ditemukan: %v", err)
	}
	if !u.Blocked || u.BlockedReason != "spam" {
		t.Fatalf("Blocked tak tersimpan: Blocked=%v BlockedReason=%q", u.Blocked, u.BlockedReason)
	}
}
