// Package errors menyediakan tipe error aplikasi terpusat (padanan AppError di
// NodeAdmin src/errors/AppError.ts).
//
// Idiom Go: di Go, `return error` adalah idiom wajib (tak seperti larangan di
// NodeAdmin yang melarang `return error`). Padanan konsep "error terpusat" di
// sini adalah:
//   - Service mengembalikan *AppError (membawa HTTP status + pesan publik).
//   - Controller TIDAK memetakan error secara manual ke status — cukup
//     `c.Error(err)` lalu middleware ErrorHandler yang memetakan ke HTTP.
//   - DILARANG (di-enforce checker): service mengembalikan error telanjang
//     (errors.New/fmt.Errorf) yang lalu di-handle status-nya di controller.
//
// Dengan begitu logika bisnis tetap memutuskan "jenis kegagalan" (404/409/422/…)
// lewat konstruktor di bawah, dan presentasi HTTP tetap di satu tempat.
package errors

import (
	"errors"
	"fmt"
	"net/http"
)

// AppError adalah error domain dengan status HTTP + pesan aman-untuk-user.
type AppError struct {
	Status  int    // kode HTTP yang akan dikirim
	Message string // pesan generik untuk user (aman ditampilkan)
	// Detail hanya untuk log internal — TIDAK dikirim ke user di production.
	Detail string
	// Fields menampung error validasi per-field (opsional).
	Fields map[string]string
	// err menyimpan error penyebab (untuk errors.Is/As & log).
	err error
}

func (e *AppError) Error() string {
	if e.Detail != "" {
		return fmt.Sprintf("%s: %s", e.Message, e.Detail)
	}
	return e.Message
}

// Unwrap mendukung errors.Is / errors.As terhadap penyebab.
func (e *AppError) Unwrap() error { return e.err }

// WithDetail menambahkan detail internal (untuk log), mengembalikan diri.
func (e *AppError) WithDetail(detail string) *AppError {
	e.Detail = detail
	return e
}

// WithCause membungkus error penyebab.
func (e *AppError) WithCause(err error) *AppError {
	e.err = err
	if e.Detail == "" && err != nil {
		e.Detail = err.Error()
	}
	return e
}

// New membuat AppError dengan status & pesan kustom.
func New(status int, message string) *AppError {
	return &AppError{Status: status, Message: message}
}

// --- Konstruktor padanan kelas error NodeAdmin ---

// NotFound → 404.
func NotFound(message string) *AppError {
	if message == "" {
		message = "Resource not found"
	}
	return &AppError{Status: http.StatusNotFound, Message: message}
}

// Conflict → 409.
func Conflict(message string) *AppError {
	if message == "" {
		message = "Resource conflict"
	}
	return &AppError{Status: http.StatusConflict, Message: message}
}

// Validation → 422 (input tak valid). Fields opsional (error per-field).
func Validation(message string, fields map[string]string) *AppError {
	if message == "" {
		message = "Validation failed"
	}
	return &AppError{Status: http.StatusUnprocessableEntity, Message: message, Fields: fields}
}

// Unauthorized → 401.
func Unauthorized(message string) *AppError {
	if message == "" {
		message = "Unauthorized"
	}
	return &AppError{Status: http.StatusUnauthorized, Message: message}
}

// Forbidden → 403.
func Forbidden(message string) *AppError {
	if message == "" {
		message = "Forbidden"
	}
	return &AppError{Status: http.StatusForbidden, Message: message}
}

// BadRequest → 400.
func BadRequest(message string) *AppError {
	if message == "" {
		message = "Bad request"
	}
	return &AppError{Status: http.StatusBadRequest, Message: message}
}

// Internal → 500 (pesan generik; detail hanya log).
func Internal(detail string) *AppError {
	return &AppError{Status: http.StatusInternalServerError, Message: "Internal server error", Detail: detail}
}

// As mengekstrak *AppError dari rantai error (atau nil bila bukan).
func As(err error) (*AppError, bool) {
	var ae *AppError
	if errors.As(err, &ae) {
		return ae, true
	}
	return nil, false
}
