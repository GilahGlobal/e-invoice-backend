package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	ActivitySingleInvoiceUpload = "single_invoice_upload"
	ActivityBulkInvoiceUpload   = "bulk_invoice_upload"
	ActivityInvitationAccepted  = "invitation_accepted"
	ActivityInvitationRejected  = "invitation_rejected"
	ActivityBusinessRemoved     = "business_removed"
)

// AggregatorActivityLog is an audit trail for aggregator actions
type AggregatorActivityLog struct {
	ID           string    `gorm:"type:uuid;primaryKey;unique;not null" json:"id"`
	AggregatorID string    `gorm:"column:aggregator_id;type:uuid;not null;index" json:"aggregator_id"`
	BusinessID   string    `gorm:"column:business_id;type:uuid;not null;index" json:"business_id"`
	Action       string    `gorm:"column:action;type:varchar(100);not null" json:"action"`
	Details      string    `gorm:"column:details;type:text" json:"details"`
	CreatedAt    time.Time `gorm:"column:created_at;not null;autoCreateTime" json:"created_at"`
}

// BeforeCreate sets the ID if not provided
func (al *AggregatorActivityLog) BeforeCreate(tx *gorm.DB) error {
	if al.ID == "" {
		al.ID = uuid.New().String()
	}
	return nil
}
