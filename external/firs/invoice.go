package firs

import (
	"einvoice-access-point/external/firs_models"
	"einvoice-access-point/internal/dtos"
	"einvoice-access-point/pkg/config"
	"einvoice-access-point/pkg/utility"
	"fmt"
)

func ValidateIRN(invoiceReq firs_models.IRNValidationRequest) (*utility.Response, error) {

	var (
		configs = config.GetConfig()
		apiURL  = fmt.Sprintf("%v/invoice/irn/validate", configs.Firs.FirsApiUrl)
	)

	config := utility.RequestConfig{
		URL: apiURL,
		Headers: map[string]string{
			"x-api-key":    configs.Firs.FirsApiKey,
			"x-api-secret": configs.Firs.FirsClientKey,
		},
		Body: invoiceReq,
	}

	theResp := &firs_models.FirsResponse{}

	return utility.PostRequest(utility.DefaultHTTPClient, config, theResp)
}

func ValidateInvoice(req dtos.UploadInvoiceRequestDto) (*utility.Response, error) {

	var (
		configs = config.GetConfig()
		apiURL  = fmt.Sprintf("%v/invoice/validate", configs.Firs.FirsApiUrl)
	)

	config := utility.RequestConfig{
		URL: apiURL,
		Headers: map[string]string{
			"x-api-key":    configs.Firs.FirsApiKey,
			"x-api-secret": configs.Firs.FirsClientKey,
		},
		Body: req,
	}

	theResp := &firs_models.FirsResponse{}

	return utility.PostRequest(utility.DefaultHTTPClient, config, theResp)
}

func SignInvoice(req dtos.UploadInvoiceRequestDto) (*utility.Response, error) {

	var (
		configs = config.GetConfig()
		apiURL  = fmt.Sprintf("%v/invoice/sign", configs.Firs.FirsApiUrl)
	)

	config := utility.RequestConfig{
		URL: apiURL,
		Headers: map[string]string{
			"x-api-key":    configs.Firs.FirsApiKey,
			"x-api-secret": configs.Firs.FirsClientKey,
		},
		Body: req,
	}

	theResp := &firs_models.FirsResponse{}

	return utility.PostRequest(utility.DefaultHTTPClient, config, theResp)
}

func ConfirmInvoice(irn string) (*utility.Response, error) {

	var (
		configs = config.GetConfig()
		apiURL  = fmt.Sprintf("%v/invoice/confirm/%s", configs.Firs.FirsApiUrl, irn)
	)

	config := utility.RequestConfig{
		URL: apiURL,
		Headers: map[string]string{
			"x-api-key":    configs.Firs.FirsApiKey,
			"x-api-secret": configs.Firs.FirsClientKey,
		},
		Body: nil,
	}

	var theResp = &firs_models.FirsResponse{}

	return utility.GetRequest(utility.DefaultHTTPClient, config, theResp)
}

func DownloadInvoice(irn string) (*utility.Response, error) {

	var (
		configs = config.GetConfig()
		apiURL  = fmt.Sprintf("%v/invoice/download/%s", configs.Firs.FirsApiUrl, irn)
	)

	config := utility.RequestConfig{
		URL: apiURL,
		Headers: map[string]string{
			"x-api-key":    configs.Firs.FirsApiKey,
			"x-api-secret": configs.Firs.FirsClientKey,
		},
		Body: nil,
	}

	var theResp = &firs_models.FirsResponse{}

	return utility.GetRequest(utility.DefaultHTTPClient, config, theResp)
}

func UpdateInvoice(invoiceUpdate firs_models.UpdateInvoice, irn string) (*utility.Response, error) {

	var (
		configs = config.GetConfig()
		apiURL  = fmt.Sprintf("%v/invoice/update/%s", configs.Firs.FirsApiUrl, irn)
	)

	config := utility.RequestConfig{
		URL: apiURL,
		Headers: map[string]string{
			"x-api-key":    configs.Firs.FirsApiKey,
			"x-api-secret": configs.Firs.FirsClientKey,
		},
		Body: invoiceUpdate,
	}

	theResp := &firs_models.FirsResponse{}

	return utility.PatchRequest(utility.DefaultHTTPClient, config, theResp)
}
