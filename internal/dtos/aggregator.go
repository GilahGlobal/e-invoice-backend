package dtos

import (
	"einvoice-access-point/pkg/database"
	"einvoice-access-point/pkg/models"
)

// --- Invitation (Business sends to Aggregator) ---
type SendAggregatorInvitationDto struct {
	AggregatorID string `json:"aggregator_id" example:"123e4567-e89b-12d3-a456-426614174000" validate:"required,uuid"`
}

type RespondToInvitationDto struct {
	InvitationID string `json:"invitation_id" example:"123e4567-e89b-12d3-a456-426614174000" validate:"required,uuid"`
	Accept       bool   `json:"accept" example:"true"`
}

// --- Aggregator User Response ---
type AggregatorUserResponse struct {
	ID          string `json:"id" example:"123e4567-e89b-12d3-a456-426614174000"`
	Email       string `json:"email" example:"aggregator@example.com"`
	Name        string `json:"name" example:"John Doe"`
	CompanyName string `json:"company_name" example:"Aggregator Corp"`
	IsSandbox   bool   `json:"is_sandbox" example:"true"`
}

// --- Available Aggregators (for business to browse) ---
type AvailableAggregatorDto struct {
	ID          string `json:"id" example:"123e4567-e89b-12d3-a456-426614174000"`
	Name        string `json:"name" example:"John Doe"`
	Email       string `json:"email" example:"aggregator@example.com"`
	CompanyName string `json:"company_name" example:"Aggregator Corp"`
	PhoneNumber string `json:"phone_number" example:"+2348012345678"`
}

// --- Aggregator Business List ---
type AggregatorBusinessDetailDto struct {
	ID          string  `json:"id" example:"123e4567-e89b-12d3-a456-426614174000"`
	Name        string  `json:"name" example:"Business Owner"`
	Email       string  `json:"email" example:"business@example.com"`
	CompanyName string  `json:"company_name" example:"Business Corp"`
	TIN         string  `json:"tin" example:"TIN-123456789"`
	PhoneNumber string  `json:"phone_number" example:"+2348012345678"`
	ServiceID   *string `json:"service_id" example:"6A2BC898"`
	AcceptedAt  string  `json:"accepted_at,omitempty" example:"2026-01-01T12:00:00Z"`
}

// --- Subscription Info for a business under an aggregator ---
type BusinessSubscriptionInfoDto struct {
	IsActive          bool    `json:"is_active" example:"true"`
	PlanID            string  `json:"plan_id" example:"123e4567-e89b-12d3-a456-426614174000"`
	PlanName          string  `json:"plan_name" example:"Starter"`
	PlanAmount        float64 `json:"plan_amount" example:"5000"`
	BillingCycleDays  int     `json:"billing_cycle_days" example:"30"`
	TotalInvoices     int     `json:"total_invoices" example:"500"`
	UsedInvoices      int     `json:"used_invoices" example:"120"`
	RemainingInvoices int     `json:"remaining_invoices" example:"380"`
	NextBillingDate   string  `json:"next_billing_date" example:"2026-05-01T00:00:00Z"`
}

// --- Full Business Detail (with subscription + stats) ---
type AggregatorBusinessFullDetailDto struct {
	ID          string  `json:"id" example:"123e4567-e89b-12d3-a456-426614174000"`
	Name        string  `json:"name" example:"Business Owner"`
	Email       string  `json:"email" example:"business@example.com"`
	CompanyName string  `json:"company_name" example:"Business Corp"`
	TIN         string  `json:"tin" example:"TIN-123456789"`
	PhoneNumber string  `json:"phone_number" example:"+2348012345678"`
	ServiceID   *string `json:"service_id" example:"6A2BC898"`
	AcceptedAt  string  `json:"accepted_at,omitempty" example:"2026-01-01T12:00:00Z"`

	// Subscription
	Subscription *BusinessSubscriptionInfoDto `json:"subscription"`

	// Stats
	TotalInvoicesUploaded int64 `json:"total_invoices_uploaded" example:"120"`
	TotalBulkUploads      int64 `json:"total_bulk_uploads" example:"5"`
}

// --- Dashboard ---
type AggregatorDashboardDto struct {
	TotalBusinesses    int64 `json:"total_businesses" example:"10"`
	PendingInvitations int64 `json:"pending_invitations" example:"3"`
	TotalInvoices      int64 `json:"total_invoices" example:"500"`
	TotalBulkUploads   int64 `json:"total_bulk_uploads" example:"25"`
}

// --- Invitation List Item ---
type AggregatorInvitationDto struct {
	ID            string `json:"id" example:"123e4567-e89b-12d3-a456-426614174000"`
	BusinessID    string `json:"business_id" example:"123e4567-e89b-12d3-a456-426614174000"`
	BusinessName  string `json:"business_name" example:"Business Corp"`
	BusinessEmail string `json:"business_email" example:"business@example.com"`
	Status        string `json:"status" example:"pending"`
	CreatedAt     string `json:"created_at" example:"2026-01-01T12:00:00Z"`
}

// --- Business Invitation List Item ---
type BusinessInvitationDto struct {
	ID              string `json:"id" example:"123e4567-e89b-12d3-a456-426614174000"`
	AggregatorID    string `json:"aggregator_id" example:"123e4567-e89b-12d3-a456-426614174000"`
	AggregatorName  string `json:"aggregator_name" example:"Aggregator Corp"`
	AggregatorEmail string `json:"aggregator_email" example:"aggregator@example.com"`
	Status          string `json:"status" example:"pending"`
	CreatedAt       string `json:"created_at" example:"2026-01-01T12:00:00Z"`
}

// --- Activity Log ---
type AggregatorActivityLogDto struct {
	ID           string `json:"id" example:"123e4567-e89b-12d3-a456-426614174000"`
	AggregatorID string `json:"aggregator_id" example:"123e4567-e89b-12d3-a456-426614174000"`
	BusinessID   string `json:"business_id" example:"123e4567-e89b-12d3-a456-426614174000"`
	BusinessName string `json:"business_name,omitempty" example:"Business Corp"`
	Action       string `json:"action" example:"single_invoice_upload"`
	Details      string `json:"details" example:"Uploaded invoice INV-001"`
	CreatedAt    string `json:"created_at" example:"2026-01-01T12:00:00Z"`
}

type AggregatorBusinessListResponseDto struct {
	BaseResponseDto
	Data       []AggregatorBusinessDetailDto `json:"data"`
	Pagination database.PaginationResponse   `json:"pagination"`
}

type AggregatorBusinessFullDetailResponseDto struct {
	BaseResponseDto
	Data AggregatorBusinessFullDetailDto `json:"data"`
}

type AggregatorInvitationListResponseDto struct {
	BaseResponseDto
	Data []AggregatorInvitationDto `json:"data"`
}

type AvailableAggregatorsResponseDto struct {
	BaseResponseDto
	Data       []AvailableAggregatorDto    `json:"data"`
	Pagination database.PaginationResponse `json:"pagination"`
}

type AggregatorDashboardResponseDto struct {
	BaseResponseDto
	Data AggregatorDashboardDto `json:"data"`
}

type AggregatorInvoiceListResponseDto struct {
	BaseResponseDto
	Data       []models.MinimalInvoiceDTO  `json:"data"`
	Pagination database.PaginationResponse `json:"pagination"`
}

type AggregatorBulkUploadListResponseDto struct {
	BaseResponseDto
	Data       []models.BulkUpload         `json:"data"`
	Pagination database.PaginationResponse `json:"pagination"`
}

type AggregatorActivityLogListResponseDto struct {
	BaseResponseDto
	Data       []AggregatorActivityLogDto  `json:"data"`
	Pagination database.PaginationResponse `json:"pagination"`
}

type BusinessInvitationListResponseDto struct {
	BaseResponseDto
	Data []BusinessInvitationDto `json:"data"`
}

type TransactionDto struct {
	ID                string  `json:"id" example:"123e4567-e89b-12d3-a456-426614174000"`
	BusinessID        string  `json:"business_id" example:"123e4567-e89b-12d3-a456-426614174000"`
	BusinessName      string  `json:"business_name" example:"Business Corp"`
	AggregatorID      string  `json:"aggregator_id" example:"123e4567-e89b-12d3-a456-426614174000"`
	Reference         string  `json:"reference" example:"txn_123456789"`
	Provider          string  `json:"provider" example:"paystack"`
	ProviderReference string  `json:"provider_reference" example:"ref_123456789"`
	Status            string  `json:"status" example:"success"`
	Amount            float64 `json:"amount" example:"5000"`
	Currency          string  `json:"currency" example:"NGN"`
	PlanID            string  `json:"plan_id" example:"plan_123"`
	Plan              string  `json:"plan" example:"Starter"`
	GatewayResponse   string  `json:"gateway_response" example:"Approved"`
	CreatedAt         string  `json:"created_at" example:"2026-01-01T12:00:00Z"`
	UpdatedAt         string  `json:"updated_at" example:"2026-01-01T12:00:00Z"`
}

type AggregatorTransactionListResponseDto struct {
	BaseResponseDto
	Data       []TransactionDto            `json:"data"`
	Pagination database.PaginationResponse `json:"pagination"`
}
