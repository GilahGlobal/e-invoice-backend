package firs

import (
	"einvoice-access-point/external/firs_models"
	"einvoice-access-point/internal/dtos"
	"einvoice-access-point/pkg/config"
	"einvoice-access-point/pkg/utility"
	"fmt"
)

func buildFirsRequest(endpoint string, IsSandbox bool) utility.RequestConfig {
	cfg := config.GetConfig()
	var apiURL string
	var headers map[string]string

	if IsSandbox {
		apiURL = fmt.Sprintf("%v/%s", cfg.FirsSandbox.FirsApiUrl, endpoint)
		headers = map[string]string{
			"x-api-key":    cfg.FirsSandbox.FirsApiKey,
			"x-api-secret": cfg.FirsSandbox.FirsClientKey,
		}
	} else {
		apiURL = fmt.Sprintf("%v/%s", cfg.Firs.FirsApiUrl, endpoint)
		headers = map[string]string{
			"x-api-key":    cfg.Firs.FirsApiKey,
			"x-api-secret": cfg.Firs.FirsClientKey,
		}
	}

	return utility.RequestConfig{
		URL:     apiURL,
		Headers: headers,
	}
}

func ValidateIRN(req firs_models.IRNValidationRequest, IsSandbox bool) (*utility.Response, error) {
	config := buildFirsRequest("invoice/irn/validate", IsSandbox)
	config.Body = req
	theResp := &firs_models.FirsResponse{}
	return utility.PostRequest(utility.DefaultHTTPClient, config, theResp)
}

func ValidateInvoice(req dtos.UploadInvoiceRequestDto, IsSandbox bool) (*utility.Response, error) {
	config := buildFirsRequest("invoice/validate", IsSandbox)
	config.Body = req
	theResp := &firs_models.FirsResponse{}
	return utility.PostRequest(utility.DefaultHTTPClient, config, theResp)
}

func SignInvoice(req dtos.UploadInvoiceRequestDto, IsSandbox bool) (*utility.Response, error) {
	config := buildFirsRequest("invoice/sign", IsSandbox)
	config.Body = req
	theResp := &firs_models.FirsResponse{}
	return utility.PostRequest(utility.DefaultHTTPClient, config, theResp)
}

func ConfirmInvoice(irn string, IsSandbox bool) (*utility.Response, error) {
	config := buildFirsRequest(fmt.Sprintf("invoice/confirm/%s", irn), IsSandbox)
	theResp := &firs_models.FirsResponse{}
	return utility.GetRequest(utility.DefaultHTTPClient, config, theResp)
}

func DownloadInvoice(irn string, IsSandbox bool) (*utility.Response, error) {
	config := buildFirsRequest(fmt.Sprintf("invoice/download/%s", irn), IsSandbox)
	theResp := &firs_models.FirsResponse{}
	return utility.GetRequest(utility.DefaultHTTPClient, config, theResp)
}

func UpdateInvoice(req firs_models.UpdateInvoice, irn string, IsSandbox bool) (*utility.Response, error) {
	config := buildFirsRequest(fmt.Sprintf("invoice/update/%s", irn), IsSandbox)
	config.Body = req
	theResp := &firs_models.FirsResponse{}
	return utility.PatchRequest(utility.DefaultHTTPClient, config, theResp)
}
