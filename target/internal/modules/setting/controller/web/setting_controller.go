// Package web berisi controller HTML modul setting. Render lewat
// view.RenderView (bukan c.HTML path mentah) — di-enforce checker.
package web

import (
	"net/http"
	"strconv"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"

	apperr "goadmin/internal/errors"
	"goadmin/internal/middleware"
	accessmw "goadmin/internal/modules/access/middleware"
	"goadmin/internal/modules/home/fetemplate"
	"goadmin/internal/modules/setting/dto"
	"goadmin/internal/modules/setting/service"
	"goadmin/internal/modules/setting/theme"
	"goadmin/internal/storage"
	"goadmin/internal/view"
)

// feSwitcherPerPage = jumlah kartu template per halaman switcher (12, sama NodeAdmin).
const feSwitcherPerPage = 12

// SettingController menyajikan halaman pengaturan global + theme switcher +
// upload logo + switcher template frontend (fetemplate).
type SettingController struct {
	settings service.ISettingService
	storage  storage.Storage
	fe       *fetemplate.Service
}

// NewSettingController merakit controller (service + storage + fetemplate di-inject).
func NewSettingController(settings service.ISettingService, store storage.Storage, fe *fetemplate.Service) *SettingController {
	return &SettingController{settings: settings, storage: store, fe: fe}
}

// Index → GET /admin/v1/setting (form pengaturan + pilihan tema + katalog FE).
func (ctl *SettingController) Index(c *gin.Context) {
	setting, err := ctl.settings.Get(c.Request.Context())
	if err != nil {
		c.Error(err)
		return
	}
	data := gin.H{
		"title":   "Pengaturan",
		"active":  "setting",
		"setting": setting,
		"themes":  ctl.settings.Themes(),
	}
	ctl.attachFeCatalog(c, setting.FeTemplate, data)
	view.RenderView(c, "setting/index", data)
}

// attachFeCatalog menyuntikkan data katalog template frontend (paginasi + search)
// ke locals view. Aman bila fetemplate tak tersedia (data kosong → seksi disembunyikan).
func (ctl *SettingController) attachFeCatalog(c *gin.Context, feTemplate string, data gin.H) {
	active := fetemplate.ResolveActive(feTemplate)
	data["feActiveSlug"] = active
	data["feEnabled"] = ctl.fe != nil
	if ctl.fe == nil {
		data["feTemplates"] = nil
		return
	}
	page, _ := strconv.Atoi(c.Query("fe_page"))
	if page < 1 {
		page = 1
	}
	search := c.Query("fe_search")
	category := c.Query("fe_category")
	items, total := ctl.fe.Paginate(c.Request.Context(), search, category, page, feSwitcherPerPage, active)
	lastPage := (total + feSwitcherPerPage - 1) / feSwitcherPerPage
	if lastPage < 1 {
		lastPage = 1
	}
	// Jendela paginasi (cur-2..cur+2 di-clamp) + first/last ellipsis — sama NodeAdmin.
	from := page - 2
	if from < 1 {
		from = 1
	}
	to := page + 2
	if to > lastPage {
		to = lastPage
	}
	window := make([]int, 0, to-from+1)
	for i := from; i <= to; i++ {
		window = append(window, i)
	}
	data["feTemplates"] = items
	data["feCategories"] = ctl.fe.Categories(c.Request.Context())
	data["feSearch"] = search
	data["feCategory"] = category
	data["fePage"] = page
	data["feLastPage"] = lastPage
	data["feTotal"] = total
	data["fePageSize"] = feSwitcherPerPage
	data["fePageWindow"] = window
	data["fePageFrom"] = from
	data["fePageTo"] = to
}

// Update → POST /admin/v1/setting (PRG: simpan lalu redirect balik). Bila ada
// file logo, divalidasi (magic-byte) + disimpan. Template FE terpilih (hidden
// input fe_template) ikut ter-submit → diunduh & di-cache saat Save (NodeAdmin).
func (ctl *SettingController) Update(c *gin.Context) {
	var in dto.UpdateSettingInput
	_ = c.ShouldBind(&in)
	sess := sessions.Default(c)

	// Validasi inline per-field (padanan SettingValidator NodeAdmin): theme,
	// fe_template, dan tiga gambar (magic-byte). Error dikumpulkan → ditampilkan
	// inline (`is-invalid`/`invalid-feedback`) + form diisi ulang (old input).
	errs := map[string]string{}
	if in.Theme != "" && !theme.IsValid(in.Theme) {
		errs["theme"] = "Tema tidak dikenali."
	}
	if in.FeTemplate != "" && !fetemplate.IsValidSlug(in.FeTemplate) {
		errs["fe_template"] = "Template tidak dikenali."
	}
	for field, dst := range map[string]*string{"icon": &in.Icon, "logo": &in.Logo, "favicon": &in.Favicon, "login_image": &in.LoginImage} {
		url, uerr := ctl.uploadImage(c, field)
		if uerr != nil {
			errs[field] = errMessage(uerr)
			continue
		}
		if url != "" {
			*dst = url
		}
	}
	if len(errs) > 0 {
		middleware.SetFieldErrors(sess, errs, settingOld(c))
		setFlashError(sess, "Please check the marked fields.")
		c.Redirect(http.StatusFound, "/admin/v1/setting")
		return
	}

	if _, err := ctl.settings.Update(c.Request.Context(), in, actorID(c)); err != nil {
		setFlashError(sess, errMessage(err))
		c.Redirect(http.StatusFound, "/admin/v1/setting")
		return
	}

	// Unduh + cache template FE terpilih saat Save (sama NodeAdmin "diunduh saat
	// Save"). Best-effort: gagal unduh → setting tetap tersimpan, landing fallback.
	if ctl.fe != nil && in.FeTemplate != "" {
		if err := ctl.fe.Ensure(c.Request.Context(), in.FeTemplate); err != nil {
			setFlashError(sess, "Pengaturan disimpan; template '"+in.FeTemplate+"' gagal diunduh (landing pakai default).")
			c.Redirect(http.StatusFound, "/admin/v1/setting")
			return
		}
	}
	setFlashSuccess(sess, "Save Setting Success.")
	c.Redirect(http.StatusFound, "/admin/v1/setting")
}

// uploadImage memproses file form opsional bernama field: validasi (magic-byte)
// + simpan → URL. Kosong (tak diunggah) → ("", nil).
func (ctl *SettingController) uploadImage(c *gin.Context, field string) (string, error) {
	fh, err := c.FormFile(field)
	if err != nil || fh == nil {
		return "", nil
	}
	f, oerr := fh.Open()
	if oerr != nil {
		return "", oerr
	}
	defer f.Close()
	return storage.ValidateAndSave(c.Request.Context(), ctl.storage, f)
}

// --- helper flash & actor (paket web access dipakai lewat re-deklarasi lokal) ---

func setFlashSuccess(sess sessions.Session, msg string) {
	middleware.SetFlashSuccess(sess, msg)
}

func setFlashError(sess sessions.Session, msg string) {
	middleware.SetFlashError(sess, msg)
}

func errMessage(err error) string {
	if ae, ok := apperr.As(err); ok {
		return ae.Message
	}
	return "Terjadi kesalahan."
}

func actorID(c *gin.Context) string {
	if u := accessmw.UserFrom(c); u != nil {
		return u.ID
	}
	return ""
}

// settingOld menangkap nilai form teks yang disubmit (untuk repopulasi inline
// saat validasi gagal — padanan `req.session.old` NodeAdmin).
func settingOld(c *gin.Context) map[string]string {
	keys := []string{"initial", "name", "description", "phone", "address", "email", "copyright", "theme", "fe_template", "favicon"}
	old := make(map[string]string, len(keys))
	for _, k := range keys {
		if v := c.PostForm(k); v != "" {
			old[k] = v
		}
	}
	return old
}
