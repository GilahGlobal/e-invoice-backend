package firs

import (
	"einvoice-access-point/external/firs_models"
	"einvoice-access-point/pkg/config"
	"einvoice-access-point/pkg/utility"
	"fmt"
)

func GetInvoiceTypes() (*utility.Response, error) {
	var (
		configs = config.GetConfig()
		apiURL  = fmt.Sprintf("%v/invoice/resources/invoice-types", configs.Firs.FirsApiUrl)
	)

	config := utility.RequestConfig{
		URL: apiURL,
	}

	theResp := &firs_models.FirsResponse{}

	return utility.GetRequest(utility.DefaultHTTPClient, config, theResp)
}

func GetPaymentMeans() (*utility.Response, error) {
	var (
		configs = config.GetConfig()
		apiURL  = fmt.Sprintf("%v/invoice/resources/payment_means", configs.Firs.FirsApiUrl)
	)

	config := utility.RequestConfig{
		URL: apiURL,
	}

	theResp := &firs_models.FirsResponse{}

	return utility.GetRequest(utility.DefaultHTTPClient, config, theResp)
}

func GetTaxCategories() (*utility.Response, error) {
	var (
		configs = config.GetConfig()
		apiURL  = fmt.Sprintf("%v/invoice/resources/tax-categories", configs.Firs.FirsApiUrl)
	)

	config := utility.RequestConfig{
		URL: apiURL,
	}

	theResp := &firs_models.FirsResponse{}

	return utility.GetRequest(utility.DefaultHTTPClient, config, theResp)
}

func GetProductCodes() (*utility.Response, error) {
	var (
		configs = config.GetConfig()
		apiURL  = fmt.Sprintf("%v/invoice/resources/hs-codes", configs.Firs.FirsApiUrl)
	)

	config := utility.RequestConfig{
		URL: apiURL,
	}

	theResp := &firs_models.FirsResponse{}

	return utility.GetRequest(utility.DefaultHTTPClient, config, theResp)
}

func GetServiceCodes() (*utility.Response, error) {
	var (
		configs = config.GetConfig()
		apiURL  = fmt.Sprintf("%v/invoice/resources/services-codes", configs.Firs.FirsApiUrl)
	)

	config := utility.RequestConfig{
		URL: apiURL,
	}

	theResp := &firs_models.FirsResponse{}

	return utility.GetRequest(utility.DefaultHTTPClient, config, theResp)
}

func GetCurrencies() (*utility.Response, error) {
	var (
		configs = config.GetConfig()
		apiURL  = fmt.Sprintf("%v/invoice/resources/currencies", configs.Firs.FirsApiUrl)
	)

	config := utility.RequestConfig{
		URL: apiURL,
	}

	theResp := &firs_models.FirsResponse{}

	return utility.GetRequest(utility.DefaultHTTPClient, config, theResp)
}

func GetLGA() (*utility.Response, error) {
	var (
		configs = config.GetConfig()
		apiURL  = fmt.Sprintf("%v/invoice/resources/lgas", configs.Firs.FirsApiUrl)
	)

	config := utility.RequestConfig{
		URL: apiURL,
	}

	theResp := &firs_models.FirsResponse{}

	return utility.GetRequest(utility.DefaultHTTPClient, config, theResp)
}

func GetCountries() (*utility.Response, error) {
	var (
		configs = config.GetConfig()
		apiURL  = fmt.Sprintf("%v/invoice/resources/countries", configs.Firs.FirsApiUrl)
	)

	config := utility.RequestConfig{
		URL: apiURL,
	}

	theResp := &firs_models.FirsResponse{}

	return utility.GetRequest(utility.DefaultHTTPClient, config, theResp)
}

func GetStates() (*utility.Response, error) {
	var (
		configs = config.GetConfig()
		apiURL  = fmt.Sprintf("%v/invoice/resources/states", configs.Firs.FirsApiUrl)
	)

	config := utility.RequestConfig{
		URL: apiURL,
	}

	theResp := &firs_models.FirsResponse{}

	return utility.GetRequest(utility.DefaultHTTPClient, config, theResp)
}
