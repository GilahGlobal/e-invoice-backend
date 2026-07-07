package firs

import (
	"einvoice-access-point/internal/pkg/firs_models"
	"einvoice-access-point/internal/utility"
	"encoding/json"
	"fmt"
)

func ParseFIRSAPIResponse(resp *utility.Response) (*firs_models.FirsResponse, *string, error) {
	if resp == nil {
		return nil, nil, fmt.Errorf("nil response received")
	}

	if string(resp.Body) == `{"healthy":true}` {
		return &firs_models.FirsResponse{Data: map[string]bool{"healthy": true}}, nil, nil
	}

	var maybeFirsErr struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Error   struct {
			Handler       string `json:"handler"`
			Details       string `json:"details"`
			PublicMessage string `json:"public_message"`
		} `json:"error"`
		Data struct {
			ID      string `json:"id"`
			Status  string `json:"status"`
			Message string `json:"message"`
		} `json:"data"`
	}

	if err := json.Unmarshal(resp.Body, &maybeFirsErr); err == nil {
		if resp.StatusCode < 200 || resp.StatusCode >= 300 || maybeFirsErr.Code < 200 || maybeFirsErr.Code >= 300 {

			msg := maybeFirsErr.Message
			pubMsg := maybeFirsErr.Error.PublicMessage

			if msg == "" && pubMsg == "" && maybeFirsErr.Data.Message != "" {
				msg = maybeFirsErr.Data.Message
			}

			if msg == "" && pubMsg == "" {
				msg = "unknown error"
				return nil, &maybeFirsErr.Error.Details, fmt.Errorf("api error %d: %s", resp.StatusCode, msg)
			} else if msg != "" && pubMsg != "" {
				return nil, &maybeFirsErr.Error.Details, fmt.Errorf("api error %d: %s - %s", resp.StatusCode, msg, pubMsg)
			} else if msg != "" {
				return nil, &maybeFirsErr.Error.Details, fmt.Errorf("api error %d: %s", resp.StatusCode, msg)
			} else {
				return nil, &maybeFirsErr.Error.Details, fmt.Errorf("api error %d: %s", resp.StatusCode, pubMsg)
			}
		}
	}

	var firsResp firs_models.FirsResponse
	if err := json.Unmarshal(resp.Body, &firsResp); err != nil {
		return nil, nil, fmt.Errorf("failed to unmarshal base response: %w", err)
	}

	var raw struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(resp.Body, &raw); err != nil {
		return nil, nil, fmt.Errorf("failed to extract raw data: %w", err)
	}

	var fallback map[string]interface{}
	if err := json.Unmarshal(raw.Data, &fallback); err == nil {
		firsResp.Data = fallback
		return &firsResp, nil, nil
	}

	firsResp.Data = raw.Data
	return &firsResp, nil, nil
}
