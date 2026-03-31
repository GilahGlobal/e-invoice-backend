package dtos

import (
	"einvoice-access-point/pkg/database"
	"time"
)

type BulkUploadValidationErrorDto struct {
	Error         string `json:"error" example:"invoice VA12239: validation failed: Key: 'UploadInvoiceRequestDto.InvoiceTypeCode' Error:Field validation for 'InvoiceTypeCode' failed on the 'oneof' tag"`
	InvoiceIndex  int    `json:"invoice_index" example:"1"`
	InvoiceNumber string `json:"invoice_number,omitempty" example:"VA12239"`
}

type BulkUploadLogDto struct {
	ID                          string                         `json:"id" example:"019d28e5-bfad-7211-a56f-4e3376a67cf9"`
	FileURL                     string                         `json:"file_url" example:"https://nexar-file-uploads.s3.amazonaws.com/invoices/sample.xlsx"`
	FileKey                     string                         `json:"file_key" example:"invoices/sample.xlsx"`
	BusinessID                  string                         `json:"business_id" example:"ac0d4848-c898-49ce-8fc7-46f529a9354a"`
	Status                      string                         `json:"status" example:"completed"`
	TotalRecords                int                            `json:"total_records" example:"4"`
	ValidRecords                int                            `json:"valid_records" example:"3"`
	SuccessfulInvoices          int                            `json:"successful_invoices" example:"0"`
	PartiallySuccessfulInvoices int                            `json:"partially_successful_invoices" example:"1"`
	UnsuccessfulInvoices        int                            `json:"unsuccessful_invoices" example:"3"`
	ValidationErrorCount        int                            `json:"validation_error_count" example:"1"`
	ValidationErrors            []BulkUploadValidationErrorDto `json:"validation_errors"`
	StartedAt                   *time.Time                     `json:"started_at" example:"2026-03-26T07:47:21.584844+01:00"`
	CompletedAt                 *time.Time                     `json:"completed_at" example:"2026-03-26T07:47:21.591765+01:00"`
	CreatedAt                   time.Time                      `json:"created_at" example:"2026-03-26T07:47:19.286025+01:00"`
	Duration                    int                            `json:"duration" example:"6920833"`
}

type GetBulkUploadLogsResponseDto struct {
	BaseResponseDto
	Data       []BulkUploadLogDto            `json:"data"`
	Pagination []database.PaginationResponse `json:"pagination"`
}
