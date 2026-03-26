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
	InvoiceIndex  int    `json:"invoice_index"`
	InvoiceNumber string `json:"invoice_number,omitempty"`
	Error         string `json:"error"`
}

type ProcessResults struct {
	SuccessCount int
	ErrorCount   int
	Errors       []ProcessError
}

type ProcessResult struct {
	Invoice *dtos.UploadInvoiceRequestDto
	Error   error
	Posted  bool
}

type ProcessError struct {
	InvoiceNumber string `json:"invoice_number"`
	Error         string `json:"error"`
}

type ProcessingStats struct {
	TotalRows            int           `json:"total_rows"`
	ValidRows            int           `json:"valid_rows"`
	InvalidRows          int           `json:"invalid_rows"`
	SuccessfulInvoices   int           `json:"successful_invoices"`
	UnsuccessfulInvoices int           `json:"unsuccessful_invoices"`
	TotalErrors          int           `json:"total_errors"`
	StartTime            time.Time     `json:"start_time"`
	EndTime              time.Time     `json:"end_time"`
	Duration             time.Duration `json:"duration"`
}
