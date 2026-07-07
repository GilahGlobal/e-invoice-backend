package entity

import (
	"einvoice-access-point/internal/data/entities"
	"einvoice-access-point/internal/pkg/firs"
	"einvoice-access-point/internal/pkg/firs_models"
	"fmt"
)

type Service struct{}

func NewService() *Service {
	return &Service{}
}

func (s *Service) FetchQueryItems(query entities.PaginationQuery) entities.PaginationQuery {
	if query.Size <= 0 {
		query.Size = 20
	}
	if query.Page <= 0 {
		query.Page = 1
	}

	return query
}

func (s *Service) GetEntities(query entities.PaginationQuery, isSandbox bool) (*firs_models.FirsResponse, *string, error) {

	resp, err := firs.GetEntities(query, isSandbox)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get entities: %w", err)
	}

	theResp, errDetails, err := firs.ParseFIRSAPIResponse(resp)
	if err != nil {
		return nil, errDetails, fmt.Errorf("failed to parse FIRS API response: %w", err)
	}

	return theResp, nil, nil
}

func (s *Service) GetEntity(entityId string, isSandbox bool) (*firs_models.FirsResponse, *string, error) {

	resp, err := firs.GetEntity(entityId, isSandbox)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get entity: %w", err)
	}

	theResp, errDetails, err := firs.ParseFIRSAPIResponse(resp)
	if err != nil {
		return nil, errDetails, fmt.Errorf("failed to parse FIRS API response: %w", err)
	}

	return theResp, nil, nil
}

func (s *Service) VerifyTin(tin string) (*firs_models.FirsResponse, *string, error) {

	resp, err := firs.VerifyTin(tin)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to verify tin: %w", err)
	}

	theResp, errDetails, err := firs.ParseFIRSAPIResponse(resp)
	if err != nil {
		return nil, errDetails, fmt.Errorf("failed to parse FIRS API response: %w", err)
	}

	return theResp, nil, nil
}

func (s *Service) PostVatPayment(req firs_models.FirsTransactionVatPayload) (*firs_models.FirsResponse, *string, error) {

	resp, err := firs.PostVatPayment(req)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to post payment: %w", err)
	}

	theResp, errDetails, err := firs.ParseFIRSAPIResponse(resp)
	if err != nil {
		return nil, errDetails, fmt.Errorf("failed to parse FIRS API response: %w", err)
	}

	return theResp, nil, nil
}

func (s *Service) CreateParty(req firs_models.PartyRegistrationPayload) (*firs_models.FirsResponse, *string, error) {

	resp, err := firs.CreateParty(req)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create party: %w", err)
	}

	theResp, errDetails, err := firs.ParseFIRSAPIResponse(resp)
	if err != nil {
		return nil, errDetails, fmt.Errorf("failed to parse FIRS API response: %w", err)
	}

	return theResp, nil, nil
}
