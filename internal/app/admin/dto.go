package admin

import (
	"einvoice-access-point/internal/data/database"
	"einvoice-access-point/internal/data/entities"
)

type AdminLoginRequestDto struct {
	Email     string `json:"email" validate:"required,email"`
	Password  string `json:"password" validate:"required"`
	IsSandbox bool   `json:"is_sandbox" validate:"required"`
}

type AdminRegisterDto struct {
	Name     string             `json:"name" validate:"required"`
	Email    string             `json:"email" validate:"required,email"`
	Password string             `json:"password" validate:"required,min=8"`
	Role     entities.AdminRole `json:"role" validate:"required,oneof=superadmin admin"`
}

type AdminResponse struct {
	ID    string             `json:"id"`
	Name  string             `json:"name"`
	Email string             `json:"email"`
	Role  entities.AdminRole `json:"role"`
}

type AdminLoginResponseDto struct {
	Data        AdminResponse `json:"data"`
	AccessToken string        `json:"access_token"`
}

type AdminBusinessResponseDto struct {
	ID                      string    `json:"id"`
	Name                    string    `json:"name"`
	ServiceID               string    `json:"service_id"`
	TIN                     string    `json:"tin"`
	Industry                string    `json:"industry"`
	CreatedAt               string    `json:"created_at"`
	Email                   string    `json:"email"`
	BusinessID              string    `json:"business_id"`
	PhoneNumber             string    `json:"phone_number"`
	CompanyName             string    `json:"company_name"`
	BmpUploadSelected       bool      `json:"bmp_upload_selected"`
	SubscribedPlan          string    `json:"subscribed_plan"`
	TotalInvoicesUploaded   int64     `json:"total_invoices_uploaded"`
	Status                  int       `json:"status"`
	LastInvoiceUploadedAt   string    `json:"last_invoice_uploaded_at,omitempty"`
}

type AdminBusinessListResponseDto struct {
	entities.Response
	Data       []AdminBusinessResponseDto `json:"data"`
	Pagination database.PaginationResponse `json:"pagination"`
}

type AdminAggregatorResponseDto struct {
	ID                    string `json:"id"`
	CompanyName           string `json:"company_name"`
	Email                 string `json:"email"`
	TIN                   string `json:"tin"`
	Industry              string `json:"industry"`
	SubscribedPlan        string `json:"subscribed_plan"`
	CompaniesManaged      int64  `json:"companies_managed"`
	TotalInvoicesManaged  int64  `json:"total_invoices_managed"`
	LastInvoiceUploadedAt string `json:"last_invoice_uploaded_at,omitempty"`
	Status                int    `json:"status"`
	CreatedAt             string `json:"created_at"`
}

type AdminAggregatorListResponseDto struct {
	entities.Response
	Data       []AdminAggregatorResponseDto `json:"data"`
	Pagination database.PaginationResponse  `json:"pagination"`
}

type AdminTransactionDto struct {
	ID              string  `json:"id"`
	BusinessID      string  `json:"business_id"`
	AggregatorID    string  `json:"aggregator_id"`
	Reference       string  `json:"reference"`
	Provider        string  `json:"provider"`
	Status          string  `json:"status"`
	Amount          float64 `json:"amount"`
	Currency        string  `json:"currency"`
	PlanID          string  `json:"plan_id"`
	Plan            string  `json:"plan"`
	CreatedAt       string  `json:"created_at"`
}

type AdminBusinessStatsDto struct {
	TotalBusinesses  int64 `json:"total_businesses"`
	TotalAggregators int64 `json:"total_aggregators"`
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
	TotalInvoices          int64 `json:"total_invoices"`
	SuccessInvoices        int64 `json:"success_invoices"`
	PartialSuccessInvoices int64 `json:"partial_success_invoices"`
	FailedInvoices         int64 `json:"failed_invoices"`
	TotalCompanies         int64 `json:"total_companies"`
	TotalApiCalls          int64 `json:"total_api_calls"`
	NewRegistrations       int64 `json:"new_registrations"`
}

type AdminDailyInvoiceStatsDto struct {
	Date                   string `json:"date"`
	SuccessInvoices        int64  `json:"success_invoices"`
	PartialSuccessInvoices int64  `json:"partial_success_invoices"`
	FailedInvoices         int64  `json:"failed_invoices"`
}

type AdminCreateBusinessDto struct {
	CompanyName string `json:"company_name" validate:"required"`
	Email       string `json:"email" validate:"required,email"`
	TIN         string `json:"tin" validate:"required"`
	Industry    string `json:"industry"`
	PhoneNumber string `json:"phone_number"`
}

type AdminCreateAggregatorDto struct {
	CompanyName string `json:"company_name" validate:"required"`
	Email       string `json:"email" validate:"required,email"`
	TIN         string `json:"tin" validate:"required"`
	Industry    string `json:"industry"`
	PhoneNumber string `json:"phone_number"`
}

type AdminUpdateBusinessDto struct {
	CompanyName string `json:"company_name"`
	TIN         string `json:"tin"`
	Industry    string `json:"industry"`
	PhoneNumber string `json:"phone_number"`
	AccStatus   *int   `json:"acc_status"`
}

type AdminAggregatorInfoResponseDto struct {
	Aggregator AdminAggregatorResponseDto `json:"aggregator"`
	Stats      AggregatorStatsDto         `json:"stats"`
	Companies  []AggregatorCompanyDto     `json:"companies"`
}

type AggregatorStatsDto struct {
	CompaniesManaged   int64 `json:"companies_managed"`
	InvoicesUploaded   int64 `json:"invoices_uploaded"`
	PendingInvitations int64 `json:"pending_invitations"`
}

type AggregatorCompanyDto struct {
	ID               string `json:"id"`
	CompanyName      string `json:"company_name"`
	TIN              string `json:"tin"`
	InvoicesUploaded int64  `json:"invoices_uploaded"`
}

type AdminAggregatorInvitationDto struct {
	ID          string `json:"id"`
	CompanyName string `json:"company_name"`
	Industry    string `json:"industry"`
	Status      string `json:"status"`
	CreatedAt   string `json:"created_at"`
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
