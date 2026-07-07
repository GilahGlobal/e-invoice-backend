package health

import (
	"einvoice-access-point/internal/pkg/firs"
	"einvoice-access-point/internal/pkg/firs_models"
	"fmt"
)

type Service struct{}

func NewService() *Service {
	return &Service{}
}

func (s *Service) ReturnTrue() bool {
	return true
}

func (s *Service) FirsApiHealthCheck() (*firs_models.FirsResponse, *string, error) {

	resp, err := firs.ApiHealthCheck()
	if err != nil {
		fmt.Println("Error checking FIRS API health:", err)
		return nil, nil, fmt.Errorf("failed to check firs api health: %w", err)
	}

	theResp, errDetails, err := firs.ParseFIRSAPIResponse(resp)
	if err != nil {
		return nil, errDetails, fmt.Errorf("failed to parse FIRS API response: %w", err)
	}

	return theResp, nil, nil
}
