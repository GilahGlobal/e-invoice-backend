package firs_models

type IRNValidationRequest struct {
	InvoiceReference string `json:"invoice_reference" validate:"required"`
	BusinessID       string `json:"business_id" validate:"required"`
	IRN              string `json:"irn" validate:"required"`
}

type IRNValidationResponse struct {
	IRN       string `json:"IRN"`
	Status    string `json:"status"`
	Timestamp string `json:"timestamp"`
}

type IRNSigningData struct {
	IRN         string `json:"irn"`
	Certificate string `json:"certificate"`
}

type IRNSigningResponse struct {
	EncryptedIRN   string `json:"encrypted_irn"`
	QrCodeImage    string `json:"qr_code_image"`
	QrCodeImageBMP string `json:"qr_code_image_bmp"`
}

type IRNSigningRequestData struct {
	IRN string `json:"irn"`
}

type GenerateIRNRequestData struct {
	InvoiceNumber string `json:"invoice_number" validate:"required"`
}

type VerifyTinData struct {
	TIN string `json:"tin" validate:"required"`
}

type UpdateInvoice struct {
	PaymentStatus string  `json:"payment_status" validate:"required,oneof=PENDING PAID REJECTED PARTIAL"`
	Reference     *string `json:"reference,omitempty"`
}

type FirsWebhookPayload struct {
	IRN     string `json:"irn" validate:"required"`
	Message string `json:"message" validate:"required"`
}

type BulkUpdateInvoiceItem struct {
	IRN           string  `json:"irn" validate:"required"`
	PaymentStatus string  `json:"payment_status" validate:"required,oneof=PENDING PAID REJECTED"`
	Reference     *string `json:"reference,omitempty"`
}

type BulkUpdateInvoiceRequest struct {
	Invoices []BulkUpdateInvoiceItem `json:"invoices" validate:"required,min=1,dive"`
}

type BulkUpdateFailedItem struct {
	IRN   string `json:"irn"`
	Error string `json:"error"`
}

type BulkUpdateInvoiceResponse struct {
	Successful []string               `json:"successful"`
	Failed     []BulkUpdateFailedItem `json:"failed"`
}
