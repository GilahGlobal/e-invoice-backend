package dtos

import (
	"time"

	"einvoice-access-point/pkg/models"
)

type PluginBusinessCheckQueryDto struct {
	Email     string `query:"email" validate:"required,email"`
	IsSandbox string `query:"is_sandbox" validate:"required,oneof=true false"`
}

type PluginPlansQueryDto struct {
	IsSandbox string `query:"is_sandbox" validate:"required,oneof=true false"`
}

type PluginSubscribeRequestDto struct {
	Email     string `json:"email" validate:"required,email"`
	PlanID    string `json:"plan_id" validate:"required"`
	IsSandbox bool   `json:"is_sandbox"`
}

type PluginBusinessSubscriptionDetailsDto struct {
	Plan              string    `json:"plan" example:"basic"`
	TotalInvoices     int       `json:"total_invoices" example:"100"`
	RemainingInvoices int       `json:"remaining_invoices" example:"80"`
	UsedInvoices      int       `json:"used_invoices" example:"20"`
	NextBillingDate   time.Time `json:"next_billing_date"`
}

type PluginBusinessCheckDataDto struct {
	Exists             bool                                  `json:"exists" example:"true"`
	Email              string                                `json:"email" example:"john.doe@example.com"`
	BusinessID         string                                `json:"business_id,omitempty" example:"123e4567-e89b-12d3-a456-426614174000"`
	ActiveSubscription bool                                  `json:"active_subscription" example:"true"`
	Subscription       *PluginBusinessSubscriptionDetailsDto `json:"subscription,omitempty"`
}

type PluginBusinessCheckResponseDto struct {
	BaseResponseDto
	Data PluginBusinessCheckDataDto `json:"data"`
}

type PluginPlansResponseDto struct {
	BaseResponseDto
	Data []models.SubscriptionPlan `json:"data"`
}

type PluginSubscribeDataDto struct {
	Provider         string `json:"provider" example:"paystack"`
	TransactionID    string `json:"transaction_id" example:"123e4567-e89b-12d3-a456-426614174000"`
	TransactionRef   string `json:"transaction_ref" example:"txn_1736272012_abcd1234ef"`
	AuthorizationURL string `json:"authorization_url" example:"https://checkout.paystack.com/3ni8kdavz62431k"`
}

type PluginSubscribeResponseDto struct {
	BaseResponseDto
	Data PluginSubscribeDataDto `json:"data"`
}

type PluginWebhookDataDto struct {
	Event             string `json:"event" example:"charge.success"`
	Reference         string `json:"reference" example:"txn_1736272012_abcd1234ef"`
	TransactionStatus string `json:"transaction_status" example:"success"`
	Database          string `json:"database,omitempty" example:"sandbox"`
}

type PluginWebhookResponseDto struct {
	BaseResponseDto
	Data PluginWebhookDataDto `json:"data"`
}

type PluginRegisteredBusinessDto struct {
	ID         string  `json:"id" example:"123e4567-e89b-12d3-a456-426614174000"`
	Email      string  `json:"email" example:"john.doe@example.com"`
	Name       string  `json:"name" example:"John Doe"`
	BusinessID *string `json:"business_id" example:"123e4567-e89b-12d3-a456-426614174000"`
	ServiceID  string  `json:"service_id" example:"6A2BC898"`
	Tin        string  `json:"tin" example:"TIN-123456789"`
	IsSandbox  bool    `json:"is_sandbox" example:"true"`
}

type PluginRegisterDataDto struct {
	Sandbox    PluginRegisteredBusinessDto `json:"sandbox"`
	Production PluginRegisteredBusinessDto `json:"production"`
}

type PluginRegisterResponseDto struct {
	BaseResponseDto
	Data PluginRegisterDataDto `json:"data"`
}
