package integration

import (
	"bytes"
	"html/template"
	"strings"
	"testing"

	"goadmin/internal/view"
)

// Memuat partials chrome + semua view modul dalam SATU set lalu merender tiap
// view — membuktikan cross-reference {{template "admin/head"}} resolve & semua
// view valid + render tanpa error (deteksi dini sebelum runtime).
func TestViews_AllRenderWithChrome(t *testing.T) {
	files := []string{
		"../../web/templates/partials/admin_chrome.html",
		"../../web/templates/partials/auth_chrome.html",
		"../../internal/modules/access/view/users.html",
		"../../internal/modules/access/view/auth_login.html",
		"../../internal/modules/access/view/auth_register.html",
		"../../internal/modules/access/view/auth_forgot.html",
		"../../internal/modules/access/view/auth_reset.html",
		"../../internal/modules/access/view/users_form.html",
		"../../internal/modules/access/view/roles_index.html",
		"../../internal/modules/access/view/roles_form.html",
		"../../internal/modules/access/view/permissions_index.html",
		"../../internal/modules/access/view/permissions_form.html",
		"../../internal/modules/dashboard/view/index.html",
		"../../internal/modules/setting/view/index.html",
		"../../internal/modules/profile/view/index.html",
		"../../internal/modules/home/view/default.html",
		"../../internal/modules/components/view/index.html",
	}
	set, err := template.New("").Funcs(view.FuncMap()).ParseFiles(files...)
	if err != nil {
		t.Fatalf("parse views: %v", err)
	}

	cases := []struct {
		name string
		data map[string]interface{}
		want string
	}{
		{"auth/login", map[string]interface{}{"title": "Masuk", "flash_error": "Salah", "_csrf": "tok"}, "Hello, Welcome Back!"},
		{"auth/register", map[string]interface{}{"title": "Daftar", "_csrf": "tok"}, "Create Account"},
		{"dashboard/index", map[string]interface{}{
			"title": "Dashboard", "active": "dashboard", "currentUser": nil,
			"stats": map[string]interface{}{"Users": 1, "Roles": 1, "Permissions": 12},
		}, "Dashboard Overview"},
		{"components/index", map[string]interface{}{"title": "UI Components", "active": "components", "currentUser": nil}, "Stat Card + Counter"},
		{"auth/forgot", map[string]interface{}{"title": "Lupa Password", "_csrf": "tok"}, "Kirim OTP"},
		{"auth/reset", map[string]interface{}{"title": "Reset Password", "_csrf": "tok", "email": "a@b.com"}, "Kode OTP"},
		{"access/users/index", map[string]interface{}{
			"title": "User", "active": "users", "currentUser": nil,
			"filter": map[string]interface{}{"q_page_size": 0, "q_code": "", "q_name": "", "q_phone": "", "q_email": "", "q_status": "", "q_role": ""},
			"qbase":  "", "roles": []map[string]interface{}{{"ID": "r1", "Name": "Administrator"}},
			"users": []map[string]interface{}{{"ID": "u1", "Code": "U-1", "Name": "Budi", "Phone": "08", "Email": "b@x.com", "Status": "Active"}},
			"meta":  map[string]interface{}{"From": 1, "To": 1, "Total": 1, "CurrentPage": 1, "LastPage": 1},
		}, "Budi"},
		{"setting/index", map[string]interface{}{
			"title": "Pengaturan", "active": "setting", "currentUser": nil,
			"errors": map[string]string{}, "old": map[string]string{}, // diset RenderView; di sini eksplisit
			"setting": map[string]interface{}{"Name": "Toko", "Initial": "T", "Email": "", "Phone": "", "Address": "", "Description": "", "Copyright": "", "Theme": "Blue"},
			"themes":  []map[string]interface{}{{"Name": "Blue"}, {"Name": "Green"}},
			// FE template switcher (folded) — branch aktif.
			"feEnabled": true, "feActiveSlug": "agency-consulting-002-creative-agency", "feCategories": []string{"Agency", "Travel"},
			"feSearch": "", "feCategory": "", "fePage": 1, "feLastPage": 1, "feTotal": 2,
			"feTemplates": []map[string]interface{}{{"Slug": "agency-consulting-002-creative-agency", "Name": "Creative Agency", "Category": "Agency", "Builtin": true}},
		}, "Frontend Template"},
		{"profile/index", map[string]interface{}{
			"title": "Profil", "active": "profile", "currentUser": nil,
			"errors": map[string]string{}, "old": map[string]string{}, // diset RenderView
			"timezones": []string{"UTC", "Asia/Jakarta"},
			"profile":   map[string]interface{}{"Code": "U-1", "Name": "Budi", "Email": "b@x.com", "Phone": "", "Timezone": "UTC", "Status": "Active"},
		}, "Password Confirm"},
		{"home/default", map[string]interface{}{
			"landing": map[string]interface{}{"AppName": "Toko Saya", "Description": "Halo", "Logo": "", "Email": "", "Phone": "", "Address": "", "Copyright": "© X", "ThemeName": "Green", "Primary": "#16a34a", "Accent": "#15803d", "Template": "default"},
		}, "Toko Saya"},
		{"users/form", map[string]interface{}{
			"title": "Tambah Pengguna", "active": "users", "currentUser": nil, "_csrf": "tok",
			"user": nil, "action": "/admin/v1/access/user", "selected": map[string]bool{},
			"roles":     []map[string]interface{}{{"ID": "r1", "Name": "Administrator"}},
			"timezones": []string{"UTC", "Asia/Jakarta"},
			"errors":    map[string]string{}, "old": map[string]string{},
		}, "Administrator"},
		{"roles/index", map[string]interface{}{
			"title": "Role", "active": "roles", "currentUser": nil, "_csrf": "tok",
			"filter": map[string]interface{}{"q_page_size": 0, "q_name": "", "q_guard": "", "q_status": "", "q_desc": ""}, "qbase": "",
			"roles": []map[string]interface{}{{"ID": "r1", "Name": "Editor", "GuardName": "web", "Status": "Active", "Description": "Editor role", "Permissions": []interface{}{}}},
			"meta":  map[string]interface{}{"From": 1, "To": 1, "Total": 1, "CurrentPage": 1, "LastPage": 1},
		}, "Editor"},
		{"roles/form", map[string]interface{}{
			"title": "Tambah Role", "active": "roles", "currentUser": nil, "_csrf": "tok",
			"role": nil, "action": "/admin/v1/access/role", "selected": map[string]bool{},
			"permissions": []map[string]interface{}{{"ID": "p1", "Name": "user.view"}},
		}, "user.view"},
		{"permissions/index", map[string]interface{}{
			"title": "Permission", "active": "permissions", "currentUser": nil, "_csrf": "tok",
			"filter": map[string]interface{}{"q_page_size": 0, "q_name": "", "q_guard": "", "q_method": "", "q_status": "", "q_desc": ""}, "qbase": "",
			"permissions": []map[string]interface{}{{"ID": "p1", "Name": "user.view", "GuardName": "web", "Method": "GET", "Status": "Active", "Description": "View user"}},
			"meta":        map[string]interface{}{"From": 1, "To": 1, "Total": 1, "CurrentPage": 1, "LastPage": 1},
		}, "user.view"},
		{"permissions/form", map[string]interface{}{
			"title": "Tambah Permission", "active": "permissions", "currentUser": nil, "_csrf": "tok",
			"permission": nil, "action": "/admin/v1/access/permission",
		}, "Permission Form"},
	}

	// Chrome admin themeable → tiap view butuh palet tema + setting di locals.
	theme := map[string]interface{}{"Primary": "#3B82F6", "Secondary": "#60A5FA", "Light": "#DBEAFE", "Dark": "#1E40AF"}
	for _, tc := range cases {
		if tc.data["theme"] == nil {
			tc.data["theme"] = theme
		}
		if tc.data["setting"] == nil {
			tc.data["setting"] = map[string]interface{}{"Name": "GoAdmin", "Logo": "", "Copyright": "© GoAdmin"}
		}
		var buf bytes.Buffer
		if err := set.ExecuteTemplate(&buf, tc.name, tc.data); err != nil {
			t.Fatalf("render %s: %v", tc.name, err)
		}
		if !strings.Contains(buf.String(), tc.want) {
			t.Fatalf("view %s tak memuat %q", tc.name, tc.want)
		}
	}
}
