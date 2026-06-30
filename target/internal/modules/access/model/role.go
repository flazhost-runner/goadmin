package model

import "time"

// Role mengelompokkan permission dan diberikan ke user (RBAC).
// Skema KANONIK lintas-port (1:1 NodeAdmin): kolom `desc` (Go field Description
// dipetakan via column:desc), created_by/updated_by.
type Role struct {
	ID          string    `gorm:"type:varchar(36);primaryKey" json:"id"`
	Name        string    `gorm:"type:varchar(255);uniqueIndex" json:"name"`
	GuardName   string    `gorm:"type:varchar(50);index;default:web" json:"guard_name"`
	Status      string    `gorm:"type:varchar(20);index;default:Active" json:"status"`
	Description string    `gorm:"column:desc;type:varchar(255);index" json:"desc"`
	CreatedBy   string    `gorm:"type:varchar(36)" json:"created_by"`
	UpdatedBy   string    `gorm:"type:varchar(36)" json:"updated_by"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`

	Permissions []Permission `gorm:"many2many:roles_permissions;" json:"permissions,omitempty"`
	Users       []User       `gorm:"many2many:users_roles;" json:"-"`
}

func (Role) TableName() string { return "roles" }

// HasPermission true bila role memuat permission untuk route `name` + `method`
// (route-driven a la NodeAdmin: cocokkan nama-route DAN HTTP method).
func (r *Role) HasPermission(name, method string) bool {
	for _, p := range r.Permissions {
		if p.Name == name && p.Method == method {
			return true
		}
	}
	return false
}
