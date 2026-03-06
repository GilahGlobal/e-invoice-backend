package paystack

import (
	"encoding/json"
	"strings"
)

type InitializeTransactionMetadata struct {
	IsSandbox bool `json:"is_sandbox"`
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

type WebhookPayload struct {
	Event string `json:"event"`
	Data  struct {
		Status          string          `json:"status"`
		Reference       string          `json:"reference"`
		Amount          float64         `json:"amount"`
		GatewayResponse string          `json:"gateway_response"`
		PaidAt          string          `json:"paid_at"`
		Currency        string          `json:"currency"`
		Metadata        json.RawMessage `json:"metadata"`
		Customer        struct {
			Email string `json:"email"`
		} `json:"customer"`
	} `json:"data"`
}

func (payload WebhookPayload) MetadataIsSandbox() (bool, bool) {
	raw := strings.TrimSpace(string(payload.Data.Metadata))
	if raw == "" || raw == "null" {
		return false, false
	}

	// Support direct boolean metadata payloads, e.g. "metadata": true
	var directBool bool
	if err := json.Unmarshal(payload.Data.Metadata, &directBool); err == nil {
		return directBool, true
	}

	var metadata struct {
		IsSandbox *bool `json:"is_sandbox"`
	}

	if err := json.Unmarshal(payload.Data.Metadata, &metadata); err != nil {
		return false, false
	}
	if metadata.IsSandbox == nil {
		return false, false
	}

	return *metadata.IsSandbox, true
}
