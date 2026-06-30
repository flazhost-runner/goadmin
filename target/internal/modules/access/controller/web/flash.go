package web

import (
	"github.com/gin-contrib/sessions"

	apperr "goadmin/internal/errors"
	"goadmin/internal/middleware"
)

// Helper flash (tulis ke sesi; middleware.Flash memindahkannya ke view pada
// request berikutnya). Pola PRG: set flash → redirect → halaman target tampil.

func setFlashSuccess(sess sessions.Session, msg string) {
	middleware.SetFlashSuccess(sess, msg)
}

func setFlashError(sess sessions.Session, msg string) {
	middleware.SetFlashError(sess, msg)
}

// errMessage mengambil pesan publik dari *AppError (fallback generik).
func errMessage(err error) string {
	if ae, ok := apperr.As(err); ok {
		return ae.Message
	}
	return "Terjadi kesalahan."
}
