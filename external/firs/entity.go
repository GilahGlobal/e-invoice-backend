package firs

import (
	"einvoice-access-point/external/firs_models"
	"einvoice-access-point/pkg/config"
	"einvoice-access-point/pkg/models"
	"einvoice-access-point/pkg/utility"
	"fmt"
)

func GetEntities(query models.PaginationQuery, isSandbox bool) (*utility.Response, error) {

	var (
		configs = config.GetConfig()
	)

	var config utility.RequestConfig
	var apiURL string
	if isSandbox {
		apiURL = fmt.Sprintf("%v/entity", configs.FirsSandbox.FirsApiUrl)
		config = utility.RequestConfig{
			URL: apiURL,
			Headers: map[string]string{
				"x-api-key":    configs.FirsSandbox.FirsApiKey,
				"x-api-secret": configs.FirsSandbox.FirsClientKey,
			},
			Body: nil,
		}

	} else {
		apiURL = fmt.Sprintf("%v/entity", configs.Firs.FirsApiUrl)
		config = utility.RequestConfig{
			URL: apiURL,
			Headers: map[string]string{
				"x-api-key":    configs.Firs.FirsApiKey,
				"x-api-secret": configs.Firs.FirsClientKey,
			},
			Body: nil,
		}

	}

	theResp := &firs_models.FirsResponse{}

	return utility.GetQueryRequest(utility.DefaultHTTPClient, config, theResp, query)
}

func GetEntity(entityId string, isSandbox bool) (*utility.Response, error) {

	var (
		configs = config.GetConfig()
	)

	var config utility.RequestConfig
	var apiURL string
	if isSandbox {
		apiURL = fmt.Sprintf("%v/entity/%s", configs.FirsSandbox.FirsApiUrl, entityId)
		config = utility.RequestConfig{
			URL: apiURL,
			Headers: map[string]string{
				"x-api-key":    configs.FirsSandbox.FirsApiKey,
				"x-api-secret": configs.FirsSandbox.FirsClientKey,
			},
			Body: nil,
		}

	} else {
		apiURL = fmt.Sprintf("%v/entity/%s", configs.Firs.FirsApiUrl, entityId)
		config = utility.RequestConfig{
			URL: apiURL,
			Headers: map[string]string{
				"x-api-key":    configs.Firs.FirsApiKey,
				"x-api-secret": configs.Firs.FirsClientKey,
			},
			Body: nil,
		}

	}

	theResp := &firs_models.FirsResponse{}

	return utility.GetRequest(utility.DefaultHTTPClient, config, theResp)
}

func PostVatPayment(req firs_models.FirsTransactionVatPayload) (*utility.Response, error) {

	var (
		configs = config.GetConfig()
		apiURL  = fmt.Sprintf("%v/vat/postPayment", configs.Firs.FirsApiUrl)
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

func CreateParty(req firs_models.PartyRegistrationPayload) (*utility.Response, error) {

	var (
		configs = config.GetConfig()
		apiURL  = fmt.Sprintf("%v/invoice/party", configs.Firs.FirsApiUrl)
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
