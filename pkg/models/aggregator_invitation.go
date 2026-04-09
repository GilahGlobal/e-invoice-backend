package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	InvitationStatusPending  = "pending"
	InvitationStatusAccepted = "accepted"
	InvitationStatusRejected = "rejected"
	InvitationStatusRevoked  = "revoked"
)

// AggregatorInvitation tracks invitations from a Business to an Aggregator
type AggregatorInvitation struct {
	ID           string         `gorm:"type:uuid;primaryKey;unique;not null" json:"id"`
	BusinessID   string         `gorm:"column:business_id;type:uuid;not null;index" json:"business_id"`
	AggregatorID string         `gorm:"column:aggregator_id;type:uuid;not null;index" json:"aggregator_id"`
	Status       string         `gorm:"column:status;type:varchar(20);not null;default:'pending'" json:"status"`
	InviteToken  string         `gorm:"column:invite_token;type:varchar(100);uniqueIndex" json:"-"`
	AcceptedAt   *time.Time     `gorm:"column:accepted_at" json:"accepted_at,omitempty"`
	RejectedAt   *time.Time     `gorm:"column:rejected_at" json:"rejected_at,omitempty"`
	CreatedAt    time.Time      `gorm:"column:created_at;not null;autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time      `gorm:"column:updated_at;null;autoUpdateTime" json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`

	// Relations (loaded via Preload when needed)
	Business   Business   `gorm:"foreignKey:BusinessID" json:"business,omitempty"`
	Aggregator Aggregator `gorm:"foreignKey:AggregatorID" json:"aggregator,omitempty"`
}

// BeforeCreate sets the ID if not provided
func (ai *AggregatorInvitation) BeforeCreate(tx *gorm.DB) error {
	if ai.ID == "" {
		ai.ID = uuid.New().String()
	}
	return nil
}
