package firs

import (
	"einvoice-access-point/external/firs_models"
	"einvoice-access-point/internal/dtos"
	"einvoice-access-point/pkg/config"
	"einvoice-access-point/pkg/utility"
	"fmt"
	"strings"
)

func Login(email, password string) (*utility.Response, error) {

	var (
		configs = config.GetConfig()
		apiURL  = fmt.Sprintf("%v/utilities/authenticate", configs.Firs.FirsApiUrl)
	)

	config := utility.RequestConfig{
		URL: apiURL,
		Headers: map[string]string{
			"x-api-key":    configs.Firs.FirsApiKey,
			"x-api-secret": configs.Firs.FirsClientKey,
		},
		Body: dtos.LoginRequestDto{
			Email:    email,
			Password: password,
		},
	}

	theResp := firs_models.FirsResponse{}

	return utility.PostRequest(utility.DefaultHTTPClient, config, theResp)

}

func VerifyTin(tin string) (*utility.Response, error) {

	var (
		configs = config.GetConfig()
		apiURL  = fmt.Sprintf("%v/utilities/verify-tin/", configs.Firs.FirsApiUrl)
	)

	config := utility.RequestConfig{
		URL: apiURL,
		Headers: map[string]string{
			"x-api-key":    configs.Firs.FirsApiKey,
			"x-api-secret": configs.Firs.FirsClientKey,
		},
		Body: firs_models.VerifyTinData{
			TIN: tin,
		},
	}

	theResp := &firs_models.FirsResponse{}

	return utility.PostRequest(utility.DefaultHTTPClient, config, theResp)

}

func ApiHealthCheck() (*utility.Response, error) {

	var (
		configs = config.GetConfig()
	)

	apiURL := configs.Firs.FirsApiUrl[:strings.Index(configs.Firs.FirsApiUrl, "/v1")]

	config := utility.RequestConfig{
		URL: apiURL,
		Headers: map[string]string{
			"x-api-key":    configs.Firs.FirsApiKey,
			"x-api-secret": configs.Firs.FirsClientKey,
		},
		Body: nil,
	}

	theResp := &firs_models.HealthCheck{}

	return utility.GetRequest(utility.DefaultHTTPClient, config, theResp)

}
