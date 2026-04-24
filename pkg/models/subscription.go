package models

import (
	"time"
)

type Subscription struct {
	ID                string    `gorm:"column:id; type:uuid; not null; primaryKey; unique;" json:"id"`
	BusinessID        string    `gorm:"column:business_id; type:uuid; not null; index:idx_subscription_business_aggregator" json:"business_id"`
	AggregatorID      string    `gorm:"column:aggregator_id; type:uuid; not null; index:idx_subscription_business_aggregator" json:"aggregator_id"`
	IsActive          bool      `gorm:"column:is_active; type:bool; default:false; not null" json:"is_active"`
	PlanID            string    `gorm:"column:plan_id; type:varchar(100)" json:"plan_id"`
	Plan              string    `gorm:"column:plan; type:text" json:"plan"`
	TotalInvoices     int       `gorm:"column:total_invoices; type:int" json:"total_invoices"`
	RemainingInvoices int       `gorm:"column:remaining_invoices; type:int" json:"remaining_invoices"`
	UsedInvoices      int       `gorm:"column:used_invoices; type:int; default:0" json:"used_invoices"`
	NextBillingDate   time.Time `gorm:"column:next_billing_date; type:timestamp" json:"next_billing_date"`
	CreatedAt         time.Time `gorm:"column:created_at; autoCreateTime" json:"created_at"`
	UpdatedAt         time.Time `gorm:"column:updated_at; autoUpdateTime" json:"updated_at"`
}
