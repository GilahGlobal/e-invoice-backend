package bulk_upload

import (
	"bytes"
	"einvoice-access-point/internal/data/database"
	"einvoice-access-point/internal/data/dbinit"
	"einvoice-access-point/internal/data/entities"
	"einvoice-access-point/internal/data/repositories"
	"einvoice-access-point/internal/utility"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type Service struct {
	repo repositories.BulkUploadRepository
}

func NewService(repo repositories.BulkUploadRepository) *Service {
	return &Service{repo: repo}
}

func NewServiceWithDB(db, testDB database.DatabaseManager) *Service {
	repo := repositories.NewBulkUploadRepository(db, testDB)
	return NewService(repo)
}

func (s *Service) GetBulkUploadFailedInvoices(db *gorm.DB, bulkUploadID, businessID string) (*BulkUploadFailedInvoicesDto, error) {
	pdb := dbinit.InitDB(db, false)

	bulkUpload, err := s.repo.GetBulkUploadLogByID(pdb, bulkUploadID, businessID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve bulk upload log: %w", err)
	}
	if bulkUpload == nil {
		return nil, fmt.Errorf("bulk upload log not found")
	}

	return s.BuildBulkUploadFailedInvoicesResponse(db, bulkUpload)
}

func (s *Service) BuildBulkUploadFailedInvoicesResponse(db *gorm.DB, bulkUpload *entities.BulkUpload) (*BulkUploadFailedInvoicesDto, error) {
	failedInvoices, err := s.parseBulkUploadFailedInvoices(json.RawMessage(bulkUpload.ValidationErrors))
	if err != nil {
		return nil, err
	}
	failedInvoices, err = s.enrichBulkUploadFailedInvoices(db, bulkUpload.BusinessID, failedInvoices)
	if err != nil {
		return nil, err
	}

	return &BulkUploadFailedInvoicesDto{
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

func (s *Service) parseBulkUploadFailedInvoices(raw json.RawMessage) ([]BulkUploadFailedInvoiceDto, error) {
	if len(raw) == 0 {
		return []BulkUploadFailedInvoiceDto{}, nil
	}

	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" || trimmed == "null" {
		return []BulkUploadFailedInvoiceDto{}, nil
	}

	var failedInvoices []BulkUploadFailedInvoiceDto
	if err := json.Unmarshal(raw, &failedInvoices); err != nil {
		return nil, fmt.Errorf("failed to parse bulk upload validation errors: %w", err)
	}

	if failedInvoices == nil {
		return []BulkUploadFailedInvoiceDto{}, nil
	}

	return failedInvoices, nil
}

func (s *Service) enrichBulkUploadFailedInvoices(db *gorm.DB, businessID string, failedInvoices []BulkUploadFailedInvoiceDto) ([]BulkUploadFailedInvoiceDto, error) {
	pdb := dbinit.InitDB(db, false)

	for i := range failedInvoices {
		reason := s.normalizeBulkUploadFailureReason(failedInvoices[i].Error)
		failedInvoices[i].Reason = reason

		if failedInvoices[i].Stage != "" {
			continue
		}

		if failedInvoices[i].InvoiceNumber != "" {
			invoiceRecord, err := s.repo.FindInvoiceByNumberAndBusinessID(pdb, failedInvoices[i].InvoiceNumber, businessID)
			if err != nil {
				return nil, fmt.Errorf("failed to retrieve invoice for failed bulk upload entry: %w", err)
			}
			if invoiceRecord != nil && invoiceRecord.CurrentStatus != "" {
				failedInvoices[i].Stage = invoiceRecord.CurrentStatus
				continue
			}
		}

		failedInvoices[i].Stage = s.inferBulkUploadFailureStage(reason)
	}

	return failedInvoices, nil
}

func (s *Service) normalizeBulkUploadFailureReason(reason any) string {
	switch v := reason.(type) {
	case string:
		return strings.TrimSpace(v)
	case map[string]any:
		if marshaled, err := json.Marshal(v); err == nil {
			return string(marshaled)
		}
	case []any:
		if marshaled, err := json.Marshal(v); err == nil {
			return string(marshaled)
		}
	}

	if reason == nil {
		return ""
	}

	return fmt.Sprintf("%v", reason)
}

func (s *Service) inferBulkUploadFailureStage(reason string) string {
	reason = strings.ToLower(strings.TrimSpace(reason))

	switch {
	case reason == "":
		return "unknown"
	case strings.Contains(reason, "parse errors"), strings.Contains(reason, "validation failed"):
		return "validation"
	case strings.Contains(reason, "duplicate"):
		return "duplicate_check"
	case strings.Contains(reason, "subscription"):
		return "subscription_check"
	case strings.Contains(reason, "database"):
		return "database"
	case strings.Contains(reason, "irn generation"):
		return entities.StatusGeneratedIRN
	case strings.Contains(reason, "failed to validate invoice"):
		return entities.StatusValidatedInvoice
	case strings.Contains(reason, "failed to sign invoice"):
		return entities.StatusSignedInvoice
	case strings.Contains(reason, "failed to transmit invoice"):
		return entities.StatusTransmitted
	case strings.Contains(reason, "failed to confirm transmit invoice"), strings.Contains(reason, "failed to confirm invoice"):
		return entities.StatusConfirmed
	default:
		return "unknown"
	}
}

func (s *Service) BuildBulkUploadFailedInvoiceExportRows(failedInvoices *BulkUploadFailedInvoicesDto) []BulkUploadFailedInvoiceExportRowDto {
	rows := make([]BulkUploadFailedInvoiceExportRowDto, 0, len(failedInvoices.FailedInvoices))
	for _, failedInvoice := range failedInvoices.FailedInvoices {
		rows = append(rows, BulkUploadFailedInvoiceExportRowDto{
			InvoiceNumber: failedInvoice.InvoiceNumber,
			Stage:         failedInvoice.Stage,
			Reason:        failedInvoice.Reason,
		})
	}
	return rows
}

func (s *Service) ExportBulkUploadFailedInvoicesCSV(failedInvoices *BulkUploadFailedInvoicesDto) ([]byte, error) {
	buffer := &bytes.Buffer{}
	writer := csv.NewWriter(buffer)

	if err := writer.Write([]string{"invoice_number", "stage", "reason"}); err != nil {
		return nil, fmt.Errorf("failed to write csv header: %w", err)
	}

	for _, row := range s.BuildBulkUploadFailedInvoiceExportRows(failedInvoices) {
		if err := writer.Write([]string{row.InvoiceNumber, row.Stage, row.Reason}); err != nil {
			return nil, fmt.Errorf("failed to write csv row: %w", err)
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, fmt.Errorf("failed to finalize csv: %w", err)
	}

	return buffer.Bytes(), nil
}

func (s *Service) ExportBulkUploadFailedInvoicesExcel(failedInvoices *BulkUploadFailedInvoicesDto) ([]byte, error) {
	file := excelize.NewFile()
	sheetName := "failed_invoices"
	file.SetSheetName(file.GetSheetName(0), sheetName)

	headers := []string{"invoice_number", "stage", "reason"}
	for i, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		if err := file.SetCellValue(sheetName, cell, header); err != nil {
			return nil, fmt.Errorf("failed to set excel header: %w", err)
		}
	}

	for rowIndex, row := range s.BuildBulkUploadFailedInvoiceExportRows(failedInvoices) {
		values := []string{row.InvoiceNumber, row.Stage, row.Reason}
		for colIndex, value := range values {
			cell, _ := excelize.CoordinatesToCellName(colIndex+1, rowIndex+2)
			if err := file.SetCellValue(sheetName, cell, value); err != nil {
				return nil, fmt.Errorf("failed to set excel cell value: %w", err)
			}
		}
	}

	buffer, err := file.WriteToBuffer()
	if err != nil {
		return nil, fmt.Errorf("failed to generate excel file: %w", err)
	}

	return buffer.Bytes(), nil
}

func (s *Service) ExportBulkUploadFailedInvoices(failedInvoices *BulkUploadFailedInvoicesDto, format string) ([]byte, string, string, error) {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "", "csv":
		data, err := s.ExportBulkUploadFailedInvoicesCSV(failedInvoices)
		if err != nil {
			return nil, "", "", err
		}
		return data, "text/csv; charset=utf-8", "csv", nil
	case "excel", "xlsx":
		data, err := s.ExportBulkUploadFailedInvoicesExcel(failedInvoices)
		if err != nil {
			return nil, "", "", err
		}
		return data, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", "xlsx", nil
	default:
		return nil, "", "", fmt.Errorf("unsupported export format: %s", format)
	}
}

func (s *Service) AddBulkUploadLog(db *gorm.DB, fileUrl, fileKey, businessID string, aggregatorID *string) (string, error) {
	pdb := dbinit.InitDB(db, false)

	payload := &entities.BulkUpload{
		ID:           utility.GenerateUUID(),
		FileURL:      fileUrl,
		FileKey:      fileKey,
		BusinessID:   businessID,
		AggregatorID: aggregatorID,
	}

	if err := s.repo.CreateBulkUploadLog(pdb, payload); err != nil {
		errDetails := "failed to save bulk upload log"
		return "", fmt.Errorf("%s: %w", errDetails, err)
	}
	return payload.ID, nil
}

func (s *Service) UpdateBulkUploadLog(db *gorm.DB, bulkID, fileKey, businessID string, payload interface{}) error {
	pdb := dbinit.InitDB(db, false)

	repositoryLog, err := s.repo.GetBulkUploadLogByID(pdb, bulkID, businessID)
	if err != nil {
		errDetails := "failed to retrieve bulk upload log"
		return fmt.Errorf("%s: %w", errDetails, err)
	}
	if repositoryLog == nil {
		errDetails := "bulk upload log not found"
		return fmt.Errorf("%s for file key: %s", errDetails, fileKey)
	}

	data, ok := payload.(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid payload type")
	}

	repositoryLog.TotalRecords = data["TotalRows"].(int)
	repositoryLog.ValidRecords = data["ValidRows"].(int)
	repositoryLog.SuccessfulInvoices = data["SuccessfulInvoices"].(int)
	repositoryLog.PartiallySuccessfulInvoices = data["PartiallySuccessfulInvoices"].(int)
	repositoryLog.UnsuccessfulInvoices = data["UnsuccessfulInvoices"].(int)
	repositoryLog.Duration = data["Duration"].(time.Duration)
	repositoryLog.StartedAt = data["StartTime"].(*time.Time)
	repositoryLog.CompletedAt = data["EndTime"].(*time.Time)
	repositoryLog.Status = "completed"

	if err := s.repo.UpdateBulkUploadLog(pdb, bulkID, businessID, repositoryLog); err != nil {

		errDetails := "failed to update bulk upload log"
		return fmt.Errorf("%s: %w", errDetails, err)
	}
	return nil
}

func (s *Service) StoreBulkUploadValidationErrors(db *gorm.DB, bulkID, fileKey, businessID string, validationErrorsJSON []byte, errorCount int) error {
	pdb := dbinit.InitDB(db, false)

	repositoryLog, err := s.repo.GetBulkUploadLogByID(pdb, bulkID, businessID)
	if err != nil {
		errDetails := "failed to retrieve bulk upload log"
		return fmt.Errorf("%s: %w", errDetails, err)
	}
	if repositoryLog == nil {
		errDetails := "bulk upload log not found"
		return fmt.Errorf("%s for file key: %s", errDetails, fileKey)
	}

	repositoryLog.ValidationErrors = datatypes.JSON(validationErrorsJSON)
	repositoryLog.ValidationErrorCount = errorCount

	if err := s.repo.UpdateBulkUploadLog(pdb, bulkID, businessID, repositoryLog); err != nil {
		errDetails := "failed to store bulk upload validation errors"
		return fmt.Errorf("%s: %w", errDetails, err)
	}

	return nil
}

func (s *Service) GetBulkUploadLogByID(db *gorm.DB, id, businessID string) (*entities.BulkUpload, error) {
	pdb := dbinit.InitDB(db, false)
	return s.repo.GetBulkUploadLogByID(pdb, id, businessID)
}

func (s *Service) GetBulkUploadLogsByBusinessID(db *gorm.DB, businessID string, page, size int) ([]entities.BulkUpload, database.PaginationResponse, error) {
	pdb := dbinit.InitDB(db, false)

	pagination := database.Pagination{
		Page:  page,
		Limit: size,
	}

	return s.repo.FindBulkUploadLogsByBusinessID(pdb, businessID, pagination)
}
