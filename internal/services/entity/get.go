package entity

import (
	"einvoice-access-point/external/firs"
	"einvoice-access-point/external/firs_models"
	"einvoice-access-point/pkg/models"
	"fmt"
)

func FetchQueryItems(query models.PaginationQuery) models.PaginationQuery {
	if query.Size <= 0 {
		query.Size = 20
	}
	if query.Page <= 0 {
		query.Page = 1
	}

	return query
}

func GetEntities(query models.PaginationQuery, isSandbox bool) (*firs_models.FirsResponse, *string, error) {

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

func GetEntity(entityId string, isSandbox bool) (*firs_models.FirsResponse, *string, error) {

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
