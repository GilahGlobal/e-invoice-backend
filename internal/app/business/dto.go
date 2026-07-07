package business

import "einvoice-access-point/internal/data/entities"

type UpdateBusinessDto struct {
	Name        *string `json:"name" example:"John Doe" validate:"omitempty,min=2,max=250"`
	Email       *string `json:"email" example:"business@example.com" validate:"omitempty,email"`
	PhoneNumber *string `json:"phone_number" example:"+1234567890" validate:"omitempty,numeric"`
	CompanyName *string `json:"company_name" example:"Acme Inc." validate:"omitempty,max=250"`
	BusinessID  *string `json:"business_id" example:"123e4567-e89b-12d3-a456-426614174000" validate:"omitempty,uuid"`
	ServiceID   *string `json:"service_id" example:"6A2BC898" validate:"omitempty"`
}

type GetBusinessResponseDto struct {
	ID                   string          `json:"id" example:"b2c8f0e7-9b6a-4d1e-bd2a-2d0d6f9f93c7"`
	Email                string          `json:"email" example:"business@example.com"`
	Name                 string          `json:"name" example:"Acme Inc."`
	BusinessID           string          `json:"business_id" example:"BUS-12345"`
	ServiceID            string          `json:"service_id" example:"SRV-98765"`
	PlatformConfigs      PlatformConfigs `json:"platform_configs,omitempty"`
	APIKey               string          `json:"api_key" example:"sk_test_12345"`
	AccStatus            string          `json:"acc_status" example:"active"`
	IRNSigningConfigured bool            `json:"irn_signing_configured" example:"true"`
	TIN                  string          `json:"tin" example:"123456789" validate:"omitempty,numeric"`
	PhoneNumber          string          `json:"phone_number" example:"+1234567890" validate:"omitempty,numeric"`
	CompanyName          string          `json:"company_name" example:"Acme Inc." validate:"omitempty,max=250"`
	CreatedAt            string          `json:"created_at" example:"2026-01-16T12:00:00Z"`
	UpdatedAt            string          `json:"updated_at" example:"2026-01-16T12:30:00Z"`
}

type UploadBusinessIRNSigningKeysDataDto struct {
	FileName             string `json:"file_name" example:"crypto_keys.txt"`
	Environment          string `json:"environment" example:"production"`
	IRNSigningConfigured bool   `json:"irn_signing_configured" example:"true"`
}

type UploadBusinessIRNSigningKeysResponseDto struct {
	entities.Response
	Data UploadBusinessIRNSigningKeysDataDto `json:"data"`
}

type PlatformConfigs map[string]AccountingPlatformConfig

type AccountingPlatformConfig struct {
	OrgID      string `json:"org_id" example:"org-123456789"`
	AuthToken  string `json:"auth_token" example:"auth-token-123456789"`
	HMACSecret string `json:"hmac_secret" example:"hmac-secret-123456789"`
	APIKey     string `json:"api_key" example:"api-key-123456789"`
}
