package firs_models

type FirsTransactionVatPayload struct {
	AgentTIN            string `json:"agentTin"`
	BaseAmount          string `json:"baseAmount"`
	BeneficiaryTIN      string `json:"beneficiaryTin"`
	Currency            int    `json:"currency"`
	ItemDescription     string `json:"itemDescription"`
	OtherTaxes          string `json:"otherTaxes"`
	TotalAmount         string `json:"totalAmount"`
	TransactionDate     string `json:"transDate"`
	VATCalculated       string `json:"vatCalculated"`
	VATRate             string `json:"vatRate"`
	VATStatus           int    `json:"vatStatus"`
	VendorTransactionID string `json:"vendorTransactionId"`
}

type PartyRegistrationPayload struct {
	BusinessID          string         `json:"business_id"`
	PartyName           string         `json:"party_name"`
	PostalAddressID     string         `json:"postal_address_id,omitempty"`
	TIN                 string         `json:"tin"`
	Email               string         `json:"email"`
	Telephone           string         `json:"telephone,omitempty"`
	BusinessDescription string         `json:"business_description,omitempty"`
	PostalAddress       *PostalAddress `json:"postal_address,omitempty"`
}
