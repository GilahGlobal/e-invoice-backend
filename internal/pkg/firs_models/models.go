package firs_models

import (
	"time"
)

type FirsResponse struct {
	Code    int         `json:"code"`
	Data    interface{} `json:"data,omitempty"`
	Message string      `json:"message,omitempty"`
	Error   *ErrorData  `json:"error,omitempty"`
}

type HealthCheck struct {
	Healthy bool `json:"healthy"`
}

type SingleData struct {
	ID          string    `json:"id,omitempty"`
	OK          bool      `json:"ok,omitempty"`
	UP          string    `json:"up,omitempty"`
	Status      string    `json:"status,omitempty"`
	Message     string    `json:"message,omitempty"`
	ReceivedAt  time.Time `json:"received_at,omitempty"`
	EntityID    string    `json:"entity_id,omitempty"`
	AccessToken *string   `json:"access_token,omitempty"`
}

type PaginatedData struct {
	Items      []DetailedEntityData `json:"items"`
	Page       PageInfo             `json:"page"`
	Attributes *string              `json:"attributes"`
}

type DetailedEntityData struct {
	ID             string      `json:"id"`
	Reference      string      `json:"reference"`
	CustomSettings interface{} `json:"custom_settings"`
	CreatedAt      time.Time   `json:"created_at"`
	UpdatedAt      time.Time   `json:"updated_at"`
	IsActive       bool        `json:"is_active"`
	AppReference   string      `json:"app_reference"`
	Businesses     []Business  `json:"businesses"`
}

type Business struct {
	ID                   string    `json:"id,omitempty"`
	Reference            string    `json:"reference,omitempty"`
	Name                 string    `json:"name,omitempty"`
	CustomSettings       *string   `json:"custom_settings,omitempty"`
	CreatedAt            time.Time `json:"created_at,omitempty"`
	UpdatedAt            time.Time `json:"updated_at,omitempty"`
	TIN                  string    `json:"tin,omitempty"`
	Sector               string    `json:"sector,omitempty"`
	AnnualTurnover       string    `json:"annual_turnover,omitempty"`
	SupportPeppol        bool      `json:"support_peppol,omitempty"`
	IsRealtimeReporting  bool      `json:"is_realtime_reporting,omitempty"`
	NotificationChannels string    `json:"notification_channels,omitempty"`
	ERPSystem            string    `json:"erp_system,omitempty"`
	IRNTemplate          string    `json:"irn_template,omitempty"`
	IsActive             bool      `json:"is_active,omitempty"`
}
type PageInfo struct {
	Page            int  `json:"page"`
	Size            int  `json:"size"`
	HasNextPage     bool `json:"hasNextPage"`
	HasPreviousPage bool `json:"hasPreviousPage"`
	TotalCount      int  `json:"totalCount"`
}
type ErrorData struct {
	ID            string `json:"id,omitempty"`
	Handler       string `json:"handler,omitempty"`
	PublicMessage string `json:"public_message,omitempty"`
}
