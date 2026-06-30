package web

import (
	"net/http"
	"net/url"
	"strconv"

	"github.com/gin-gonic/gin"

	"goadmin/internal/modules/access/dto"
)

// redirectBack mengarahkan kembali ke halaman asal (Referer) — padanan
// `res.redirect(req.get('Referrer') || fallback)` NodeAdmin. Dipakai aksi
// assign/unassign permission agar kembali ke halaman + filter yang sama.
func redirectBack(c *gin.Context, fallback string) {
	target := c.Request.Referer()
	if target == "" {
		target = fallback
	}
	c.Redirect(http.StatusFound, target)
}

// bindListQuery membaca parameter tabel index (q_page/q_page_size + filter
// per-kolom q_*) lalu mengembalikan query ternormalisasi + "qbase": potongan
// query-string berisi seluruh filter aktif (tanpa q_page) untuk menjaga filter
// tetap menempel saat pindah halaman pagination. qbase diakhiri "&" bila tak
// kosong sehingga template cukup menulis `?{{.qbase}}q_page=N`.
func bindListQuery(c *gin.Context) (dto.ListQuery, string) {
	var q dto.ListQuery
	_ = c.ShouldBindQuery(&q)
	q.Normalize()

	vals := url.Values{}
	set := func(k, v string) {
		if v != "" {
			vals.Set(k, v)
		}
	}
	set("q_code", q.QCode)
	set("q_name", q.QName)
	set("q_phone", q.QPhone)
	set("q_email", q.QEmail)
	set("q_status", q.QStatus)
	set("q_role", q.QRole)
	set("q_method", q.QMethod)
	set("q_desc", q.QDesc)
	set("q_guard", q.QGuard)
	if q.QPageSize > 0 {
		vals.Set("q_page_size", strconv.Itoa(q.QPageSize))
	}
	base := vals.Encode()
	if base != "" {
		base += "&"
	}
	return q, base
}

// selectedIDs mengambil daftar id tercentang dari form "selected[]" (aksi
// bulk "Delete Selected" di tabel index).
func selectedIDs(c *gin.Context) []string {
	return c.PostFormArray("selected[]")
}

// filterMap mengekspos nilai filter aktif ke template (mengisi ulang input
// pencarian per-kolom setelah submit).
func filterMap(q dto.ListQuery) gin.H {
	return gin.H{
		"q_code": q.QCode, "q_name": q.QName, "q_phone": q.QPhone,
		"q_email": q.QEmail, "q_status": q.QStatus, "q_role": q.QRole,
		"q_method": q.QMethod, "q_desc": q.QDesc, "q_guard": q.QGuard, "q_page_size": q.QPageSize,
	}
}
