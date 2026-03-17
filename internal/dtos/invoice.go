package dtos

type UploadInvoiceRequestDto struct {
	InvoiceNumber               string                 `json:"invoice_number" example:"INV-001" validate:"min=1"`
	BusinessID                  string                 `json:"business_id" example:"123e4567-e89b-12d3-a456-426614174000" validate:"required,uuid"`
	IRN                         *string                `json:"irn" example:"IRN-001-20122345" validate:"omitempty"`
	IssueDate                   string                 `json:"issue_date" example:"2026-01-16" validate:"required,nrsdate"`
	DueDate                     *string                `json:"due_date" example:"2026-01-20" validate:"omitempty,nrsdate"`
	IssueTime                   *string                `json:"issue_time" example:"12:00:00" validate:"omitempty"`
	InvoiceTypeCode             string                 `json:"invoice_type_code" example:"381" validate:"required,oneof=380 381 384 385 386 388 389 390 392 393 394 395 396 397 399 400 402 404 406 408"`
	PaymentStatus               *string                `json:"payment_status" example:"PENDING" validate:"omitempty,oneof=PENDING PAID REJECTED"`
	Note                        *string                `json:"note" example:"Invoice note" validate:"omitempty"`
	TaxPointDate                *string                `json:"tax_point_date" example:"2026-01-16" validate:"omitempty,nrsdate"`
	DocumentCurrencyCode        string                 `json:"document_currency_code" example:"NGN" validate:"required"`
	TaxCurrencyCode             string                 `json:"tax_currency_code" example:"NGN" validate:"required"`
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
	ActualDeliveryDate          *string                `json:"actual_delivery_date" example:"2026-01-16" validate:"omitempty,nrsdate"`
	PaymentMeans                []PaymentMeans         `json:"payment_means" validate:"omitempty,dive"`
	PaymentTermsNote            *string                `json:"payment_terms_note" example:"Payment terms note" validate:"omitempty"`
	AllowanceCharge             []AllowanceCharge      `json:"allowance_charge" validate:"omitempty,dive"`
	TaxTotal                    []TaxTotal             `json:"tax_total" validate:"required,dive"`
	LegalMonetaryTotal          LegalMonetaryTotal     `json:"legal_monetary_total" validate:"required"`
	InvoiceLine                 []InvoiceLine          `json:"invoice_line" validate:"required,dive"`
	SmeID                       *string                `json:"sme_id" example:"123e4567-e89b-12d3-a456-426614174000" validate:"omitempty,uuid"`
}

type InvoiceDeliveryPeriod struct {
	StartDate string `json:"start_date" example:"2026-01-16" validate:"required,nrsdate"`
	EndDate   string `json:"end_date" example:"2026-01-16" validate:"required,nrsdate"`
}

type DocumentReference struct {
	IRN       string `json:"irn" example:"ITW001-E9E0C0D3-20240619" validate:"required"`
	IssueDate string `json:"issue_date" example:"2026-01-16" validate:"required,nrsdate"`
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
	StreetName string `json:"street_name" example:"123 Broad Street" validate:"required"`
	CityName   string `json:"city_name" example:"Ikeja" validate:"required"`
	PostalZone string `json:"postal_zone" example:"10001" validate:"required"`
	LGA        string `json:"lga" example:"NG-AB-ANO" validate:"required"`
	State      string `json:"state" example:"NG-AB" validate:"required"`
	Country    string `json:"country" example:"NG" validate:"required"`
}

type PaymentMeans struct {
	PaymentMeansCode string `json:"payment_means_code" example:"10" validate:"required"`
	PaymentDueDate   string `json:"payment_due_date" example:"2026-01-16" validate:"required,nrsdate"`
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
	ID      string  `json:"id" example:"STANDARD_VAT" validate:"required,oneof=STANDARD_GST REDUCED_GST ZERO_GST STANDARD_VAT REDUCED_VAT ZERO_VAT STATE_SALES_TAX LOCAL_SALES_TAX ALCOHOL_EXCISE_TAX TOBACCO_EXCISE_TAX FUEL_EXCISE_TAX CORPORATE_INCOME_TAX PERSONAL_INCOME_TAX SOCIAL_SECURITY_TAX MEDICARE_TAX REAL_ESTATE_TAX PERSONAL_PROPERTY_TAX CARBON_TAX PLASTIC_TAX IMPORT_DUTY EXPORT_DUTY LUXURY_TAX SERVICE_TAX TOURISM_TAX"`
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

type InvoiceData struct {
	InvoiceNumber string `json:"invoice_number" example:"INV-1001"`
	IRN           string `json:"irn" example:"123e4567-e89b-12d3-a456-426614174000"`
	QRCode        string `json:"qr_code" example:"iVBORw0KGgoAAAANSUhEUgAAAQAAAAEAAQMAAABmvDolAAAABlBMVEX///8AAABVwtN..."`
	EncryptedIRN  string `json:"encrypted_irn" example:"eeleGz7LXrt3gignmXGi9DAeXoVS7GjMR/8WK4f8G76DSP14SA2PSyArr4oaS6ojo0EqCTlp2UBjT2eRpn51..."`
}

type InvoiceStepMetadata struct {
	Step      string `json:"step" example:"validated_irn"`
	Status    string `json:"status" example:"success"`
	Timestamp string `json:"timestamp" example:"2024-01-01T12:00:00Z"`
}

type UploadInvoiceResponseDto struct {
	BaseResponseDto
	Data     InvoiceData           `json:"data"`
	Metadata []InvoiceStepMetadata `json:"metadata"`
}

type MinimalInvoiceDTO struct {
	ID            string `json:"id" example:"123e4567-e89b-12d3-a456-426614174000"`
	InvoiceNumber string `json:"invoice_number" example:"INV-1001"`
	IRN           string `json:"irn" example:"INV-1001-a456-426614174000"`
	Platform      string `json:"platform" example:"internal"`
	CurrentStatus string `json:"current_status" example:"validated_irn"`
	StatusText    string `json:"status_text" example:"success"`
	CreatedAt     string `json:"created_at" example:"2024-01-01T12:00:00Z"`
}

type GetAllInvoicesResponseDto struct {
	BaseResponseDto
	Data []MinimalInvoiceDTO `json:"data"`
}

type Invoice struct {
	ID               string                  `json:"id" example:"123e4567-e89b-12d3-a456-426614174000"`
	InvoiceNumber    string                  `json:"invoice_number" example:"INV-1001"`
	IRN              string                  `json:"irn" example:"123e4567-e89b-12d3-a456-426614174000"`
	BusinessID       string                  `json:"business_id" example:"business-uuid"`
	Platform         string                  `json:"platform" example:"zoho"` // e.g., zoho, quickbooks
	PlatformMetadata string                  `json:"platform_metadata"`
	InvoiceData      UploadInvoiceRequestDto `json:"invoice_data"`
	CurrentStatus    string                  `json:"current_status" example:"validated_irn"`
	StatusHistory    []InvoiceStepMetadata   `json:"status_history"`
	Timestamp        string                  `json:"timestamp" example:"2024-01-01T12:00:00Z"`
	CreatedAt        string                  `json:"created_at" example:"2024-01-01T12:00:00Z"`
	UpdatedAt        string                  `json:"updated_at" example:"2024-01-02T12:00:00Z"`
}

type GetInvoiceDetailsResponseDto struct {
	BaseResponseDto
	Data Invoice `json:"data"`
}
