package paystack

import (
	"einvoice-access-point/pkg/config"
	"einvoice-access-point/pkg/utility"
	"fmt"
)

func InitializeTransaction(payload InitializeTransactionRequest) (*InitializeTransactionResponse, error) {
	cfg := config.GetConfig()
	if cfg.Paystack.SecretKey == "" {
		return nil, fmt.Errorf("paystack secret key is not configured")
	}

	initializeURL := cfg.Paystack.InitializeURL
	if initializeURL == "" {
		initializeURL = "https://api.paystack.co/transaction/initialize"
	}

	req := utility.RequestConfig{
		URL: initializeURL,
		Headers: map[string]string{
			"Authorization": fmt.Sprintf("Bearer %s", cfg.Paystack.SecretKey),
		},
		Body: payload,
	}

	resp := &InitializeTransactionResponse{}
	_, err := utility.PostRequest(utility.DefaultHTTPClient, req, resp)
	if err != nil {
		return nil, err
	}

	return resp, nil
}
