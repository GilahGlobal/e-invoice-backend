package bulkupload

import (
	"einvoice-access-point/internal/pkg/firs_models"
	"time"
)

type UploadInvoiceRequestDto = firs_models.UploadInvoiceRequestDto
type InvoiceDeliveryPeriod = firs_models.InvoiceDeliveryPeriod
type DocumentReference = firs_models.DocumentReference
type Party = firs_models.Party
type PostalAddress = firs_models.PostalAddress
type PaymentMeans = firs_models.PaymentMeans
type AllowanceCharge = firs_models.AllowanceCharge
type TaxTotal = firs_models.TaxTotal
type TaxSubtotal = firs_models.TaxSubtotal
type TaxCategory = firs_models.TaxCategory
type LegalMonetaryTotal = firs_models.LegalMonetaryTotal
type InvoiceLine = firs_models.InvoiceLine
type Item = firs_models.Item
type Price = firs_models.Price

type InvoiceData struct {
	InvoiceNumber string `json:"invoice_number" example:"INV-1001"`
	IRN           string `json:"irn" example:"123e4567-e89b-12d3-a456-426614174000"`
	QRCode        string `json:"qr_code" example:"iVBORw0KGgoAAAANSUhEUgAAAQAAAAEAAQMAAABmvDolAAAABlBMVEX///8AAABVwtN..."`
	QRCode2       string `json:"qr_code_2" example:"eeleGz7LXrt3gignmXGi9DAeXoVS7GjMR/8WK4f8G76DSP14SA2PSyArr4oaS6ojo0EqCTlp2UBjT2eRpn51..."`
	QRCodeBMP     string `json:"qr_code_bmp" example:"Qk02AAAAAAAAAAAAAAAoAAAAAQAAAAEAAAABAAEAAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAAAP..."`
}

type ValidationResults struct {
	ValidInvoices []UploadInvoiceRequestDto
	Errors        []ValidationError
	ValidCount    int
	ErrorCount    int
}

type ValidationError struct {
	InvoiceIndex  int                      `json:"invoice_index"`
	InvoiceNumber string                   `json:"invoice_number,omitempty"`
	Stage         string                   `json:"stage,omitempty"`
	Invoice       *UploadInvoiceRequestDto `json:"invoice,omitempty"`
	Error         any                      `json:"error"`
}

type ProcessResults struct {
	SuccessCount int
	PartialCount int
	ErrorCount   int
	Errors       []ProcessError
}

type ProcessResult struct {
	Invoice *UploadInvoiceRequestDto
	Error   error
	Status  string
	Posted  bool
}

type ProcessError struct {
	InvoiceNumber string                   `json:"invoice_number"`
	Stage         string                   `json:"stage,omitempty"`
	Invoice       *UploadInvoiceRequestDto `json:"invoice,omitempty"`
	Error         string                   `json:"error"`
}

type InvoiceProcessingError struct {
	InvoiceIndex  int
	InvoiceNumber string
	Stage         string
	Invoice       *UploadInvoiceRequestDto
	Err           error
}

const (
	FailureStageValidation     = "validation"
	FailureStageDuplicateCheck = "duplicate_check"
	FailureStageSubscription   = "subscription_check"
	FailureStageDatabase       = "database"
)

func (e *InvoiceProcessingError) Error() string {
	if e == nil || e.Err == nil {
		return ""
	}
	return e.Err.Error()
}

func newInvoiceProcessingError(invoiceIndex int, stage string, invoice UploadInvoiceRequestDto, err error) error {
	if err == nil {
		return nil
	}

	return &InvoiceProcessingError{
		InvoiceIndex:  invoiceIndex,
		InvoiceNumber: invoice.InvoiceNumber,
		Stage:         stage,
		Invoice:       cloneInvoiceForError(invoice),
		Err:           err,
	}
}

func cloneInvoiceForError(invoice UploadInvoiceRequestDto) *UploadInvoiceRequestDto {
	cloned := invoice
	return &cloned
}

type ProcessingStats struct {
	TotalRows                   int           `json:"total_rows"`
	ValidRows                   int           `json:"valid_rows"`
	InvalidRows                 int           `json:"invalid_rows"`
	SuccessfulInvoices          int           `json:"successful_invoices"`
	PartiallySuccessfulInvoices int           `json:"partially_successful_invoices"`
	UnsuccessfulInvoices        int           `json:"unsuccessful_invoices"`
	TotalErrors                 int           `json:"total_errors"`
	StartTime                   time.Time     `json:"start_time"`
	EndTime                     time.Time     `json:"end_time"`
	Duration                    time.Duration `json:"duration"`
}
