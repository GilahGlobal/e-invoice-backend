package zoho

type WebhookPayload struct {
	Invoice Invoice `json:"invoice" validate:"required"`
}

type Invoice struct {
	InvoiceID             string          `json:"invoice_id" validate:"required"`
	InvoiceNumber         string          `json:"invoice_number" validate:"required"`
	Status                string          `json:"status" validate:"required"`
	CustomerID            string          `json:"customer_id" validate:"required"`
	CustomerName          string          `json:"customer_name" validate:"required"`
	Email                 string          `json:"email" validate:"email"`
	ContactPersonsDetails []ContactPerson `json:"contact_persons_details"`                      // For phone extraction
	Date                  string          `json:"date" validate:"required,datetime=2006-01-02"` // Invoice date
	DueDate               string          `json:"due_date" validate:"required,datetime=2006-01-02"`
	Total                 float64         `json:"total" validate:"gte=0"`
	Balance               float64         `json:"balance" validate:"gte=0"`
	CurrencyCode          string          `json:"currency_code" validate:"required"`
	ExchangeRate          float64         `json:"exchange_rate"`
	LineItems             []LineItem      `json:"line_items" validate:"dive"`
	BillingAddress        Address         `json:"billing_address"`
	ShippingAddress       Address         `json:"shipping_address"`
	Notes                 string          `json:"notes"`
	Terms                 string          `json:"terms"`
	PaymentTerms          int             `json:"payment_terms"`
	Adjustment            float64         `json:"adjustment"`
	ShippingCharge        float64         `json:"shipping_charge"`
	CreatedTime           string          `json:"created_time"`
	LastModifiedTime      string          `json:"last_modified_time"`
	SalespersonName       string          `json:"salesperson_name"`
}

type ContactPerson struct {
	Phone     string `json:"phone"`
	Mobile    string `json:"mobile"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Email     string `json:"email"`
}

type LineItem struct {
	ItemID        string  `json:"item_id" validate:"required"`
	Name          string  `json:"name" validate:"required"`
	Description   string  `json:"description"`
	Quantity      float64 `json:"quantity" validate:"gte=0"`
	Rate          float64 `json:"rate" validate:"gte=0"`
	ItemTotal     float64 `json:"item_total" validate:"gte=0"`
	TaxID         *string `json:"tax_id"`   // Nullable
	TaxName       *string `json:"tax_name"` // Nullable
	TaxPercentage float64 `json:"tax_percentage"`
	Discount      float64 `json:"discount"`
}

type Address struct {
	Address     string `json:"address"`
	Street      string `json:"street"`
	Street2     string `json:"street2"`
	City        string `json:"city"`
	State       string `json:"state"`
	Zip         string `json:"zip"`
	Country     string `json:"country"`
	CountryCode string `json:"country_code"`
	Phone       string `json:"phone"`
	Fax         string `json:"fax"`
}

// ZohoCustomField represents one custom field object in Zoho
type ZohoCustomField struct {
	ApiName string      `json:"api_name"`
	Value   interface{} `json:"value"`
}

// ZohoUpdateInvoice is the wrapper for updating invoice custom fields
type ZohoUpdateInvoice struct {
	CustomFields []ZohoCustomField `json:"custom_fields"`
}

// WebhookResponse represents the response structure
type WebhookResponse struct {
	InvoiceID      string  `json:"invoice_id"`
	InvoiceNumber  string  `json:"invoice_number"`
	CustomerName   string  `json:"customer_name"`
	Total          float64 `json:"total"`
	OrganizationID string  `json:"organization_id"`
	Updated        bool    `json:"updated"`
}

type OAuthTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token,omitempty"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
	Scope        string `json:"scope"`
	API_Domain   string `json:"api_domain"`
	Error        string `json:"error"`
}

// TokenResponse represents the structure of the token response from Zoho.
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	API_Domain   string `json:"api_domain"`
	TokenType    string `json:"token_type"`
	Error        string `json:"error"`
}
