package dtos

import "einvoice-access-point/pkg/models"

type SubscriptionPlanQueryDto struct {
	IsSandbox string `query:"is_sandbox" validate:"required,oneof=true false"`
}

type CreateSubscriptionPlanDto struct {
	IsSandbox     *bool   `json:"is_sandbox" validate:"required"`
	Name          string  `json:"name" validate:"required"`
	Amount        float64 `json:"amount" validate:"required,gt=0"`
	TotalInvoices int     `json:"total_invoices" validate:"required,gt=0"`
	BillingCycle  int     `json:"billing_cycle" validate:"required,gt=0"`
}

type SubscriptionPlansResponseDto struct {
	BaseResponseDto
	Data []models.SubscriptionPlan `json:"data"`
}

type CreateSubscriptionPlanDataDto struct {
	IsSandbox bool                    `json:"is_sandbox" example:"true"`
	Plan      models.SubscriptionPlan `json:"plan"`
}

type CreateSubscriptionPlanResponseDto struct {
	BaseResponseDto
	Data CreateSubscriptionPlanDataDto `json:"data"`
}

type AggregatorSubscribeRequestDto struct {
	BusinessID string `json:"business_id" validate:"required,uuid"`
	PlanID     string `json:"plan_id" validate:"required"`
}

type AggregatorSubscribeDataDto struct {
	Provider         string `json:"provider" example:"paystack"`
	TransactionID    string `json:"transaction_id" example:"123e4567-e89b-12d3-a456-426614174000"`
	TransactionRef   string `json:"transaction_ref" example:"aggsub_1712345678_abcdef"`
	AuthorizationURL string `json:"authorization_url" example:"https://checkout.paystack.com/..."`
	BusinessID       string `json:"business_id" example:"123e4567-e89b-12d3-a456-426614174000"`
	PlanID           string `json:"plan_id" example:"123e4567-e89b-12d3-a456-426614174000"`
}

type AggregatorSubscribeResponseDto struct {
	BaseResponseDto
	Data AggregatorSubscribeDataDto `json:"data"`
}
