package firs_models

type InvoiceRequest struct {
	InvoiceNumber               *string                `json:"invoice_number" validate:"omitempty,alphanum,min=1"`
	BusinessID                  string                 `json:"business_id" validate:"required,uuid"`
	IRN                         *string                `json:"irn" validate:"omitempty"`
	IssueDate                   string                 `json:"issue_date" validate:"required"`
	DueDate                     *string                `json:"due_date" validate:"omitempty"`
	IssueTime                   *string                `json:"issue_time" validate:"omitempty"`
	InvoiceTypeCode             string                 `json:"invoice_type_code" validate:"required"`
	PaymentStatus               string                 `json:"payment_status" validate:"oneof=PENDING PAID REJECTED"`
	Note                        *string                `json:"note" validate:"omitempty"`
	TaxPointDate                *string                `json:"tax_point_date" validate:"omitempty"`
	DocumentCurrencyCode        string                 `json:"document_currency_code" validate:"required"`
	TaxCurrencyCode             *string                `json:"tax_currency_code" validate:"omitempty"`
	AccountingCost              *string                `json:"accounting_cost" validate:"omitempty"`
	BuyerReference              *string                `json:"buyer_reference" validate:"omitempty"`
	InvoiceDeliveryPeriod       *InvoiceDeliveryPeriod `json:"invoice_delivery_period" validate:"omitempty"`
	OrderReference              *string                `json:"order_reference" validate:"omitempty"`
	BillingReference            []DocumentReference    `json:"billing_reference" validate:"omitempty,dive"`
	DispatchDocumentReference   *DocumentReference     `json:"dispatch_document_reference" validate:"omitempty"`
	ReceiptDocumentReference    *DocumentReference     `json:"receipt_document_reference" validate:"omitempty"`
	OriginatorDocumentReference *DocumentReference     `json:"originator_document_reference" validate:"omitempty"`
	ContractDocumentReference   *DocumentReference     `json:"contract_document_reference" validate:"omitempty"`
	AdditionalDocumentReference []DocumentReference    `json:"_document_reference" validate:"omitempty,dive"`
	AccountingSupplierParty     Party                  `json:"accounting_supplier_party" validate:"required"`
	AccountingCustomerParty     Party                  `json:"accounting_customer_party" validate:"required"`
	PayeeParty                  *Party                 `json:"payee_party" validate:"omitempty"`
	TaxRepresentativeParty      *Party                 `json:"tax_representative_party" validate:"omitempty"`
	ActualDeliveryDate          *string                `json:"actual_delivery_date" validate:"omitempty"`
	PaymentMeans                []PaymentMeans         `json:"payment_means" validate:"omitempty,dive"`
	PaymentTermsNote            *string                `json:"payment_terms_note" validate:"omitempty"`
	AllowanceCharge             []AllowanceCharge      `json:"allowance_charge" validate:"omitempty,dive"`
	TaxTotal                    []TaxTotal             `json:"tax_total" validate:"omitempty,dive"`
	LegalMonetaryTotal          LegalMonetaryTotal     `json:"legal_monetary_total" validate:"required"`
	InvoiceLine                 []InvoiceLine          `json:"invoice_line" validate:"required,dive"`
}

type InvoiceDeliveryPeriod struct {
	StartDate string `json:"start_date" validate:"required"`
	EndDate   string `json:"end_date" validate:"required"`
}

type DocumentReference struct {
	IRN       string `json:"irn" validate:"required"`
	IssueDate string `json:"issue_date" validate:"required"`
}

type Party struct {
	PartyName           *string        `json:"party_name" validate:"omitempty,min=2"`
	TIN                 string         `json:"tin" validate:"required"`
	Email               string         `json:"email" validate:"required,email"`
	Telephone           *string        `json:"telephone" validate:"omitempty,startswith=+,numeric,min=13,max=14"`
	BusinessDescription *string        `json:"business_description" validate:"omitempty,min=5"`
	PostalAddress       *PostalAddress `json:"postal_address" validate:"omitempty"`
}

type PostalAddress struct {
	StreetName  string `json:"street_name,omitempty" validate:"omitempty"`
	CityName    string `json:"city_name,omitempty" validate:"omitempty"`
	PostalZone  string `json:"postal_zone,omitempty" validate:"omitempty"`
	Country     string `json:"country,omitempty" validate:"omitempty"`
	CountryCode string `json:"country_code,omitempty" validate:"omitempty"`
}

type PaymentMeans struct {
	PaymentMeansCode string `json:"payment_means_code" validate:"required"`
	PaymentDueDate   string `json:"payment_due_date" validate:"required"`
}

type AllowanceCharge struct {
	ChargeIndicator bool    `json:"charge_indicator"`
	Amount          float64 `json:"amount" validate:"required"`
}

type TaxTotal struct {
	TaxAmount   float64       `json:"tax_amount" validate:"required"`
	TaxSubtotal []TaxSubtotal `json:"tax_subtotal,omitempty" validate:"omitempty,dive"`
}

type TaxSubtotal struct {
	TaxableAmount float64     `json:"taxable_amount" validate:"required"`
	TaxAmount     float64     `json:"tax_amount" validate:"required"`
	TaxCategory   TaxCategory `json:"tax_category" validate:"required"`
}

type TaxCategory struct {
	ID      string  `json:"id" validate:"required"`
	Percent float64 `json:"percent" validate:"required"`
}

type LegalMonetaryTotal struct {
	LineExtensionAmount float64 `json:"line_extension_amount" validate:"required"`
	TaxExclusiveAmount  float64 `json:"tax_exclusive_amount" validate:"required"`
	TaxInclusiveAmount  float64 `json:"tax_inclusive_amount" validate:"required"`
	PayableAmount       float64 `json:"payable_amount" validate:"required"`
}

type InvoiceLine struct {
	HSNCode             string  `json:"hsn_code" validate:"required"`
	ProductCategory     string  `json:"product_category" validate:"required"`
	DiscountRate        float64 `json:"discount_rate"`
	DiscountAmount      float64 `json:"discount_amount"`
	FeeRate             float64 `json:"fee_rate"`
	FeeAmount           float64 `json:"fee_amount"`
	InvoicedQuantity    int     `json:"invoiced_quantity" validate:"required,min=1"`
	LineExtensionAmount float64 `json:"line_extension_amount" validate:"required"`
	Item                Item    `json:"item" validate:"required"`
	Price               Price   `json:"price" validate:"required"`
}

type Item struct {
	Name                      string  `json:"name" validate:"required"`
	Description               string  `json:"description"`
	SellersItemIdentification *string `json:"sellers_item_identification,omitempty"`
}

type Price struct {
	PriceAmount  float64 `json:"price_amount" validate:"required"`
	BaseQuantity int     `json:"base_quantity" validate:"required"`
	PriceUnit    string  `json:"price_unit" validate:"required"`
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
