package subscription

import (
	"einvoice-access-point/internal/data/entities"
	"time"
)

const (
	// ManualPaymentProvider tags transactions recorded by an admin for offline payments.
	ManualPaymentProvider = "manual"
	// manualReferencePrefix namespaces offline payment references so they can never collide with Paystack references.
	manualReferencePrefix = "MAN-"
)

// ManualSubscriptionInput describes an offline payment (e.g. bank transfer) being recorded by an admin.
type ManualSubscriptionInput struct {
	BusinessID       string
	PlanID           string
	PaymentReference string
	AmountPaid       *float64
	PaymentDate      *time.Time
	PaymentMethod    string
	Notes            string
	SendReceipt      bool
	AdminID          string
	AdminEmail       string
}

// ManualSubscriptionResult summarises the subscription state after recording an offline payment.
type ManualSubscriptionResult struct {
	SubscriptionID    string
	TransactionID     string
	Reference         string
	BusinessID        string
	AggregatorID      string
	PlanID            string
	Plan              string
	AmountPaid        float64
	PaymentMethod     string
	TotalInvoices     int
	RemainingInvoices int
	NextBillingDate   time.Time
	Renewed           bool
}

type SubscriptionPlanQueryDto struct {
	IsSandbox string `query:"is_sandbox" validate:"required,oneof=true false"`
}

type CreateSubscriptionPlanDto struct {
	IsSandbox          *bool   `json:"is_sandbox" validate:"required"`
	Name               string  `json:"name" validate:"required" example:"Example Name"`
	Amount             float64 `json:"amount" validate:"required,gt=0" example:"0.0"`
	TotalInvoices      int     `json:"total_invoices" validate:"required,gt=0" example:"0"`
	BillingCycle       int     `json:"billing_cycle" validate:"required,gt=0" example:"0"`
	ReminderDaysBefore []int   `json:"reminder_days_before,omitempty"`
}

type SubscriptionPlansResponseDto struct {
	entities.Response
	Data []entities.SubscriptionPlan `json:"data"`
}

type CreateSubscriptionPlanDataDto struct {
	IsSandbox bool                      `json:"is_sandbox" example:"true"`
	Plan      entities.SubscriptionPlan `json:"plan"`
}

type CreateSubscriptionPlanResponseDto struct {
	entities.Response
	Data CreateSubscriptionPlanDataDto `json:"data"`
}

type SubscribeRequestDto struct {
	BusinessID string `json:"business_id,omitempty" example:"657cf2f1-46e5-4977-a6a6-484a691866d5"`
	PlanID     string `json:"plan_id" validate:"required" example:"f435c4ed-694e-4d4a-8d1e-75be3bb5e43b"`
}

type AggregatorSubscribeRequestDto = SubscribeRequestDto

type SubscribeDataDto struct {
	Provider         string  `json:"provider" example:"paystack"`
	TransactionID    string  `json:"transaction_id" example:"123e4567-e89b-12d3-a456-426614174000"`
	TransactionRef   string  `json:"transaction_ref" example:"sub_1712345678_abcdef"`
	AuthorizationURL string  `json:"authorization_url" example:"https://checkout.paystack.com/..."`
	BusinessID       string  `json:"business_id" example:"123e4567-e89b-12d3-a456-426614174000"`
	AggregatorID     *string `json:"aggregator_id,omitempty" example:"123e4567-e89b-12d3-a456-426614174000"`
	PlanID           string  `json:"plan_id" example:"123e4567-e89b-12d3-a456-426614174000"`
}

type AggregatorSubscribeDataDto = SubscribeDataDto

type SubscribeResponseDto struct {
	entities.Response
	Data SubscribeDataDto `json:"data"`
}

type AggregatorSubscribeResponseDto = SubscribeResponseDto

type BusinessSubscriptionStatusDto struct {
	HasActiveSubscription bool       `json:"has_active_subscription" example:"true"`
	SubscriptionID        string     `json:"subscription_id,omitempty" example:"123e4567-e89b-12d3-a456-426614174000"`
	PlanID                string     `json:"plan_id,omitempty" example:"plan_123"`
	Plan                  string     `json:"plan,omitempty" example:"Growth"`
	TotalInvoices         int        `json:"total_invoices" example:"500"`
	RemainingInvoices     int        `json:"remaining_invoices" example:"450"`
	UsedInvoices          int        `json:"used_invoices" example:"50"`
	NextBillingDate       *time.Time `json:"next_billing_date,omitempty"`
	DaysRemaining         int        `json:"days_remaining" example:"25"`
}

type SubscriptionStatusResponseDto struct {
	entities.Response
	Data BusinessSubscriptionStatusDto `json:"data"`
}

