package models

import (
	"encoding/json"
	"time"
)

type TransactionStatus string

const (
	TransactionStatusInitialized TransactionStatus = "initialized"
	TransactionStatusSuccess     TransactionStatus = "success"
	TransactionStatusFailed      TransactionStatus = "failed"
	TransactionStatusProcessing  TransactionStatus = "processing"
)

type Transaction struct {
	ID                       string            `gorm:"column:id; type:uuid; not null; primaryKey; unique;" json:"id"`
	BusinessID               string            `gorm:"column:business_id; type:uuid; not null; index" json:"business_id"`
	AggregatorID             string            `gorm:"column:aggregator_id; type:uuid; not null; index" json:"aggregator_id"`
	Reference                string            `gorm:"column:reference; type:varchar(120); uniqueIndex; not null" json:"reference"`
	Provider                 string            `gorm:"column:provider; type:varchar(50); not null" json:"provider"`
	Status                   TransactionStatus `gorm:"column:status; type:varchar(50); not null; default:pending" json:"status"`
	Amount                   float64           `gorm:"column:amount; type:decimal(10,2)" json:"amount"`
	Currency                 string            `gorm:"column:currency; type:varchar(10); default:'NGN'" json:"currency"`
	PlanID                   string            `gorm:"column:plan_id; type:varchar(100)" json:"plan_id"`
	Plan                     string            `gorm:"column:plan; type:text" json:"plan"`
	ErrorMessage             *string           `gorm:"column:error_message; type:text" json:"error_message"`
	ProviderResponseMetadata json.RawMessage   `gorm:"column:provider_response_metadata; type:jsonb" json:"provider_response_metadata,omitempty"`
	CreatedAt                time.Time         `gorm:"column:created_at; autoCreateTime" json:"created_at"`
	UpdatedAt                time.Time         `gorm:"column:updated_at; autoUpdateTime" json:"updated_at"`
}
