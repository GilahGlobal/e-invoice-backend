package entities

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// NotificationPreference holds business email notification preferences
type NotificationPreference struct {
	ID                         string         `gorm:"type:uuid;primaryKey;unique;not null" json:"id"`
	BusinessID                 string         `gorm:"column:business_id;type:uuid;uniqueIndex;not null" json:"business_id"`
	InvoiceValidationCompleted bool           `gorm:"column:invoice_validation_completed;type:bool;default:true;not null" json:"invoice_validation_completed"`
	InvoiceValidationFailed    bool           `gorm:"column:invoice_validation_failed;type:bool;default:true;not null" json:"invoice_validation_failed"`
	BulkUploadFinished         bool           `gorm:"column:bulk_upload_finished;type:bool;default:true;not null" json:"bulk_upload_finished"`
	CreatedAt                  time.Time      `gorm:"column:created_at;not null;autoCreateTime" json:"created_at"`
	UpdatedAt                  time.Time      `gorm:"column:updated_at;null;autoUpdateTime" json:"updated_at"`
	DeletedAt                  gorm.DeletedAt `gorm:"index" json:"-"`
}

func (np *NotificationPreference) BeforeCreate(tx *gorm.DB) error {
	if np.ID == "" {
		np.ID = uuid.New().String()
	}
	return nil
}

// DefaultNotificationPreference returns the default notification preferences for a business (all enabled)
func DefaultNotificationPreference(businessID string) *NotificationPreference {
	return &NotificationPreference{
		ID:                         uuid.New().String(),
		BusinessID:                 businessID,
		InvoiceValidationCompleted: true,
		InvoiceValidationFailed:    true,
		BulkUploadFinished:         true,
	}
}
