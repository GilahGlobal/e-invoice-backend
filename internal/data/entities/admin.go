package entities

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AdminRole string

const (
	RoleSuperAdmin AdminRole = "superadmin"
	RoleAdmin      AdminRole = "admin"
)

type Admin struct {
	ID        string         `gorm:"type:uuid;primaryKey;unique;not null" json:"id"`
	Name      string         `gorm:"type:varchar(250);not null" json:"name"`
	Email     string         `gorm:"type:varchar(100);unique;not null" json:"email"`
	Password  string         `gorm:"type:text;not null" json:"-"`
	Role      AdminRole      `gorm:"type:varchar(20);not null;default:'admin'" json:"role"`
	CreatedAt time.Time      `gorm:"not null;autoCreateTime" json:"created_at"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

func (a *Admin) BeforeCreate(tx *gorm.DB) error {
	if a.ID == "" {
		a.ID = uuid.New().String()
	}
	return nil
}
