package entities

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ApiLog struct {
	ID         uuid.UUID      `gorm:"type:uuid;primaryKey;not null" json:"id"`
	Path       string         `gorm:"column:path;type:varchar(255);not null;index" json:"path"`
	Method     string         `gorm:"column:method;type:varchar(10);not null;index" json:"method"`
	StatusCode int            `gorm:"column:status_code;type:int;not null" json:"status_code"`
	ClientIP   string         `gorm:"column:client_ip;type:varchar(45)" json:"client_ip"`
	UserAgent  string         `gorm:"column:user_agent;type:text" json:"user_agent"`
	CreatedAt  time.Time      `gorm:"column:created_at;not null;autoCreateTime;index" json:"created_at"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`
}

func (a *ApiLog) BeforeCreate(tx *gorm.DB) error {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	return nil
}
