// Package web berisi controller HTML modul access (jalur sesi). Render lewat
// view.RenderView (bukan c.HTML path mentah) — di-enforce checker.
package web

import (
	"net/http"
	"net/mail"
	"strings"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"

	"goadmin/internal/helpers"
	"goadmin/internal/middleware"
	"goadmin/internal/modules/access/dto"
	accessmw "goadmin/internal/modules/access/middleware"
	"goadmin/internal/modules/access/model"
	"goadmin/internal/modules/access/service"
	"goadmin/internal/storage"
	"goadmin/internal/view"
)

// UserController menyajikan CRUD user (web). Memakai RoleService untuk pilihan
// role pada form (checkbox) + storage untuk upload foto.
type UserController struct {
	users   service.IUserService
	roles   service.IRoleService
	storage storage.Storage
}

// NewUserController merakit controller (service + storage di-inject).
func NewUserController(users service.IUserService, roles service.IRoleService, store storage.Storage) *UserController {
	return &UserController{users: users, roles: roles, storage: store}
}

// pictureURL memproses file "picture" (opsional): validasi + simpan → URL.
func (ctl *UserController) pictureURL(c *gin.Context) (string, error) {
	fh, err := c.FormFile("picture")
	if err != nil || fh == nil {
		return "", nil // tak ada file
	}
	f, oerr := fh.Open()
	if oerr != nil {
		return "", oerr
	}
	defer f.Close()
	return storage.ValidateAndSave(c.Request.Context(), ctl.storage, f)
}

// Index → GET /admin/v1/access/user (daftar user + filter per-kolom + paginasi).
func (ctl *UserController) Index(c *gin.Context) {
	q, qbase := bindListQuery(c)
	res, err := ctl.users.Index(c.Request.Context(), q)
	if err != nil {
		c.Error(err)
		return
	}
	roles, err := ctl.allRoles(c) // untuk dropdown filter q_role
	if err != nil {
		c.Error(err)
		return
	}
	view.RenderView(c, "access/users/index", gin.H{
		"title": "Manajemen User", "active": "user",
		"users": res.Data, "meta": res.Meta,
		"filter": filterMap(q), "qbase": qbase, "roles": roles,
	})
}

// DeleteSelected → POST /admin/v1/access/user/delete_selected (bulk delete tabel).
func (ctl *UserController) DeleteSelected(c *gin.Context) {
	if err := ctl.users.DestroyMany(c.Request.Context(), selectedIDs(c)); err != nil {
		setFlashError(sessions.Default(c), errMessage(err))
	} else {
		setFlashSuccess(sessions.Default(c), "Delete User Success.")
	}
	c.Redirect(http.StatusFound, "/admin/v1/access/user")
}

// Create → GET /admin/v1/access/user/create.
func (ctl *UserController) Create(c *gin.Context) {
	roles, err := ctl.allRoles(c)
	if err != nil {
		c.Error(err)
		return
	}
	view.RenderView(c, "users/form", gin.H{
		"title": "Tambah Pengguna", "active": "user",
		"action": "/admin/v1/access/user/store", "user": nil,
		"roles": roles, "selected": map[string]bool{},
		"timezones": helpers.Timezones(),
	})
}

// Store → POST /admin/v1/access/user.
func (ctl *UserController) Store(c *gin.Context) {
	var in dto.CreateUserInput
	_ = c.ShouldBind(&in)
	sess := sessions.Default(c)

	errs := userFormErrors(in.Name, in.Email, in.Password, in.PasswordConfirmation, true)
	if url, perr := ctl.pictureURL(c); perr != nil {
		errs["picture"] = errMessage(perr)
	} else if url != "" {
		in.Picture = url
	}
	if len(errs) > 0 {
		middleware.SetFieldErrors(sess, errs, userOld(c))
		setFlashError(sess, "Please check the marked fields.")
		c.Redirect(http.StatusFound, "/admin/v1/access/user/create")
		return
	}
	if _, err := ctl.users.Store(c.Request.Context(), in, actorID(c)); err != nil {
		setFlashError(sess, errMessage(err))
		c.Redirect(http.StatusFound, "/admin/v1/access/user/create")
		return
	}
	setFlashSuccess(sess, "Create User Success.")
	c.Redirect(http.StatusFound, "/admin/v1/access/user")
}

// Edit → GET /admin/v1/access/user/:id/edit.
func (ctl *UserController) Edit(c *gin.Context) {
	user, err := ctl.users.Show(c.Request.Context(), c.Param("id"))
	if err != nil {
		c.Error(err)
		return
	}
	roles, err := ctl.allRoles(c)
	if err != nil {
		c.Error(err)
		return
	}
	selected := make(map[string]bool, len(user.Roles))
	for _, r := range user.Roles {
		selected[r.ID] = true
	}
	view.RenderView(c, "users/form", gin.H{
		"title": "Ubah Pengguna", "active": "user",
		"action": "/admin/v1/access/user/" + user.ID + "/update?_method=PUT", "user": user,
		"roles": roles, "selected": selected,
		"timezones": helpers.Timezones(),
	})
}

// Update → POST /admin/v1/access/user/:id.
func (ctl *UserController) Update(c *gin.Context) {
	id := c.Param("id")
	var in dto.UpdateUserInput
	_ = c.ShouldBind(&in)
	sess := sessions.Default(c)

	// Password opsional pada update (kosong = tetap) → passwordRequired=false.
	errs := userFormErrors(in.Name, in.Email, in.Password, in.PasswordConfirmation, false)
	if url, perr := ctl.pictureURL(c); perr != nil {
		errs["picture"] = errMessage(perr)
	} else if url != "" {
		in.Picture = url
	}
	if len(errs) > 0 {
		middleware.SetFieldErrors(sess, errs, userOld(c))
		setFlashError(sess, "Please check the marked fields.")
		c.Redirect(http.StatusFound, "/admin/v1/access/user/"+id+"/edit")
		return
	}
	if _, err := ctl.users.Update(c.Request.Context(), id, in, actorID(c)); err != nil {
		setFlashError(sess, errMessage(err))
		c.Redirect(http.StatusFound, "/admin/v1/access/user/"+id+"/edit")
		return
	}
	setFlashSuccess(sess, "Update User Success.")
	c.Redirect(http.StatusFound, "/admin/v1/access/user")
}

// userFormErrors memvalidasi field form user (web) untuk inline error — padanan
// validasi NodeAdmin: name wajib, email format, password min8 + cocok konfirmasi.
// passwordRequired=true pada create; pada update password boleh kosong (tetap).
func userFormErrors(name, email, password, passwordConfirmation string, passwordRequired bool) map[string]string {
	errs := map[string]string{}
	if strings.TrimSpace(name) == "" {
		errs["name"] = "Name wajib diisi."
	}
	if strings.TrimSpace(email) == "" {
		errs["email"] = "Email wajib diisi."
	} else if _, e := mail.ParseAddress(email); e != nil {
		errs["email"] = "Format email tidak valid."
	}
	if passwordRequired && password == "" {
		errs["password"] = "Password wajib diisi."
	}
	if password != "" {
		if len(password) < 8 {
			errs["password"] = "Password minimal 8 karakter."
		}
		if password != passwordConfirmation {
			errs["password_confirmation"] = "Password & confirm password not match."
		}
	}
	return errs
}

// userOld menyalin nilai field teks dari form (untuk repopulate saat validasi gagal).
func userOld(c *gin.Context) map[string]string {
	old := map[string]string{}
	for _, k := range []string{"code", "name", "phone", "email", "timezone", "status", "blocked_reason"} {
		if v := c.PostForm(k); v != "" {
			old[k] = v
		}
	}
	return old
}

// Destroy → DELETE /admin/v1/access/user/:id/delete (form POST + ?_method=DELETE).
func (ctl *UserController) Destroy(c *gin.Context) {
	if err := ctl.users.Destroy(c.Request.Context(), c.Param("id")); err != nil {
		setFlashError(sessions.Default(c), errMessage(err))
	} else {
		setFlashSuccess(sessions.Default(c), "Delete User Success.")
	}
	c.Redirect(http.StatusFound, "/admin/v1/access/user")
}

// allRoles mengambil seluruh role (untuk pilihan di form).
func (ctl *UserController) allRoles(c *gin.Context) ([]model.Role, error) {
	res, err := ctl.roles.Index(c.Request.Context(), dto.ListQuery{Page: 1, PerPage: 100})
	if err != nil {
		return nil, err
	}
	return res.Data, nil
}

// actorID mengambil id user terautentikasi dari context (untuk audit created_by).
func actorID(c *gin.Context) string {
	if u := accessmw.UserFrom(c); u != nil {
		return u.ID
	}
	return ""
}
