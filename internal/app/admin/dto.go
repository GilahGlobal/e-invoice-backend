package admin

import (
	"einvoice-access-point/internal/data/database"
	"einvoice-access-point/internal/data/entities"

	"github.com/google/uuid"
)

type AdminLoginRequestDto struct {
	Email     string `json:"email" validate:"required,email" example:"admin@example.com"`
	Password  string `json:"password" validate:"required" example:"password123"`
	IsSandbox bool   `json:"is_sandbox" validate:"required" example:"false"`
}

type AdminRegisterDto struct {
	Name     string    `json:"name" validate:"required" example:"Super Admin"`
	Email    string    `json:"email" validate:"required,email" example:"admin@example.com"`
	Password string    `json:"password" validate:"required,min=8" example:"password123"`
	RoleID   uuid.UUID `json:"role_id" validate:"required" example:"123e4567-e89b-12d3-a456-426614174000" swaggertype:"string"`
}

type AdminResponse struct {
	ID     string `json:"id" example:"123e4567-e89b-12d3-a456-426614174000"`
	Name   string `json:"name" example:"Super Admin"`
	Email  string `json:"email" example:"admin@example.com"`
	RoleID string `json:"role_id" example:"123e4567-e89b-12d3-a456-426614174000"`
	Role   string `json:"role" example:"superadmin"`
}

type AdminLoginResponseDto struct {
	Data        AdminResponse `json:"data"`
	AccessToken string        `json:"access_token" example:"string"`
}

type AdminBusinessResponseDto struct {
	ID                    string `json:"id" example:"e4b7712b-1461-4ae1-aabd-a591ce653b8a"`
	Name                  string `json:"name" example:"Example Name"`
	ServiceID             string `json:"service_id" example:"8817a77d-22d9-4bcc-8b33-dcd1328e31e4"`
	TIN                   string `json:"tin" example:"12345678-0001"`
	Industry              string `json:"industry" example:"string"`
	CreatedAt             string `json:"created_at" example:"2023-10-12T07:20:50.52Z"`
	Email                 string `json:"email" example:"user@example.com"`
	BusinessID            string `json:"business_id" example:"4f7ba55f-1c44-4ac4-989e-1d5c3d948c16"`
	PhoneNumber           string `json:"phone_number" example:"+1234567890"`
	CompanyName           string `json:"company_name" example:"Example Name"`
	BmpUploadSelected     bool   `json:"bmp_upload_selected" example:"true"`
	SubscribedPlan        string `json:"subscribed_plan" example:"string"`
	TotalInvoicesUploaded int64  `json:"total_invoices_uploaded" example:"0"`
	Status                int    `json:"status" example:"1"`
	LastInvoiceUploadedAt string `json:"last_invoice_uploaded_at,omitempty" example:"2023-10-12T07:20:50.52Z"`
}

type AdminBusinessListResponseDto struct {
	entities.Response
	Data       []AdminBusinessResponseDto  `json:"data"`
	Pagination database.PaginationResponse `json:"pagination"`
}

type AdminAggregatorResponseDto struct {
	ID                    string `json:"id" example:"8c596cb2-ac83-489a-bb00-10e0d83c0510"`
	CompanyName           string `json:"company_name" example:"Example Name"`
	Email                 string `json:"email" example:"user@example.com"`
	TIN                   string `json:"tin" example:"12345678-0001"`
	Industry              string `json:"industry" example:"string"`
	SubscribedPlan        string `json:"subscribed_plan" example:"string"`
	CompaniesManaged      int64  `json:"companies_managed" example:"0"`
	TotalInvoicesManaged  int64  `json:"total_invoices_managed" example:"0"`
	LastInvoiceUploadedAt string `json:"last_invoice_uploaded_at,omitempty" example:"2023-10-12T07:20:50.52Z"`
	Status                int    `json:"status" example:"1"`
	CreatedAt             string `json:"created_at" example:"2023-10-12T07:20:50.52Z"`
}

type AdminAggregatorListResponseDto struct {
	entities.Response
	Data       []AdminAggregatorResponseDto `json:"data"`
	Pagination database.PaginationResponse  `json:"pagination"`
}

type AdminTransactionDto struct {
	ID           string  `json:"id" example:"0a0ecb8d-413a-4bf8-8b45-f86980c92e8f"`
	BusinessID   string  `json:"business_id" example:"8e2fde82-5ab0-4f1b-ba5e-6f50f831daa9"`
	AggregatorID string  `json:"aggregator_id" example:"74edb1a0-54f7-41ae-b68b-1904f9f64acf"`
	Reference    string  `json:"reference" example:"string"`
	Provider     string  `json:"provider" example:"99a22e3e-371b-42cc-b77a-55b656af65f0"`
	Status       string  `json:"status" example:"2023-10-12T07:20:50.52Z"`
	Amount       float64 `json:"amount" example:"0.0"`
	Currency     string  `json:"currency" example:"string"`
	PlanID       string  `json:"plan_id" example:"223f3eba-f611-4b8c-94e3-993b0201fa3f"`
	Plan         string  `json:"plan" example:"string"`
	CreatedAt    string  `json:"created_at" example:"2023-10-12T07:20:50.52Z"`
}

type AdminBusinessStatsDto struct {
	TotalBusinesses  int64 `json:"total_businesses" example:"0"`
	TotalAggregators int64 `json:"total_aggregators" example:"0"`
}

type AdminBusinessStatsResponseDto struct {
	entities.Response
	Data AdminBusinessStatsDto `json:"data"`
}

type AdminInvoiceStatsResponseDto struct {
	entities.Response
	Data entities.InvoiceStatsResponseData `json:"data"`
}

type AdminInvoiceListResponseDto struct {
	entities.Response
	Data       []entities.MinimalInvoiceDTO `json:"data"`
	Pagination database.PaginationResponse  `json:"pagination"`
}

type AdminAggregatorInvoiceListResponseDto struct {
	entities.Response
	Data       []entities.Invoice          `json:"data"`
	Pagination database.PaginationResponse `json:"pagination"`
}

type AdminTransactionListResponseDto struct {
	entities.Response
	Data       []AdminTransactionDto       `json:"data"`
	Pagination database.PaginationResponse `json:"pagination"`
}

type AdminOverviewStatsDto struct {
	TotalInvoices          int64 `json:"total_invoices" example:"0"`
	SuccessInvoices        int64 `json:"success_invoices" example:"0"`
	PartialSuccessInvoices int64 `json:"partial_success_invoices" example:"0"`
	FailedInvoices         int64 `json:"failed_invoices" example:"0"`
	TotalCompanies         int64 `json:"total_companies" example:"0"`
	TotalApiCalls          int64 `json:"total_api_calls" example:"0"`
	NewRegistrations       int64 `json:"new_registrations" example:"0"`
}

type AdminDailyInvoiceStatsDto struct {
	Date                   string `json:"date" example:"2023-10-12T07:20:50.52Z"`
	SuccessInvoices        int64  `json:"success_invoices" example:"0"`
	PartialSuccessInvoices int64  `json:"partial_success_invoices" example:"0"`
	FailedInvoices         int64  `json:"failed_invoices" example:"0"`
}

type AdminCreateBusinessDto struct {
	CompanyName string `json:"company_name" validate:"required" example:"Example Name"`
	Email       string `json:"email" validate:"required,email" example:"user@example.com"`
	TIN         string `json:"tin" validate:"required" example:"12345678-0001"`
	Industry    string `json:"industry" example:"string"`
	PhoneNumber string `json:"phone_number" example:"+1234567890"`
}

type AdminCreateAggregatorDto struct {
	CompanyName string `json:"company_name" validate:"required" example:"Example Name"`
	Email       string `json:"email" validate:"required,email" example:"user@example.com"`
	TIN         string `json:"tin" validate:"required" example:"12345678-0001"`
	Industry    string `json:"industry" example:"string"`
	PhoneNumber string `json:"phone_number" example:"+1234567890"`
}

type AdminUpdateBusinessDto struct {
	CompanyName string `json:"company_name" example:"Example Name"`
	TIN         string `json:"tin" example:"12345678-0001"`
	Industry    string `json:"industry" example:"string"`
	PhoneNumber string `json:"phone_number" example:"+1234567890"`
	AccStatus   *int   `json:"acc_status"`
}

type AdminAggregatorInfoResponseDto struct {
	Aggregator AdminAggregatorResponseDto `json:"aggregator"`
	Stats      AggregatorStatsDto         `json:"stats"`
	Companies  []AggregatorCompanyDto     `json:"companies"`
}

type AggregatorStatsDto struct {
	CompaniesManaged   int64 `json:"companies_managed" example:"0"`
	InvoicesUploaded   int64 `json:"invoices_uploaded" example:"0"`
	PendingInvitations int64 `json:"pending_invitations" example:"0"`
}

type AggregatorCompanyDto struct {
	ID               string `json:"id" example:"103e792c-c952-499d-a40c-9f3c237ac293"`
	CompanyName      string `json:"company_name" example:"Example Name"`
	TIN              string `json:"tin" example:"12345678-0001"`
	InvoicesUploaded int64  `json:"invoices_uploaded" example:"0"`
}

type AdminAggregatorInvitationDto struct {
	ID          string `json:"id" example:"76a73e5c-6fe0-4f83-9b95-715147812f65"`
	CompanyName string `json:"company_name" example:"Example Name"`
	Industry    string `json:"industry" example:"string"`
	Status      string `json:"status" example:"2023-10-12T07:20:50.52Z"`
	CreatedAt   string `json:"created_at" example:"2023-10-12T07:20:50.52Z"`
}

type AdminOverviewStatsResponseDto struct {
	entities.Response
	Data AdminOverviewStatsDto `json:"data"`
}

type AdminBusinessDailyStatsResponseDto struct {
	entities.Response
	Data []AdminDailyInvoiceStatsDto `json:"data"`
}

type AdminBusinessAggregatorInfoResponse struct {
	entities.Response
	Data AdminBusinessAggregatorInfoResponseDto `json:"data"`
}

type AdminAggregatorInfoResponse struct {
	entities.Response
	Data AdminAggregatorInfoResponseDto `json:"data"`
}

type AdminAggregatorInvitationsResponse struct {
	entities.Response
	Data []AdminAggregatorInvitationDto `json:"data"`
}

type RoleResponseDto struct {
	ID          uuid.UUID `json:"id" example:"123e4567-e89b-12d3-a456-426614174000" swaggertype:"string"`
	Name        string    `json:"name" example:"superadmin"`
	Description string    `json:"description" example:"Super Administrator with full access"`
}

type RoleListResponseDto struct {
	entities.Response
	Data []RoleResponseDto `json:"data"`
}
