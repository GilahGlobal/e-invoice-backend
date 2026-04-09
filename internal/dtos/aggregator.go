package dtos

import "einvoice-access-point/pkg/database"

// --- Registration ---
type AggregatorRegisterDto struct {
	Name        string `json:"name" example:"John Doe" validate:"required,min=2,max=250"`
	Email       string `json:"email" example:"aggregator@example.com" validate:"required,email"`
	Password    string `json:"password" example:"password123" validate:"required,min=6"`
	CompanyName string `json:"company_name" example:"Aggregator Corp" validate:"required"`
	PhoneNumber string `json:"phone_number" example:"+2348012345678" validate:"required"`
}

// --- Login ---
type AggregatorLoginDto struct {
	Email     string `json:"email" example:"aggregator@example.com" validate:"required,email"`
	Password  string `json:"password" example:"password123" validate:"required"`
	IsSandbox bool   `json:"is_sandbox" default:"true"`
}

// --- Email Verification ---
type AggregatorVerifyEmailDto struct {
	Email string `json:"email" example:"aggregator@example.com" validate:"required,email"`
	OTP   string `json:"otp" example:"123456" validate:"required,numeric"`
}

type AggregatorResendOtpDto struct {
	Email string `json:"email" example:"aggregator@example.com" validate:"required,email"`
}

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
	ID          string `json:"id" example:"123e4567-e89b-12d3-a456-426614174000"`
	Name        string `json:"name" example:"Business Owner"`
	Email       string `json:"email" example:"business@example.com"`
	CompanyName string `json:"company_name" example:"Business Corp"`
	TIN         string `json:"tin" example:"TIN-123456789"`
	PhoneNumber string `json:"phone_number" example:"+2348012345678"`
	ServiceID   string `json:"service_id" example:"6A2BC898"`
	AcceptedAt  string `json:"accepted_at,omitempty" example:"2026-01-01T12:00:00Z"`
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
	ID           string `json:"id" example:"123e4567-e89b-12d3-a456-426614174000"`
	BusinessID   string `json:"business_id" example:"123e4567-e89b-12d3-a456-426614174000"`
	BusinessName string `json:"business_name" example:"Business Corp"`
	BusinessEmail string `json:"business_email" example:"business@example.com"`
	Status       string `json:"status" example:"pending"`
	CreatedAt    string `json:"created_at" example:"2026-01-01T12:00:00Z"`
}

// --- Business Invitation List Item ---
type BusinessInvitationDto struct {
	ID             string `json:"id" example:"123e4567-e89b-12d3-a456-426614174000"`
	AggregatorID   string `json:"aggregator_id" example:"123e4567-e89b-12d3-a456-426614174000"`
	AggregatorName string `json:"aggregator_name" example:"Aggregator Corp"`
	AggregatorEmail string `json:"aggregator_email" example:"aggregator@example.com"`
	Status         string `json:"status" example:"pending"`
	CreatedAt      string `json:"created_at" example:"2026-01-01T12:00:00Z"`
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

// --- Response DTOs ---
type AggregatorLoginResponseDto struct {
	BaseResponseDto
	Data        AggregatorUserResponse `json:"data"`
	AccessToken string                 `json:"access_token"`
}

type AggregatorBusinessListResponseDto struct {
	BaseResponseDto
	Data       []AggregatorBusinessDetailDto `json:"data"`
	Pagination database.PaginationResponse   `json:"pagination"`
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
