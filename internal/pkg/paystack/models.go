package paystack

import (
	"strings"
)

type InitializeTransactionMetadata struct {
	IsSandbox    bool   `json:"is_sandbox"`
	BusinessID   string `json:"business_id,omitempty"`
	AggregatorID string `json:"aggregator_id,omitempty"`
	PlanID       string `json:"plan_id,omitempty"`
}

type InitializeTransactionRequest struct {
	Email     string                         `json:"email"`
	Amount    string                         `json:"amount"`
	Reference string                         `json:"reference"`
	Metadata  *InitializeTransactionMetadata `json:"metadata,omitempty"`
}

type InitializeTransactionResponse struct {
	Status  bool   `json:"status"`
	Message string `json:"message"`
	Data    struct {
		AuthorizationURL string `json:"authorization_url"`
		AccessCode       string `json:"access_code"`
		Reference        string `json:"reference"`
	} `json:"data"`
}

type PaystackWebhookPayload struct {
	Event string `json:"event"`
	Data  Data   `json:"data"`
}

type Data struct {
	ID                 int64         `json:"id"`
	Domain             string        `json:"domain"`
	Status             string        `json:"status"`
	Reference          string        `json:"reference"`
	Amount             int64         `json:"amount"`
	Message            *string       `json:"message"`
	GatewayResponse    string        `json:"gateway_response"`
	PaidAt             string        `json:"paid_at"`
	CreatedAt          string        `json:"created_at"`
	Channel            string        `json:"channel"`
	Currency           string        `json:"currency"`
	IPAddress          string        `json:"ip_address"`
	Metadata           Metadata      `json:"metadata"`
	FeesBreakdown      interface{}   `json:"fees_breakdown"`
	Log                interface{}   `json:"log"`
	Fees               int64         `json:"fees"`
	FeesSplit          interface{}   `json:"fees_split"`
	Authorization      Authorization `json:"authorization"`
	Customer           Customer      `json:"customer"`
	Plan               interface{}   `json:"plan"`
	Subaccount         interface{}   `json:"subaccount"`
	Split              interface{}   `json:"split"`
	OrderID            *string       `json:"order_id"`
	PaidAtAlt          string        `json:"paidAt"`
	RequestedAmount    int64         `json:"requested_amount"`
	PosTransactionData interface{}   `json:"pos_transaction_data"`
	Source             Source        `json:"source"`
}

type Metadata struct {
	IsSandbox    string `json:"is_sandbox"`
	BusinessID   string `json:"business_id"`
	AggregatorID string `json:"aggregator_id"`
	PlanID       string `json:"plan_id"`
	Referrer     string `json:"referrer"`
}

type Authorization struct {
	AuthorizationCode         string  `json:"authorization_code"`
	Bin                       string  `json:"bin"`
	Last4                     string  `json:"last4"`
	ExpMonth                  string  `json:"exp_month"`
	ExpYear                   string  `json:"exp_year"`
	Channel                   string  `json:"channel"`
	CardType                  string  `json:"card_type"`
	Bank                      string  `json:"bank"`
	CountryCode               string  `json:"country_code"`
	Brand                     string  `json:"brand"`
	Reusable                  bool    `json:"reusable"`
	Signature                 string  `json:"signature"`
	AccountName               *string `json:"account_name"`
	ReceiverBankAccountNumber *string `json:"receiver_bank_account_number"`
	ReceiverBank              *string `json:"receiver_bank"`
}

type Customer struct {
	ID                       int64       `json:"id"`
	FirstName                *string     `json:"first_name"`
	LastName                 *string     `json:"last_name"`
	Email                    string      `json:"email"`
	CustomerCode             string      `json:"customer_code"`
	Phone                    *string     `json:"phone"`
	Metadata                 interface{} `json:"metadata"`
	RiskAction               string      `json:"risk_action"`
	InternationalFormatPhone *string     `json:"international_format_phone"`
}

type Source struct {
	Type       string      `json:"type"`
	Source     string      `json:"source"`
	EntryPoint string      `json:"entry_point"`
	Identifier interface{} `json:"identifier"`
}

func (payload PaystackWebhookPayload) MetadataIsSandbox() (bool, bool) {
	raw := strings.TrimSpace(string(payload.Data.Metadata.IsSandbox))
	if raw == "" || raw == "null" {
		return false, false
	}

	if raw == "true" {
		return true, true
	}

	if raw == "false" {
		return false, true
	}

	return false, false
}
