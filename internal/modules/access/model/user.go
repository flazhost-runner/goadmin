package model

import "time"

// Status user (varchar + konstanta Go, bukan ENUM native — seragam lintas dialek).
const (
	StatusActive   = "Active"
	StatusInactive = "Inactive"
)

// RoleAdministrator = role super-user yang mem-bypass pengecekan permission.
const RoleAdministrator = "Administrator"

// User adalah akun yang bisa login (web sesi / API JWT).
type User struct {
	ID            string     `gorm:"type:varchar(36);primaryKey" json:"id"`
	Code          string     `gorm:"type:varchar(20);uniqueIndex" json:"code"`
	Name          string     `gorm:"type:varchar(50);index" json:"name"`
	Phone         string     `gorm:"type:varchar(15);index" json:"phone"`
	Email         string     `gorm:"type:varchar(255);uniqueIndex" json:"email"`
	EmailVerified *time.Time `json:"email_verified_at,omitempty"`
	// Password hash bcrypt — tak pernah diserialisasi ke JSON.
	Password string `gorm:"type:varchar(255)" json:"-"`
	// OTP reset password: hash + masa berlaku (epoch ms). Tak pernah ke JSON.
	PasswordOTP        string `gorm:"type:varchar(255)" json:"-"`
	PasswordOTPExpires *int64 `gorm:"type:bigint" json:"-"`

	Status        string    `gorm:"type:varchar(20);default:Active;index" json:"status"`
	Picture       string    `gorm:"type:varchar(255)" json:"picture"`
	Blocked       bool      `gorm:"default:false;index" json:"blocked"`
	BlockedReason string    `gorm:"type:varchar(255)" json:"blocked_reason"`
	Timezone      string    `gorm:"type:varchar(64);default:UTC;index" json:"timezone"`
	CreatedBy     string    `gorm:"type:varchar(36)" json:"created_by"`
	UpdatedBy     string    `gorm:"type:varchar(36)" json:"updated_by"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`

	Roles []Role `gorm:"many2many:users_roles;" json:"roles,omitempty"`
}

func (User) TableName() string { return "users" }

// IsAdministrator true bila user punya role Administrator (bypass RBAC).
func (u *User) IsAdministrator() bool {
	for _, r := range u.Roles {
		if r.Name == RoleAdministrator {
			return true
		}
	}
	return false
}

// HasRole true bila user memiliki role dengan nama yang diberikan.
// Dipakai template helper hasRole di FuncMap view.
func (u *User) HasRole(name string) bool {
	for _, r := range u.Roles {
		if r.Name == name {
			return true
		}
	}
	return false
}

// HasAccess true bila user (lewat salah satu role) memiliki permission untuk
// route bernama `name` + HTTP `method` (model route-driven a la NodeAdmin:
// permission.name == nama-route DAN permission.method == method). Administrator
// selalu true (bypass).
func (u *User) HasAccess(name, method string) bool {
	if u.IsAdministrator() {
		return true
	}
	for _, r := range u.Roles {
		if r.HasPermission(name, method) {
			return true
		}
	}
	return false
}
