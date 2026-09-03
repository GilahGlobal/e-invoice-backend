package entities

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type BusinessAggregatorHistory struct {
	ID           uuid.UUID      `gorm:"type:uuid;primaryKey;not null" json:"id"`
	BusinessID   string         `gorm:"type:uuid;not null;index" json:"business_id"`
	AggregatorID string         `gorm:"type:uuid;index" json:"aggregator_id"`
	Action       string         `gorm:"type:varchar(20);not null" json:"action"` // "assigned" or "removed"
	CreatedAt    time.Time      `gorm:"column:created_at;not null;autoCreateTime;index" json:"created_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}

func (h *BusinessAggregatorHistory) BeforeCreate(tx *gorm.DB) error {
	if h.ID == uuid.Nil {
		h.ID = uuid.New()
	}
	return nil
}
