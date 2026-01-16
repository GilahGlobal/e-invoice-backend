package firs_models

type InvoiceRequest struct {
	InvoiceNumber               *string                `json:"invoice_number" example:"INV-001" validate:"omitempty,alphanum,min=1"`
	BusinessID                  string                 `json:"business_id" example:"123e4567-e89b-12d3-a456-426614174000" validate:"required,uuid"`
	IRN                         *string                `json:"irn" example:"IRN-001-20122345" validate:"omitempty"`
	IssueDate                   string                 `json:"issue_date" example:"2026-01-16" validate:"required"`
	DueDate                     *string                `json:"due_date" example:"2026-01-20" validate:"omitempty"`
	IssueTime                   *string                `json:"issue_time" example:"12:00:00" validate:"omitempty"`
	InvoiceTypeCode             string                 `json:"invoice_type_code" example:"381" validate:"required"`
	PaymentStatus               string                 `json:"payment_status" example:"PENDING" validate:"oneof=PENDING PAID REJECTED"`
	Note                        *string                `json:"note" example:"Invoice note" validate:"omitempty"`
	TaxPointDate                *string                `json:"tax_point_date" example:"2026-01-16" validate:"omitempty"`
	DocumentCurrencyCode        string                 `json:"document_currency_code" example:"NGN" validate:"required"`
	TaxCurrencyCode             *string                `json:"tax_currency_code" example:"NGN" validate:"omitempty"`
	AccountingCost              *string                `json:"accounting_cost" example:"2000" validate:"omitempty"`
	BuyerReference              *string                `json:"buyer_reference" example:"ITW001-E9E0C0D3-20240619" validate:"omitempty"`
	InvoiceDeliveryPeriod       *InvoiceDeliveryPeriod `json:"invoice_delivery_period" validate:"omitempty"`
	OrderReference              *string                `json:"order_reference" example:"ITW001-E9E0C0D3-20240619" validate:"omitempty"`
	BillingReference            []DocumentReference    `json:"billing_reference" validate:"omitempty,dive"`
	DispatchDocumentReference   *DocumentReference     `json:"dispatch_document_reference" validate:"omitempty"`
	ReceiptDocumentReference    *DocumentReference     `json:"receipt_document_reference" validate:"omitempty"`
	OriginatorDocumentReference *DocumentReference     `json:"originator_document_reference" validate:"omitempty"`
	ContractDocumentReference   *DocumentReference     `json:"contract_document_reference" validate:"omitempty"`
	AdditionalDocumentReference []DocumentReference    `json:"_document_reference" validate:"omitempty,dive"`
	AccountingSupplierParty     Party                  `json:"accounting_supplier_party" validate:"required"`
	AccountingCustomerParty     *Party                 `json:"accounting_customer_party" validate:"omitempty"`
	PayeeParty                  *Party                 `json:"payee_party" validate:"omitempty"`
	TaxRepresentativeParty      *Party                 `json:"tax_representative_party" validate:"omitempty"`
	ActualDeliveryDate          *string                `json:"actual_delivery_date" example:"2026-01-16" validate:"omitempty"`
	PaymentMeans                []PaymentMeans         `json:"payment_means" validate:"omitempty,dive"`
	PaymentTermsNote            *string                `json:"payment_terms_note" example:"Payment terms note" validate:"omitempty"`
	AllowanceCharge             []AllowanceCharge      `json:"allowance_charge" validate:"omitempty,dive"`
	TaxTotal                    []TaxTotal             `json:"tax_total" validate:"required,dive"`
	LegalMonetaryTotal          LegalMonetaryTotal     `json:"legal_monetary_total" validate:"required"`
	InvoiceLine                 []InvoiceLine          `json:"invoice_line" validate:"required,dive"`
}

type InvoiceDeliveryPeriod struct {
	StartDate string `json:"start_date" example:"2026-01-16" validate:"required"`
	EndDate   string `json:"end_date" example:"2026-01-16" validate:"required"`
}

type DocumentReference struct {
	IRN       string `json:"irn" example:"ITW001-E9E0C0D3-20240619" validate:"required"`
	IssueDate string `json:"issue_date" example:"2026-01-16" validate:"required"`
}

type Party struct {
	PartyName           string         `json:"party_name" example:"Acme Inc." validate:"required,min=2"`
	TIN                 string         `json:"tin" example:"123456789012345" validate:"required"`
	Email               string         `json:"email" example:"business@example.com" validate:"required,email"`
	Telephone           *string        `json:"telephone" example:"+234804567890" validate:"omitempty,startswith=+,numeric,min=7"`
	BusinessDescription *string        `json:"business_description" example:"Acme Inc. is a leading technology company." validate:"omitempty,min=5"`
	PostalAddress       *PostalAddress `json:"postal_address" validate:"required"`
}

type PostalAddress struct {
	StreetName  string `json:"street_name" example:"123 Broad Street" validate:"required"`
	CityName    string `json:"city_name" example:"Ikeja" validate:"required"`
	PostalZone  string `json:"postal_zone" example:"10001" validate:"required"`
	Country     string `json:"country" example:"Nigeria" validate:"required"`
	CountryCode string `json:"country_code" example:"NG" validate:"required"`
}

type PaymentMeans struct {
	PaymentMeansCode string `json:"payment_means_code" example:"10" validate:"required"`
	PaymentDueDate   string `json:"payment_due_date" example:"2026-01-16" validate:"required"`
}

type AllowanceCharge struct {
	ChargeIndicator bool    `json:"charge_indicator" example:"true"`
	Amount          float64 `json:"amount" validate:"required" example:"1500.75"`
}

type TaxTotal struct {
	TaxAmount   float64       `json:"tax_amount" example:"1500.75" validate:"required"`
	TaxSubtotal []TaxSubtotal `json:"tax_subtotal" validate:"omitempty,dive"`
}

type TaxSubtotal struct {
	TaxableAmount float64     `json:"taxable_amount" example:"1500.75" validate:"required"`
	TaxAmount     float64     `json:"tax_amount" example:"1500.75" validate:"required"`
	TaxCategory   TaxCategory `json:"tax_category" validate:"required"`
}

type TaxCategory struct {
	ID      string  `json:"id" example:"VAT" validate:"required"`
	Percent float64 `json:"percent" example:"15.00" validate:"required"`
}

type LegalMonetaryTotal struct {
	LineExtensionAmount float64 `json:"line_extension_amount" example:"1500.75" validate:"required"`
	TaxExclusiveAmount  float64 `json:"tax_exclusive_amount" example:"1500.75" validate:"required"`
	TaxInclusiveAmount  float64 `json:"tax_inclusive_amount" example:"1700.75" validate:"required"`
	PayableAmount       float64 `json:"payable_amount" example:"1700.75" validate:"required"`
}

type InvoiceLine struct {
	HSNCode             string  `json:"hsn_code" example:"1282.10" validate:"required"`
	ProductCategory     string  `json:"product_category" example:"Electronics" validate:"required"`
	DiscountRate        float64 `json:"discount_rate" example:"5"`
	DiscountAmount      float64 `json:"discount_amount" example:"2500"`
	FeeRate             float64 `json:"fee_rate" example:"2"`
	FeeAmount           float64 `json:"fee_amount" example:"450"`
	InvoicedQuantity    int     `json:"invoiced_quantity" example:"10" validate:"required,min=1"`
	LineExtensionAmount float64 `json:"line_extension_amount" example:"1500.75" validate:"required"`
	Item                Item    `json:"item" validate:"required"`
	Price               Price   `json:"price" validate:"required"`
}

type Item struct {
	Name                      string  `json:"name" example:"Laptop" validate:"required"`
	Description               string  `json:"description" example:"A high-performance laptop suitable for gaming and work." validate:"omitempty"`
	SellersItemIdentification *string `json:"sellers_item_identification" example:"LAP-12345" validate:"omitempty"`
}

type Price struct {
	PriceAmount  float64 `json:"price_amount" example:"5000" validate:"required"`
	BaseQuantity int     `json:"base_quantity" example:"1" validate:"required"`
	PriceUnit    string  `json:"price_unit" example:"NGN per 1" validate:"required"`
}

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
	EncryptedMessage string `json:"encrypted_message"`
	QrCodeImage      string `json:"qr_code_image"`
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
	PaymentStatus string  `json:"payment_status" validate:"required,oneof=PENDING PAID REJECTED"`
	Reference     *string `json:"reference,omitempty"`
}

type FirsWebhookPayload struct {
	IRN     string `json:"irn" validate:"required"`
	Message string `json:"message" validate:"required"`
}
