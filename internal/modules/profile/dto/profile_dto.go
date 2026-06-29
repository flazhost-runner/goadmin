// Package dto berisi input tervalidasi modul profile (whitelist field → anti
// mass-assignment). CATATAN least-privilege: profil sengaja TIDAK memuat
// status/role — user tak boleh mengubah status/peran dirinya sendiri.
package dto

// UpdateProfileInput = payload ubah profil sendiri (opsional ganti password).
type UpdateProfileInput struct {
	Name     string `json:"name" form:"name" binding:"required,max=50"`
	Email    string `json:"email" form:"email" binding:"required,email,max=255"`
	Phone    string `json:"phone" form:"phone" binding:"omitempty,max=15"`
	Timezone string `json:"timezone" form:"timezone" binding:"omitempty,max=64"`
	// Picture = URL avatar hasil upload (diisi controller setelah validasi+simpan).
	Picture string `json:"picture" form:"picture" binding:"omitempty,max=255"`
	// Password kosong = tidak diubah. Bila diisi, PasswordConfirmation wajib sama.
	Password             string `json:"password" form:"password" binding:"omitempty,min=8,max=72"`
	PasswordConfirmation string `json:"password_confirmation" form:"password_confirmation" binding:"omitempty"`
}
