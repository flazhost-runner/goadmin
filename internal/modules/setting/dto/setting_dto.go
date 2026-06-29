// Package dto berisi input tervalidasi modul setting (whitelist field → anti
// mass-assignment). Semua field opsional: update bersifat PARSIAL — hanya
// field non-kosong yang ditimpa (padanan removeEmptyFields di NodeAdmin).
package dto

// UpdateSettingInput = payload ubah setting global.
type UpdateSettingInput struct {
	Initial     string `json:"initial" form:"initial" binding:"omitempty,max=255"`
	Name        string `json:"name" form:"name" binding:"omitempty,max=255"`
	Description string `json:"description" form:"description" binding:"omitempty"`
	Phone       string `json:"phone" form:"phone" binding:"omitempty,max=255"`
	Address     string `json:"address" form:"address" binding:"omitempty,max=255"`
	Email       string `json:"email" form:"email" binding:"omitempty,email,max=255"`
	Copyright   string `json:"copyright" form:"copyright" binding:"omitempty,max=255"`
	// Icon/Logo/Favicon/LoginImage = URL hasil upload (diisi controller setelah validasi+simpan).
	Icon       string `json:"icon" form:"icon" binding:"omitempty,max=255"`
	Logo       string `json:"logo" form:"logo" binding:"omitempty,max=255"`
	Favicon    string `json:"favicon" form:"favicon" binding:"omitempty,max=255"`
	LoginImage string `json:"login_image" form:"login_image" binding:"omitempty,max=255"`
	// Theme & FeTemplate = pilihan switcher (divalidasi terhadap katalog di service).
	Theme      string `json:"theme" form:"theme" binding:"omitempty,max=20"`
	FeTemplate string `json:"fe_template" form:"fe_template" binding:"omitempty,max=80"`
}
