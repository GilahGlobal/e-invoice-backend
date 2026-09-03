package entities

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	RoleSuperAdmin = "superadmin"
	RoleAdmin      = "admin"
)

type Role struct {
	ID          uuid.UUID      `gorm:"type:uuid;primaryKey;unique;not null" json:"id"`
	Name        string         `gorm:"type:varchar(50);unique;not null" json:"name"`
	Description string         `gorm:"type:varchar(250)" json:"description"`
	CreatedAt   time.Time      `gorm:"not null;autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

func (r *Role) BeforeCreate(tx *gorm.DB) error {
	if r.ID == uuid.Nil {
		r.ID = uuid.New()
	}
	return nil
}
