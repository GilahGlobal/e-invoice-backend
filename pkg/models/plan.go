package models

import "time"

type SubscriptionPlan struct {
	ID            string    `gorm:"column:id; type:varchar(100); not null; primaryKey" json:"id"`
	Name          string    `gorm:"column:name; type:varchar(120); not null" json:"name"`
	Amount        float64   `gorm:"column:amount; type:decimal(12,2); not null" json:"amount"`
	IsActive      bool      `gorm:"column:is_active; type:bool; not null; default:false" json:"is_active"`
	TotalInvoices int       `gorm:"column:total_invoices; type:int; not null" json:"total_invoices"`
	BillingCycle  int       `gorm:"column:billing_cycle; type:int; not null" json:"billing_cycle"`
	CreatedAt     time.Time `gorm:"column:created_at; autoCreateTime" json:"created_at"`
	UpdatedAt     time.Time `gorm:"column:updated_at; autoUpdateTime" json:"updated_at"`
}
