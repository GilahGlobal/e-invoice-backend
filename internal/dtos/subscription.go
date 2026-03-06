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
