package entities

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// SubscriptionReminderLog records each reminder email sent for a subscription
type SubscriptionReminderLog struct {
	ID               string    `gorm:"column:id;type:uuid;not null;primaryKey;unique" json:"id"`
	SubscriptionID   string    `gorm:"column:subscription_id;type:uuid;index;not null" json:"subscription_id"`
	BusinessID       string    `gorm:"column:business_id;type:uuid;index;not null" json:"business_id"`
	AggregatorID     string    `gorm:"column:aggregator_id;type:uuid;index" json:"aggregator_id"`
	RecipientEmail   string    `gorm:"column:recipient_email;type:varchar(150);not null" json:"recipient_email"`
	RecipientType    string    `gorm:"column:recipient_type;type:varchar(30);not null" json:"recipient_type"` // "business" or "aggregator"
	ReminderNumber   int       `gorm:"column:reminder_number;type:int;not null" json:"reminder_number"`
	DaysBeforeExpiry int       `gorm:"column:days_before_expiry;type:int;not null" json:"days_before_expiry"`
	NextBillingDate  time.Time `gorm:"column:next_billing_date;type:timestamp;not null" json:"next_billing_date"`
	Status           string    `gorm:"column:status;type:varchar(30);not null" json:"status"` // "sent", "failed"
	ErrorMessage     *string   `gorm:"column:error_message;type:text" json:"error_message,omitempty"`
	SentAt           time.Time `gorm:"column:sent_at;autoCreateTime" json:"sent_at"`
}

func (l *SubscriptionReminderLog) BeforeCreate(tx *gorm.DB) error {
	if l.ID == "" {
		l.ID = uuid.New().String()
	}
	return nil
}
