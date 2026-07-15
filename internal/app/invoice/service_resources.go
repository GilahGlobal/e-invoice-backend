package invoice

import (
	"einvoice-access-point/internal/pkg/firs"
	"einvoice-access-point/internal/pkg/firs_models"
	"fmt"
)

func (s *Service) GetInvoiceTypes() (*firs_models.FirsResponse, *string, error) {
	resp, err := firs.GetInvoiceTypes()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get invoice types: %w", err)
	}

	theResp, errDetails, err := firs.ParseFIRSAPIResponse(resp)
	if err != nil {
		return nil, errDetails, fmt.Errorf("failed to parse FIRS API response: %w", err)
	}

	return theResp, nil, nil
}

func (s *Service) GetPaymentMeans() (*firs_models.FirsResponse, *string, error) {
	resp, err := firs.GetPaymentMeans()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get payment means: %w", err)
	}

	theResp, errDetails, err := firs.ParseFIRSAPIResponse(resp)
	if err != nil {
		return nil, errDetails, fmt.Errorf("failed to parse FIRS API response: %w", err)
	}

	return theResp, nil, nil
}

func (s *Service) GetTaxCategories() (*firs_models.FirsResponse, *string, error) {
	resp, err := firs.GetTaxCategories()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get tax categories: %w", err)
	}

	theResp, errDetails, err := firs.ParseFIRSAPIResponse(resp)
	if err != nil {
		return nil, errDetails, fmt.Errorf("failed to parse FIRS API response: %w", err)
	}

	return theResp, nil, nil
}

func (s *Service) GetProductCodes() (*firs_models.FirsResponse, *string, error) {
	resp, err := firs.GetProductCodes()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get product codes: %w", err)
	}

	theResp, errDetails, err := firs.ParseFIRSAPIResponse(resp)
	if err != nil {
		return nil, errDetails, fmt.Errorf("failed to parse FIRS API response: %w", err)
	}

	return theResp, nil, nil
}

func (s *Service) GetServiceCodes() (*firs_models.FirsResponse, *string, error) {
	resp, err := firs.GetServiceCodes()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get service codes: %w", err)
	}

	theResp, errDetails, err := firs.ParseFIRSAPIResponse(resp)
	if err != nil {
		return nil, errDetails, fmt.Errorf("failed to parse FIRS API response: %w", err)
	}

	return theResp, nil, nil
}

func (s *Service) GetCurrencies() (*firs_models.FirsResponse, *string, error) {
	resp, err := firs.GetCurrencies()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get currencies: %w", err)
	}

	theResp, errDetails, err := firs.ParseFIRSAPIResponse(resp)
	if err != nil {
		return nil, errDetails, fmt.Errorf("failed to parse FIRS API response: %w", err)
	}

	return theResp, nil, nil
}

func (s *Service) GetLGA() (*firs_models.FirsResponse, *string, error) {
	resp, err := firs.GetLGA()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get LGAs: %w", err)
	}

	theResp, errDetails, err := firs.ParseFIRSAPIResponse(resp)
	if err != nil {
		return nil, errDetails, fmt.Errorf("failed to parse FIRS API response: %w", err)
	}

	return theResp, nil, nil
}

func (s *Service) GetCountries() (*firs_models.FirsResponse, *string, error) {
	resp, err := firs.GetCountries()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get countries: %w", err)
	}

	theResp, errDetails, err := firs.ParseFIRSAPIResponse(resp)
	if err != nil {
		return nil, errDetails, fmt.Errorf("failed to parse FIRS API response: %w", err)
	}

	return theResp, nil, nil
}

func (s *Service) GetStates() (*firs_models.FirsResponse, *string, error) {
	resp, err := firs.GetStates()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get states: %w", err)
	}

	theResp, errDetails, err := firs.ParseFIRSAPIResponse(resp)
	if err != nil {
		return nil, errDetails, fmt.Errorf("failed to parse FIRS API response: %w", err)
	}

	return theResp, nil, nil
}
