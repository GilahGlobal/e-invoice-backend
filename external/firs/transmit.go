package firs

import (
	"einvoice-access-point/external/firs_models"
	"einvoice-access-point/pkg/config"
	"einvoice-access-point/pkg/models"
	"einvoice-access-point/pkg/utility"
	"fmt"
)

func LookUpByIRN(irn string) (*utility.Response, error) {

	var (
		configs = config.GetConfig()
		apiURL  = fmt.Sprintf("%v/invoice/transmit/lookup/%s", configs.Firs.FirsApiUrl, irn)
	)

	config := utility.RequestConfig{
		URL: apiURL,
		Headers: map[string]string{
			"x-api-key":    configs.Firs.FirsApiKey,
			"x-api-secret": configs.Firs.FirsClientKey,
		},
		Body: nil,
	}

	theResp := &firs_models.FirsResponse{}

	return utility.GetRequest(utility.DefaultHTTPClient, config, theResp)
}

func LookUpByTIN(tin string) (*utility.Response, error) {

	var (
		configs = config.GetConfig()
		apiURL  = fmt.Sprintf("%v/invoice/transmit/lookup/tin/%s", configs.Firs.FirsApiUrl, tin)
	)

	config := utility.RequestConfig{
		URL: apiURL,
		Headers: map[string]string{
			"x-api-key":    configs.Firs.FirsApiKey,
			"x-api-secret": configs.Firs.FirsClientKey,
		},
		Body: nil,
	}

	theResp := &firs_models.FirsResponse{}

	return utility.GetRequest(utility.DefaultHTTPClient, config, theResp)
}

func LookUpByPartyID(partyId string) (*utility.Response, error) {

	var (
		configs = config.GetConfig()
		apiURL  = fmt.Sprintf("%v/invoice/transmit/lookup/party/%s", configs.Firs.FirsApiUrl, partyId)
	)

	config := utility.RequestConfig{
		URL: apiURL,
		Headers: map[string]string{
			"x-api-key":    configs.Firs.FirsApiKey,
			"x-api-secret": configs.Firs.FirsClientKey,
		},
		Body: nil,
	}

	theResp := &firs_models.FirsResponse{}

	return utility.GetRequest(utility.DefaultHTTPClient, config, theResp)
}

func TransmitInvoice(irn string, IsSandbox bool) (*utility.Response, error) {
	configs := config.GetConfig()
	var apiURL string
	var headers map[string]string

	if IsSandbox {
		apiURL = fmt.Sprintf("%v/invoice/transmit/%s", configs.FirsSandbox.FirsApiUrl, irn)
		headers = map[string]string{
			"x-api-key":    configs.FirsSandbox.FirsApiKey,
			"x-api-secret": configs.FirsSandbox.FirsClientKey,
		}
	} else {
		apiURL = fmt.Sprintf("%v/invoice/transmit/%s", configs.Firs.FirsApiUrl, irn)
		headers = map[string]string{
			"x-api-key":    configs.Firs.FirsApiKey,
			"x-api-secret": configs.Firs.FirsClientKey,
		}
	}

	config := utility.RequestConfig{
		URL:     apiURL,
		Headers: headers,
		Body:    nil,
	}

	var theResp = &firs_models.FirsResponse{}

	return utility.PostRequest(utility.DefaultHTTPClient, config, theResp)
}

func TransmitConfirmInvoice(irn string) (*utility.Response, error) {

	var (
		configs = config.GetConfig()
		apiURL  = fmt.Sprintf("%v/invoice/transmit/%s", configs.Firs.FirsApiUrl, irn)
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

	return utility.PatchRequest(utility.DefaultHTTPClient, config, theResp)
}

func TransmitPull(query models.PullDataQuery) (*utility.Response, error) {

	var (
		configs = config.GetConfig()
		apiURL  = fmt.Sprintf("%v/invoice/transmit/pull", configs.Firs.FirsApiUrl)
	)

	config := utility.RequestConfig{
		URL: apiURL,
		Headers: map[string]string{
			"x-api-key":    configs.Firs.FirsApiKey,
			"x-api-secret": configs.Firs.FirsClientKey,
		},
		Body: nil,
	}

	theResp := &firs_models.FirsResponse{}

	return utility.GetQueryPullRequest(utility.DefaultHTTPClient, config, theResp, query)
}

func DebugHealthCheck() (*utility.Response, error) {

	var (
		configs = config.GetConfig()
		apiURL  = fmt.Sprintf("%v/invoice/transmit/self-health-check", configs.Firs.FirsApiUrl)
	)

	config := utility.RequestConfig{
		URL: apiURL,
		Headers: map[string]string{
			"x-api-key":    configs.Firs.FirsApiKey,
			"x-api-secret": configs.Firs.FirsClientKey,
		},
		Body: nil,
	}

	theResp := &firs_models.FirsResponse{}

	return utility.GetRequest(utility.DefaultHTTPClient, config, theResp)
}
