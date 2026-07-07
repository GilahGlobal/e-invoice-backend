package entities

import (
	"time"

	"gorm.io/gorm"
)

type TokenManager struct {
	ID             uint      `gorm:"primaryKey"`
	Provider       string    `gorm:"not null;index:idx_provider_org,unique"`
	OrganizationID string    `gorm:"not null;index:idx_provider_org,unique"`
	RefreshToken   string    `gorm:"not null;index"`
	AccessToken    string    `gorm:"not null"`
	ExpiresAt      time.Time `gorm:"not null"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
	DeletedAt      gorm.DeletedAt `gorm:"index"`
}
