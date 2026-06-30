// Package dto berisi struct input tervalidasi (padanan validator Joi
// stripUnknown di NodeAdmin). Hanya field di DTO yang diterima → anti
// mass-assignment (whitelist). Validasi lewat tag go-playground/validator.
package dto

// CreateUserInput = payload buat user baru.
type CreateUserInput struct {
	Name     string   `json:"name" form:"name" binding:"required,max=50"`
	Email    string   `json:"email" form:"email" binding:"required,email,max=255"`
	Phone    string   `json:"phone" form:"phone" binding:"omitempty,max=15"`
	Password string   `json:"password" form:"password" binding:"required,min=8,max=72"`
	Status   string   `json:"status" form:"status" binding:"omitempty,oneof=Active Inactive"`
	Timezone string   `json:"timezone" form:"timezone" binding:"omitempty,max=64"`
	Picture  string   `json:"picture" form:"picture" binding:"omitempty,max=255"`
	RoleIDs  []string `json:"role_ids" form:"roles[]" binding:"omitempty,dive,max=36"`
	// Blocked: checkbox form (value "1" → true; absen → false). BlockedReason: alasan blokir.
	Blocked       bool   `json:"blocked" form:"blocked"`
	BlockedReason string `json:"blocked_reason" form:"blocked_reason" binding:"omitempty,max=255"`
	// PasswordConfirmation: hanya divalidasi di controller web (cocok dgn Password).
	PasswordConfirmation string `json:"password_confirmation" form:"password_confirmation" binding:"omitempty"`
}

// UpdateUserInput = payload ubah user. Password opsional (kosong = tak diubah).
type UpdateUserInput struct {
	Name     string   `json:"name" form:"name" binding:"required,max=50"`
	Email    string   `json:"email" form:"email" binding:"required,email,max=255"`
	Phone    string   `json:"phone" form:"phone" binding:"omitempty,max=15"`
	Password string   `json:"password" form:"password" binding:"omitempty,min=8,max=72"`
	Status   string   `json:"status" form:"status" binding:"omitempty,oneof=Active Inactive"`
	Timezone string   `json:"timezone" form:"timezone" binding:"omitempty,max=64"`
	Picture  string   `json:"picture" form:"picture" binding:"omitempty,max=255"`
	RoleIDs  []string `json:"role_ids" form:"roles[]" binding:"omitempty,dive,max=36"`
	// Blocked: checkbox form (value "1" → true; absen → false). BlockedReason: alasan blokir.
	Blocked       bool   `json:"blocked" form:"blocked"`
	BlockedReason string `json:"blocked_reason" form:"blocked_reason" binding:"omitempty,max=255"`
	// PasswordConfirmation: hanya divalidasi di controller web (cocok dgn Password).
	PasswordConfirmation string `json:"password_confirmation" form:"password_confirmation" binding:"omitempty"`
}

// ListQuery = parameter list (paginasi + search/filter).
//
// Dua gaya hidup berdampingan:
//   - API JSON  : page / per_page / search (sederhana).
//   - Web admin : q_page / q_page_size + filter PER-KOLOM (q_code, q_name, …) —
//     replika 1:1 tabel index NodeAdmin. Normalize() memetakan q_page/q_page_size
//     ke Page/PerPage agar helpers.Paginate tetap dipakai.
type ListQuery struct {
	Page    int    `form:"page"`
	PerPage int    `form:"per_page"`
	Search  string `form:"search"`

	// Filter web per-kolom (NodeAdmin q_*).
	QPage     int    `form:"q_page"`
	QPageSize int    `form:"q_page_size"`
	QCode     string `form:"q_code"`
	QName     string `form:"q_name"`
	QPhone    string `form:"q_phone"`
	QEmail    string `form:"q_email"`
	QStatus   string `form:"q_status"`
	QRole     string `form:"q_role"`
	QMethod   string `form:"q_method"`
	QDesc     string `form:"q_desc"`
	QGuard    string `form:"q_guard"`
}

// Normalize menyalin parameter web (q_page/q_page_size) ke Page/PerPage bila
// terisi, sehingga jalur paginasi generik tak perlu tahu asal parameter.
func (q *ListQuery) Normalize() {
	if q.QPage > 0 {
		q.Page = q.QPage
	}
	if q.QPageSize > 0 {
		q.PerPage = q.QPageSize
	}
}
