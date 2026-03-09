package models

import (
	"time"
)

type Subscription struct {
	ID                string    `gorm:"column:id; type:uuid; not null; primaryKey; unique;" json:"id"`
	SmeID             string    `gorm:"column:sme_id; type:uuid; not null" json:"sme_id"`
	IsActive          bool      `gorm:"column:is_active; type:bool; default:false; not null" json:"is_active"`
	Plan              string    `gorm:"column:plan; type:text" json:"plan"`
	TotalInvoices     int       `gorm:"column:total_invoices; type:int" json:"total_invoices"`
	RemainingInvoices int       `gorm:"column:remaining_invoices; type:int" json:"remaining_invoices"`
	UsedInvoices      int       `gorm:"column:used_invoices; type:int; default:0" json:"used_invoices"`
	NextBillingDate   time.Time `gorm:"column:next_billing_date; type:timestamp" json:"next_billing_date"`
	CreatedAt         time.Time `gorm:"column:created_at; autoCreateTime" json:"created_at"`
	UpdatedAt         time.Time `gorm:"column:updated_at; autoUpdateTime" json:"updated_at"`
}
