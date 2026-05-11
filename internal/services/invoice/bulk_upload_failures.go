package invoice

import (
	"einvoice-access-point/internal/dtos"
	repository "einvoice-access-point/internal/repository/invoice"
	inst "einvoice-access-point/pkg/dbinit"
	"einvoice-access-point/pkg/models"
	"encoding/json"
	"fmt"
	"strings"

	"gorm.io/gorm"
)

func GetBulkUploadFailedInvoices(db *gorm.DB, bulkUploadID, businessID string) (*dtos.BulkUploadFailedInvoicesDto, error) {
	pdb := inst.InitDB(db, false)

	bulkUpload, err := repository.GetBulkUploadLogByID(pdb, bulkUploadID, businessID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve bulk upload log: %w", err)
	}
	if bulkUpload == nil {
		return nil, fmt.Errorf("bulk upload log not found")
	}

	return BuildBulkUploadFailedInvoicesResponse(bulkUpload)
}

func BuildBulkUploadFailedInvoicesResponse(bulkUpload *models.BulkUpload) (*dtos.BulkUploadFailedInvoicesDto, error) {
	failedInvoices, err := parseBulkUploadFailedInvoices(json.RawMessage(bulkUpload.ValidationErrors))
	if err != nil {
		return nil, err
	}

	return &dtos.BulkUploadFailedInvoicesDto{
		BulkUploadID:                bulkUpload.ID,
		FileURL:                     bulkUpload.FileURL,
		FileKey:                     bulkUpload.FileKey,
		BusinessID:                  bulkUpload.BusinessID,
		AggregatorID:                bulkUpload.AggregatorID,
		Status:                      bulkUpload.Status,
		TotalRecords:                bulkUpload.TotalRecords,
		ValidRecords:                bulkUpload.ValidRecords,
		SuccessfulInvoices:          bulkUpload.SuccessfulInvoices,
		PartiallySuccessfulInvoices: bulkUpload.PartiallySuccessfulInvoices,
		UnsuccessfulInvoices:        bulkUpload.UnsuccessfulInvoices,
		FailedInvoicesCount:         len(failedInvoices),
		FailedInvoices:              failedInvoices,
		CreatedAt:                   bulkUpload.CreatedAt,
		StartedAt:                   bulkUpload.StartedAt,
		CompletedAt:                 bulkUpload.CompletedAt,
	}, nil
}

func parseBulkUploadFailedInvoices(raw json.RawMessage) ([]dtos.BulkUploadFailedInvoiceDto, error) {
	if len(raw) == 0 {
		return []dtos.BulkUploadFailedInvoiceDto{}, nil
	}

	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" || trimmed == "null" {
		return []dtos.BulkUploadFailedInvoiceDto{}, nil
	}

	var failedInvoices []dtos.BulkUploadFailedInvoiceDto
	if err := json.Unmarshal(raw, &failedInvoices); err != nil {
		return nil, fmt.Errorf("failed to parse bulk upload validation errors: %w", err)
	}

	if failedInvoices == nil {
		return []dtos.BulkUploadFailedInvoiceDto{}, nil
	}

	return failedInvoices, nil
}
