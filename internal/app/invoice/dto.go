package invoice

import (
	"einvoice-access-point/internal/data/database"
	"einvoice-access-point/internal/data/entities"
	"einvoice-access-point/internal/pkg/firs_models"
)

type InvoiceStepMetadata struct {
	Step      string `json:"step" example:"validated_irn"`
	Status    string `json:"status" example:"success"`
	Timestamp string `json:"timestamp" example:"2024-01-01T12:00:00Z"`
}

type InvoiceData struct {
	InvoiceNumber string `json:"invoice_number" example:"INV-1001"`
	IRN           string `json:"irn" example:"123e4567-e89b-12d3-a456-426614174000"`
	QRCode        string `json:"qr_code" example:"iVBORw0KGgoAAAANSUhEUgAAAQAAAAEAAQMAAABmvDolAAAABlBMVEX///8AAABVwtN..."`
	QRCode2       string `json:"qr_code_2" example:"eeleGz7LXrt3gignmXGi9DAeXoVS7GjMR/8WK4f8G76DSP14SA2PSyArr4oaS6ojo0EqCTlp2UBjT2eRpn51..."`
	QRCodeBMP     string `json:"qr_code_bmp" example:"Qk02AAAAAAAAAAAAAAAoAAAAAQAAAAEAAAABAAEAAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAAAP..."`
}

type InvoiceUploadData struct {
	ID            string `json:"id" example:"123e4567-e89b-12d3-a456-426614174000"`
	InvoiceNumber string `json:"invoice_number" example:"INV-1001"`
	IRN           string `json:"irn" example:"123e4567-e89b-12d3-a456-426614174000"`
	QRCode        string `json:"qr_code" example:"iVBORw0KGgoAAAANSUhEUgAAAQAAAAEAAQMAAABmvDolAAAABlBMVEX///8AAABVwtN..."`
	QRCodeBMPURL  string `json:"qr_code_bmp_url" example:"https://res.cloudinary.com/demo/image/upload/v1712345678/invoice-bmp-123.bmp"`
}

type UploadInvoiceResponseDto struct {
	entities.Response
	Data     InvoiceUploadData     `json:"data"`
	Metadata []InvoiceStepMetadata `json:"metadata"`
}

type GetAllInvoicesResponseDto struct {
	entities.Response
	Data       []entities.MinimalInvoiceDTO `json:"data"`
	Pagination database.PaginationResponse  `json:"pagination"`
}

type GetInvoiceDetailsResponseDto struct {
	entities.Response
	Data entities.Invoice `json:"data"`
}

type GetInvoiceStatsResponseDto struct {
	entities.Response
	Data entities.InvoiceStatsResponseData `json:"data"`
}

type InvoiceResponse struct {
	ID               string                              `json:"id" example:"123e4567-e89b-12d3-a456-426614174000"`
	InvoiceNumber    string                              `json:"invoice_number" example:"INV-1001"`
	IRN              string                              `json:"irn" example:"123e4567-e89b-12d3-a456-426614174000"`
	BusinessID       string                              `json:"business_id" example:"business-uuid"`
	Platform         string                              `json:"platform" example:"zoho"`
	PlatformMetadata string                              `json:"platform_metadata"`
	InvoiceData      firs_models.UploadInvoiceRequestDto `json:"invoice_data"`
	CurrentStatus    string                              `json:"current_status" example:"validated_irn"`
	StatusHistory    []InvoiceStepMetadata               `json:"status_history"`
	Timestamp        string                              `json:"timestamp" example:"2024-01-01T12:00:00Z"`
	CreatedAt        string                              `json:"created_at" example:"2024-01-01T12:00:00Z"`
	UpdatedAt        string                              `json:"updated_at" example:"2024-01-02T12:00:00Z"`
}

type GetInvoiceResponseDto struct {
	entities.Response
	Data InvoiceResponse `json:"data"`
}

type BulkUpdateInvoiceResponseDto struct {
	entities.Response
	Data firs_models.BulkUpdateInvoiceResponse `json:"data"`
}
