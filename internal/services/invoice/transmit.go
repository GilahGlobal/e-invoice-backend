package invoice

import (
	"einvoice-access-point/external/firs"
	"einvoice-access-point/external/firs_models"
	"einvoice-access-point/pkg/models"
	"fmt"
)

func LookUpIRN(irn string) (*firs_models.FirsResponse, *string, error) {

	resp, err := firs.LookUpByIRN(irn)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get invoice: %w", err)
	}

	theResp, errDetails, err := firs.ParseFIRSAPIResponse(resp)
	if err != nil {
		return nil, errDetails, fmt.Errorf("failed to parse FIRS API response: %w", err)
	}

	//fmt.Println("Invoice gotten successful: ", theResp)
	return theResp, nil, nil
}

func LookUpTIN(tin string) (*firs_models.FirsResponse, *string, error) {

	resp, err := firs.LookUpByTIN(tin)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get invoices with TIN: %w", err)
	}

	theResp, errDetails, err := firs.ParseFIRSAPIResponse(resp)
	if err != nil {
		return nil, errDetails, fmt.Errorf("failed to parse FIRS API response: %w", err)
	}

	//fmt.Println("Invoice gotten successful: ", theResp)
	return theResp, nil, nil
}

func LookUpPartyID(partyId string) (*firs_models.FirsResponse, *string, error) {

	resp, err := firs.LookUpByPartyID(partyId)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get invoices with PartyID: %w", err)
	}

	theResp, errDetails, err := firs.ParseFIRSAPIResponse(resp)
	if err != nil {
		return nil, errDetails, fmt.Errorf("failed to parse FIRS API response: %w", err)
	}

	//fmt.Println("Invoice gotten successful: ", theResp)
	return theResp, nil, nil
}

func TransmitInvoice(irn string, isSandbox bool) (*firs_models.FirsResponse, *string, error) {

	resp, err := firs.TransmitInvoice(irn, isSandbox)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to transmit invoice: %w", err)
	}

	theResp, errDetails, err := firs.ParseFIRSAPIResponse(resp)
	if err != nil {
		return nil, errDetails, fmt.Errorf("failed to parse FIRS API response: %w", err)
	}

	//fmt.Println("Invoice transmitted successful: ", theResp)
	return theResp, nil, nil
}

func TransmitConfirmInvoice(irn string, isSandbox bool) (*firs_models.FirsResponse, *string, error) {

	resp, err := firs.TransmitConfirmInvoice(irn, isSandbox)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to confirm transamitted invoicee: %w", err)
	}

	theResp, errDetails, err := firs.ParseFIRSAPIResponse(resp)
	if err != nil {
		return nil, errDetails, fmt.Errorf("failed to parse FIRS API response: %w", err)
	}

	//fmt.Println("Invoice confirm transmitted successful: ", theResp)
	return theResp, nil, nil
}

func TransmitPull(query models.PullDataQuery) (*firs_models.FirsResponse, *string, error) {

	resp, err := firs.TransmitPull(query)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get transmit pull: %w", err)
	}

	theResp, errDetails, err := firs.ParseFIRSAPIResponse(resp)
	if err != nil {
		return nil, errDetails, fmt.Errorf("failed to parse FIRS API response: %w", err)
	}

	return theResp, nil, nil

}

func DebugHealthCheck() (*firs_models.FirsResponse, *string, error) {

	resp, err := firs.DebugHealthCheck()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get health status: %w", err)
	}

	theResp, errDetails, err := firs.ParseFIRSAPIResponse(resp)
	if err != nil {
		return nil, errDetails, fmt.Errorf("failed to parse FIRS API response: %w", err)
	}

	//fmt.Println("Invoice gotten successful: ", theResp)
	return theResp, nil, nil
}
