// Package model berisi entity GORM modul access (User, Role, Permission).
// Tipe kolom dijaga PORTABEL (string/text/int/bool/timestamp abstrak) — tanpa
// tipe vendor (longtext/datetime) atau collation hardcoded. Checker menolak
// pelanggaran ini agar app tetap multi-DB (MySQL/Postgres/SQLite).
package model

import "time"

// Permission = satu izin granular (mis. "user.create").
// Skema KANONIK lintas-port (1:1 NodeAdmin): name index NON-unik, kolom `desc`
// (column:desc), guard_name (web/api, filter), method/status, +created_by/updated_by.
type Permission struct {
	ID          string    `gorm:"type:varchar(36);primaryKey" json:"id"`
	Name        string    `gorm:"type:varchar(255);index" json:"name"`
	GuardName   string    `gorm:"column:guard_name;type:varchar(20);index;default:web" json:"guard_name"`
	Method      string    `gorm:"type:varchar(255);index" json:"method"`
	Status      string    `gorm:"type:varchar(20);index;default:Active" json:"status"`
	Description string    `gorm:"column:desc;type:varchar(255);index" json:"desc"`
	CreatedBy   string    `gorm:"type:varchar(36)" json:"created_by"`
	UpdatedBy   string    `gorm:"type:varchar(36)" json:"updated_by"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`

	Roles []Role `gorm:"many2many:roles_permissions;" json:"-"`
}

// TableName memetakan ke tabel 'permissions'.
func (Permission) TableName() string { return "permissions" }
