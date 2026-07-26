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
	ID                string    `json:"id"`
	Name              string    `json:"name"`
	ServiceID         string    `json:"service_id"`
	TIN               string    `json:"tin"`
	CreatedAt         string    `json:"created_at"`
	Email             string    `json:"email"`
	BusinessID        string    `json:"business_id"`
	PhoneNumber       string    `json:"phone_number"`
	CompanyName       string    `json:"company_name"`
	BmpUploadSelected bool      `json:"bmp_upload_selected"`
}

type AdminBusinessListResponseDto struct {
	entities.Response
	Data       []AdminBusinessResponseDto `json:"data"`
	Pagination database.PaginationResponse `json:"pagination"`
}

type AdminAggregatorListResponseDto struct {
	entities.Response
	Data       []AdminBusinessResponseDto `json:"data"`
	Pagination database.PaginationResponse `json:"pagination"`
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
