// Package web berisi controller HTML modul profile. Render lewat
// view.RenderView (bukan c.HTML path mentah) — di-enforce checker.
package web

import (
	"net/http"
	"net/mail"
	"strings"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"

	apperr "goadmin/internal/errors"
	"goadmin/internal/helpers"
	"goadmin/internal/middleware"
	accessmw "goadmin/internal/modules/access/middleware"
	"goadmin/internal/modules/profile/dto"
	"goadmin/internal/modules/profile/service"
	"goadmin/internal/storage"
	"goadmin/internal/view"
)

// ProfileController menyajikan halaman profil milik-sendiri.
type ProfileController struct {
	profiles service.IProfileService
	storage  storage.Storage
}

// NewProfileController merakit controller (service + storage di-inject).
func NewProfileController(profiles service.IProfileService, store storage.Storage) *ProfileController {
	return &ProfileController{profiles: profiles, storage: store}
}

// Index → GET /admin/v1/profile (form profil; struktur field PERSIS NodeAdmin:
// Code, Name, Phone, Email, Timezone(select), Password, Password Confirm,
// Status(select), Picture).
func (ctl *ProfileController) Index(c *gin.Context) {
	user := accessmw.UserFrom(c)
	if user == nil {
		c.Redirect(http.StatusFound, "/auth/login")
		return
	}
	profile, err := ctl.profiles.Get(c.Request.Context(), user.ID)
	if err != nil {
		c.Error(err)
		return
	}
	view.RenderView(c, "profile/index", gin.H{
		"title":     "Profil Saya",
		"active":    "profile",
		"profile":   profile,
		"timezones": helpers.Timezones(), // select Timezone (padanan getTimezones NodeAdmin)
	})
}

// Update → PUT /admin/v1/profile/update (PRG). Validasi inline per-field (sejajar
// ProfileUpdateValidator NodeAdmin): name wajib, email format, password min 8 +
// konfirmasi cocok, avatar (magic-byte). Error → ditampilkan inline + old input.
func (ctl *ProfileController) Update(c *gin.Context) {
	user := accessmw.UserFrom(c)
	if user == nil {
		c.Redirect(http.StatusFound, "/auth/login")
		return
	}
	var in dto.UpdateProfileInput
	_ = c.ShouldBind(&in)
	sess := sessions.Default(c)

	errs := map[string]string{}
	if strings.TrimSpace(in.Name) == "" {
		errs["name"] = "Name wajib diisi."
	}
	if strings.TrimSpace(in.Email) == "" {
		errs["email"] = "Email wajib diisi."
	} else if _, e := mail.ParseAddress(in.Email); e != nil {
		errs["email"] = "Format email tidak valid."
	}
	if in.Password != "" {
		if len(in.Password) < 8 {
			errs["password"] = "Password minimal 8 karakter."
		}
		if in.Password != in.PasswordConfirmation {
			errs["password_confirmation"] = "Password & confirm password not match."
		}
	}

	// Avatar opsional: validasi (magic-byte) + simpan → URL.
	if fh, ferr := c.FormFile("picture"); ferr == nil && fh != nil {
		f, oerr := fh.Open()
		if oerr != nil {
			errs["picture"] = "Gagal membaca file."
		} else {
			defer f.Close()
			url, uerr := storage.ValidateAndSave(c.Request.Context(), ctl.storage, f)
			if uerr != nil {
				errs["picture"] = publicErr(uerr)
			} else {
				in.Picture = url
			}
		}
	}

	if len(errs) > 0 {
		middleware.SetFieldErrors(sess, errs, profileOld(c))
		middleware.SetFlashError(sess, "Please check the marked fields.")
		c.Redirect(http.StatusFound, "/admin/v1/profile")
		return
	}

	if _, err := ctl.profiles.Update(c.Request.Context(), user.ID, in); err != nil {
		middleware.SetFlashError(sess, publicErr(err))
		c.Redirect(http.StatusFound, "/admin/v1/profile")
		return
	}
	middleware.SetFlashSuccess(sess, "Update Profile Success.")
	c.Redirect(http.StatusFound, "/admin/v1/profile")
}

// profileOld menangkap nilai form teks (repopulasi inline saat validasi gagal).
func profileOld(c *gin.Context) map[string]string {
	keys := []string{"code", "name", "phone", "email", "timezone", "status"}
	old := make(map[string]string, len(keys))
	for _, k := range keys {
		if v := c.PostForm(k); v != "" {
			old[k] = v
		}
	}
	return old
}

// publicErr mengambil pesan publik dari *AppError (fallback generik).
func publicErr(err error) string {
	if ae, ok := apperr.As(err); ok {
		return ae.Message
	}
	return "Terjadi kesalahan."
}
