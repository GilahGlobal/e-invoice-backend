package models

import (
	"encoding/json"
	"time"
)

type Transaction struct {
	ID                string          `gorm:"column:id; type:uuid; not null; primaryKey; unique;" json:"id"`
	BusinessID        string          `gorm:"column:sme_id; type:uuid; not null; index" json:"business_id"`
	AggregatorID      string          `gorm:"column:aggregator_id; type:uuid; not null; index" json:"aggregator_id"`
	Reference         string          `gorm:"column:reference; type:varchar(120); uniqueIndex; not null" json:"reference"`
	Provider          string          `gorm:"column:provider; type:varchar(50); not null" json:"provider"`
	ProviderReference string          `gorm:"column:provider_reference; type:varchar(120)" json:"provider_reference"`
	Status            string          `gorm:"column:status; type:varchar(50); not null; default:pending" json:"status"`
	Amount            float64         `gorm:"column:amount; type:decimal(10,2)" json:"amount"`
	Currency          string          `gorm:"column:currency; type:varchar(10); default:'NGN'" json:"currency"`
	PlanID            string          `gorm:"column:plan_id; type:varchar(100)" json:"plan_id"`
	Plan              string          `gorm:"column:plan; type:text" json:"plan"`
	AuthorizationURL  string          `gorm:"column:authorization_url; type:text" json:"authorization_url"`
	AccessCode        string          `gorm:"column:access_code; type:text" json:"access_code"`
	GatewayResponse   string          `gorm:"column:gateway_response; type:text" json:"gateway_response"`
	ProviderPayload   json.RawMessage `gorm:"column:provider_payload; type:jsonb" json:"provider_payload,omitempty"`
	CreatedAt         time.Time       `gorm:"column:created_at; autoCreateTime" json:"created_at"`
	UpdatedAt         time.Time       `gorm:"column:updated_at; autoUpdateTime" json:"updated_at"`
}
