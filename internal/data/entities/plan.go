package entities

import (
	"encoding/json"
	"sort"
	"time"

	"gorm.io/datatypes"
)

type SubscriptionPlan struct {
	ID                 string         `gorm:"column:id; type:varchar(100); not null; primaryKey" json:"id"`
	Name               string         `gorm:"column:name; type:varchar(120); not null" json:"name"`
	Amount             float64        `gorm:"column:amount; type:decimal(12,2); not null" json:"amount"`
	IsActive           bool           `gorm:"column:is_active; type:bool; not null; default:false" json:"is_active"`
	TotalInvoices      int            `gorm:"column:total_invoices; type:int; not null" json:"total_invoices"`
	BillingCycle       int            `gorm:"column:billing_cycle; type:int; not null" json:"billing_cycle"`
	ReminderDaysBefore datatypes.JSON `gorm:"column:reminder_days_before; type:jsonb; default:'[7,3,1]'" json:"reminder_days_before" swaggertype:"array,integer"`
	CreatedAt          time.Time      `gorm:"column:created_at; autoCreateTime" json:"created_at"`
	UpdatedAt          time.Time      `gorm:"column:updated_at; autoUpdateTime" json:"updated_at"`
}

// GetReminderDays returns the sorted descending list of days before expiry when reminders should be sent.
func (p *SubscriptionPlan) GetReminderDays() []int {
	if len(p.ReminderDaysBefore) == 0 {
		return []int{7, 3, 1}
	}
	var days []int
	if err := json.Unmarshal(p.ReminderDaysBefore, &days); err != nil || len(days) == 0 {
		return []int{7, 3, 1}
	}
	sort.Sort(sort.Reverse(sort.IntSlice(days)))
	return days
}
