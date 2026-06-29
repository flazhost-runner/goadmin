package model

import "time"

// Setting adalah konfigurasi global aplikasi — disimpan sebagai BARIS TUNGGAL
// (singleton). Dibaca tiap request (layout/landing) lewat service ber-cache,
// di-update lewat halaman admin. Tipe kolom portabel (varchar/text), tanpa
// collation vendor — seragam lintas dialek.
type Setting struct {
	ID          string `gorm:"type:varchar(36);primaryKey" json:"id"`
	Initial     string `gorm:"type:varchar(255);index" json:"initial"`
	Name        string `gorm:"type:varchar(255);index" json:"name"`
	Description string `gorm:"type:text" json:"description"`
	Icon        string `gorm:"type:varchar(255)" json:"icon"`
	Logo        string `gorm:"type:varchar(255)" json:"logo"`
	Favicon     string `gorm:"type:varchar(255)" json:"favicon"`
	LoginImage  string `gorm:"type:varchar(255)" json:"login_image"`
	Phone       string `gorm:"type:varchar(255)" json:"phone"`
	Address     string `gorm:"type:varchar(255)" json:"address"`
	Email       string `gorm:"type:varchar(255);index" json:"email"`
	Copyright   string `gorm:"type:varchar(255)" json:"copyright"`

	// Theme = nama palet aktif (theme switcher) — divalidasi terhadap katalog
	// theme; layout memetakannya ke CSS variable (ganti tanpa rebuild).
	Theme string `gorm:"type:varchar(20);default:Blue" json:"theme"`
	// FeTemplate = slug template landing aktif (frontend template switcher).
	FeTemplate string `gorm:"type:varchar(80)" json:"fe_template"`

	CreatedBy string    `gorm:"type:varchar(36)" json:"created_by"`
	UpdatedBy string    `gorm:"type:varchar(36)" json:"updated_by"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (Setting) TableName() string { return "settings" }
