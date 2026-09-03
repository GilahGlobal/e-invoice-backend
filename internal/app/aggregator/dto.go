package aggregator

import (
	bulkUploadPkg "einvoice-access-point/internal/app/bulk_upload"
	invoicePkg "einvoice-access-point/internal/app/invoice"
	"einvoice-access-point/internal/data/database"
	"einvoice-access-point/internal/data/entities"
	"einvoice-access-point/internal/pkg/firs_models"
)

type SendAggregatorInvitationDto struct {
	AggregatorID string `json:"aggregator_id" example:"123e4567-e89b-12d3-a456-426614174000" validate:"required,uuid"`
}

type CreateBusinessDto struct {
	Name        string `json:"name" example:"John Doe" validate:"required,min=2,max=250"`
	Email       string `json:"email" example:"business@example.com" validate:"required,email"`
	Password    string `json:"password" example:"Password123!" validate:"required,min=6"`
	CompanyName string `json:"company_name" example:"Acme Inc." validate:"required,min=2,max=250"`
	PhoneNumber string `json:"phone_number" example:"+1234567890" validate:"required,numeric"`
	TIN         string `json:"tin" example:"123456789" validate:"required,numeric"`
}

type CreateBusinessResponseDto struct {
	Status     string `json:"status" example:"success"`
	StatusCode int    `json:"status_code" example:"201"`
	Message    string `json:"message" example:"Business created successfully"`
}

type RespondToInvitationDto struct {
	InvitationID string `json:"invitation_id" example:"123e4567-e89b-12d3-a456-426614174000" validate:"required,uuid"`
	Accept       bool   `json:"accept" example:"true"`
}

type AggregatorUserResponse struct {
	ID          string `json:"id" example:"123e4567-e89b-12d3-a456-426614174000"`
	Email       string `json:"email" example:"aggregator@example.com"`
	Name        string `json:"name" example:"John Doe"`
	CompanyName string `json:"company_name" example:"Aggregator Corp"`
	IsSandbox   bool   `json:"is_sandbox" example:"true"`
}

type AvailableAggregatorDto struct {
	ID          string `json:"id" example:"123e4567-e89b-12d3-a456-426614174000"`
	Name        string `json:"name" example:"John Doe"`
	Email       string `json:"email" example:"aggregator@example.com"`
	CompanyName string `json:"company_name" example:"Aggregator Corp"`
	PhoneNumber string `json:"phone_number" example:"+2348012345678"`
}

type AggregatorBusinessDetailDto struct {
	ID          string  `json:"id" example:"123e4567-e89b-12d3-a456-426614174000"`
	Name        string  `json:"name" example:"Business Owner"`
	Email       string  `json:"email" example:"business@example.com"`
	CompanyName string  `json:"company_name" example:"Business Corp"`
	TIN         string  `json:"tin" example:"TIN-123456789"`
	PhoneNumber string  `json:"phone_number" example:"+2348012345678"`
	ServiceID   *string `json:"service_id" example:"6A2BC898"`
	BusinessID  *string `json:"business_id" example:"123e4567-e89b-12d3-a456-426614174000"`
	KeysSet     bool    `json:"keys_set" example:"true"`
	AcceptedAt  string  `json:"accepted_at,omitempty" example:"2026-01-01T12:00:00Z"`
}

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

type AggregatorBusinessFullDetailDto struct {
	ID          string  `json:"id" example:"123e4567-e89b-12d3-a456-426614174000"`
	Name        string  `json:"name" example:"Business Owner"`
	Email       string  `json:"email" example:"business@example.com"`
	CompanyName string  `json:"company_name" example:"Business Corp"`
	TIN         string  `json:"tin" example:"TIN-123456789"`
	PhoneNumber string  `json:"phone_number" example:"+2348012345678"`
	ServiceID   *string `json:"service_id" example:"6A2BC898"`
	BusinessID  *string `json:"business_id" example:"123e4567-e89b-12d3-a456-426614174000"`
	KeysSet     bool    `json:"keys_set" example:"true"`
	AcceptedAt  string  `json:"accepted_at,omitempty" example:"2026-01-01T12:00:00Z"`

	Subscription *BusinessSubscriptionInfoDto `json:"subscription"`

	TotalInvoicesUploaded int64 `json:"total_invoices_uploaded" example:"120"`
	TotalBulkUploads      int64 `json:"total_bulk_uploads" example:"5"`
}

type AggregatorDashboardDto struct {
	TotalBusinesses    int64 `json:"total_businesses" example:"10"`
	PendingInvitations int64 `json:"pending_invitations" example:"3"`
	TotalInvoices      int64 `json:"total_invoices" example:"500"`
	TotalBulkUploads   int64 `json:"total_bulk_uploads" example:"25"`
}

type AggregatorInvitationDto struct {
	ID            string `json:"id" example:"123e4567-e89b-12d3-a456-426614174000"`
	BusinessID    string `json:"business_id" example:"123e4567-e89b-12d3-a456-426614174000"`
	BusinessName  string `json:"business_name" example:"Business Corp"`
	BusinessEmail string `json:"business_email" example:"business@example.com"`
	Status        string `json:"status" example:"pending"`
	CreatedAt     string `json:"created_at" example:"2026-01-01T12:00:00Z"`
}

type BusinessInvitationDto struct {
	ID              string `json:"id" example:"123e4567-e89b-12d3-a456-426614174000"`
	AggregatorID    string `json:"aggregator_id" example:"123e4567-e89b-12d3-a456-426614174000"`
	AggregatorName  string `json:"aggregator_name" example:"Aggregator Corp"`
	AggregatorEmail string `json:"aggregator_email" example:"aggregator@example.com"`
	Status          string `json:"status" example:"pending"`
	CreatedAt       string `json:"created_at" example:"2026-01-01T12:00:00Z"`
}

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
	entities.Response
	Data       []AggregatorBusinessDetailDto `json:"data"`
	Pagination database.PaginationResponse   `json:"pagination"`
}

type AggregatorBusinessFullDetailResponseDto struct {
	entities.Response
	Data AggregatorBusinessFullDetailDto `json:"data"`
}

type AggregatorInvitationListResponseDto struct {
	entities.Response
	Data []AggregatorInvitationDto `json:"data"`
}

type AvailableAggregatorsResponseDto struct {
	entities.Response
	Data       []AvailableAggregatorDto    `json:"data"`
	Pagination database.PaginationResponse `json:"pagination"`
}

type AggregatorDashboardResponseDto struct {
	entities.Response
	Data AggregatorDashboardDto `json:"data"`
}

type AggregatorInvoiceListResponseDto struct {
	entities.Response
	Data       []entities.MinimalInvoiceDTO `json:"data"`
	Pagination database.PaginationResponse  `json:"pagination"`
}

type AggregatorBulkUploadListResponseDto struct {
	entities.Response
	Data       []entities.BulkUpload       `json:"data"`
	Pagination database.PaginationResponse `json:"pagination"`
}

type AggregatorActivityLogListResponseDto struct {
	entities.Response
	Data       []AggregatorActivityLogDto  `json:"data"`
	Pagination database.PaginationResponse `json:"pagination"`
}

type BusinessInvitationListResponseDto struct {
	entities.Response
	Data []BusinessInvitationDto `json:"data"`
}

type TransactionDto struct {
	ID              string  `json:"id" example:"123e4567-e89b-12d3-a456-426614174000"`
	BusinessID      string  `json:"business_id" example:"123e4567-e89b-12d3-a456-426614174000"`
	BusinessName    string  `json:"business_name" example:"Business Corp"`
	AggregatorID    string  `json:"aggregator_id" example:"123e4567-e89b-12d3-a456-426614174000"`
	Reference       string  `json:"reference" example:"txn_123456789"`
	Provider        string  `json:"provider" example:"paystack"`
	Status          string  `json:"status" example:"success"`
	Amount          float64 `json:"amount" example:"5000"`
	Currency        string  `json:"currency" example:"NGN"`
	PlanID          string  `json:"plan_id" example:"plan_123"`
	Plan            string  `json:"plan" example:"Starter"`
	GatewayResponse string  `json:"gateway_response" example:"Approved"`
	CreatedAt       string  `json:"created_at" example:"2026-01-01T12:00:00Z"`
	UpdatedAt       string  `json:"updated_at" example:"2026-01-01T12:00:00Z"`
}

type AggregatorGetInvoiceResponseDto struct {
	Status     string           `json:"status" example:"success"`
	StatusCode int              `json:"status_code" example:"200"`
	Message    string           `json:"message" example:"Invoice details fetched successfully"`
	Data       entities.Invoice `json:"data"`
}

type AggregatorTransactionListResponseDto struct {
	entities.Response
	Data       []TransactionDto            `json:"data"`
	Pagination database.PaginationResponse `json:"pagination"`
}

type AggregatorUpdateBusinessSetupDto struct {
	ServiceID      *string `json:"service_id" example:"6A2BC898" validate:"omitempty"`
	BusinessID     *string `json:"business_id" example:"123e4567-e89b-12d3-a456-426614174000" validate:"omitempty,uuid"`
	IRNPublicKey   *string `json:"irn_public_key" example:"public-key-content" validate:"omitempty"`
	IRNCertificate *string `json:"irn_certificate" example:"certificate-content" validate:"omitempty"`
}

type SendAggregatorInvitationByEmailDto struct {
	Email string `json:"email" example:"aggregator@example.com" validate:"required,email"`
}

type UploadInvoiceRequestDto = firs_models.UploadInvoiceRequestDto
type InvoiceData = invoicePkg.InvoiceData
type InvoiceUploadData = invoicePkg.InvoiceUploadData

type AggregatorInvoiceUploadResponseDto struct {
	entities.Response
	Data     InvoiceUploadData                `json:"data"`
	Metadata []invoicePkg.InvoiceStepMetadata `json:"metadata"`
}

type GetBulkUploadLogsResponseDto = bulkUploadPkg.GetBulkUploadLogsResponseDto
type GetBulkUploadFailedInvoicesResponseDto = bulkUploadPkg.GetBulkUploadFailedInvoicesResponseDto
type GetInvoiceStatsResponseDto = invoicePkg.GetInvoiceStatsResponseDto

type BulkUploadFailedInvoicesDto = bulkUploadPkg.BulkUploadFailedInvoicesDto
