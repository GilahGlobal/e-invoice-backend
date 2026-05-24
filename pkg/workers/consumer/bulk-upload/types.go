package bulkupload

import (
	"einvoice-access-point/internal/dtos"
	"time"
)

type ValidationResults struct {
	ValidInvoices []dtos.UploadInvoiceRequestDto
	Errors        []ValidationError
	ValidCount    int
	ErrorCount    int
}

type ValidationError struct {
	InvoiceIndex  int                           `json:"invoice_index"`
	InvoiceNumber string                        `json:"invoice_number,omitempty"`
	Stage         string                        `json:"stage,omitempty"`
	Invoice       *dtos.UploadInvoiceRequestDto `json:"invoice,omitempty"`
	Error         any                           `json:"error"`
}

type ProcessResults struct {
	SuccessCount int
	PartialCount int
	ErrorCount   int
	Errors       []ProcessError
}

type ProcessResult struct {
	Invoice *dtos.UploadInvoiceRequestDto
	Error   error
	Status  string
	Posted  bool
}

type ProcessError struct {
	InvoiceNumber string                        `json:"invoice_number"`
	Stage         string                        `json:"stage,omitempty"`
	Invoice       *dtos.UploadInvoiceRequestDto `json:"invoice,omitempty"`
	Error         string                        `json:"error"`
}

type InvoiceProcessingError struct {
	InvoiceIndex  int
	InvoiceNumber string
	Stage         string
	Invoice       *dtos.UploadInvoiceRequestDto
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

func newInvoiceProcessingError(invoiceIndex int, stage string, invoice dtos.UploadInvoiceRequestDto, err error) error {
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

func cloneInvoiceForError(invoice dtos.UploadInvoiceRequestDto) *dtos.UploadInvoiceRequestDto {
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
