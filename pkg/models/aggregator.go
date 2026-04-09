package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Aggregator represents an aggregator entity that manages invoices on behalf of businesses
type Aggregator struct {
	ID            string         `gorm:"type:uuid;primaryKey;unique;not null" json:"id"`
	Name          string         `gorm:"column:name;type:varchar(250);not null" json:"name"`
	Email         string         `gorm:"column:email;type:varchar(100);unique;not null" json:"email"`
	Password      string         `gorm:"column:password;type:text;not null" json:"-"`
	CompanyName   string         `gorm:"column:company_name;type:varchar(250);not null" json:"company_name"`
	PhoneNumber   string         `gorm:"column:phone_number;type:varchar(13)" json:"phone_number"`
	EmailVerified bool           `gorm:"column:email_verified;type:bool;default:false" json:"email_verified"`
	IsActive      bool           `gorm:"column:is_active;type:bool;default:true" json:"is_active"`
	CreatedAt     time.Time      `gorm:"column:created_at;not null;autoCreateTime" json:"created_at"`
	UpdatedAt     time.Time      `gorm:"column:updated_at;null;autoUpdateTime" json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
}

// BeforeCreate sets the ID if not provided
func (a *Aggregator) BeforeCreate(tx *gorm.DB) error {
	if a.ID == "" {
		a.ID = uuid.New().String()
	}
	return nil
}
