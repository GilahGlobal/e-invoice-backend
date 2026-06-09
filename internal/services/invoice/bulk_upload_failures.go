package invoice

import (
	"bytes"
	"einvoice-access-point/internal/dtos"
	repository "einvoice-access-point/internal/repository/invoice"
	inst "einvoice-access-point/pkg/dbinit"
	"einvoice-access-point/pkg/models"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/xuri/excelize/v2"
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

	return BuildBulkUploadFailedInvoicesResponse(db, bulkUpload)
}

func BuildBulkUploadFailedInvoicesResponse(db *gorm.DB, bulkUpload *models.BulkUpload) (*dtos.BulkUploadFailedInvoicesDto, error) {
	failedInvoices, err := parseBulkUploadFailedInvoices(json.RawMessage(bulkUpload.ValidationErrors))
	if err != nil {
		return nil, err
	}
	failedInvoices, err = enrichBulkUploadFailedInvoices(db, bulkUpload.BusinessID, failedInvoices)
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

func enrichBulkUploadFailedInvoices(db *gorm.DB, businessID string, failedInvoices []dtos.BulkUploadFailedInvoiceDto) ([]dtos.BulkUploadFailedInvoiceDto, error) {
	pdb := inst.InitDB(db, false)

	for i := range failedInvoices {
		reason := normalizeBulkUploadFailureReason(failedInvoices[i].Error)
		failedInvoices[i].Reason = reason

		if failedInvoices[i].Stage != "" {
			continue
		}

		if failedInvoices[i].InvoiceNumber != "" {
			invoiceRecord, err := repository.FindInvoiceByNumberAndBusinessID(pdb, failedInvoices[i].InvoiceNumber, businessID)
			if err != nil {
				return nil, fmt.Errorf("failed to retrieve invoice for failed bulk upload entry: %w", err)
			}
			if invoiceRecord != nil && invoiceRecord.CurrentStatus != "" {
				failedInvoices[i].Stage = invoiceRecord.CurrentStatus
				continue
			}
		}

		failedInvoices[i].Stage = inferBulkUploadFailureStage(reason)
	}

	return failedInvoices, nil
}

func normalizeBulkUploadFailureReason(reason any) string {
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

func inferBulkUploadFailureStage(reason string) string {
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
		return models.StatusGeneratedIRN
	case strings.Contains(reason, "failed to validate invoice"):
		return models.StatusValidatedInvoice
	case strings.Contains(reason, "failed to sign invoice"):
		return models.StatusSignedInvoice
	case strings.Contains(reason, "failed to transmit invoice"):
		return models.StatusTransmitted
	case strings.Contains(reason, "failed to confirm transmit invoice"), strings.Contains(reason, "failed to confirm invoice"):
		return models.StatusConfirmed
	default:
		return "unknown"
	}
}

func BuildBulkUploadFailedInvoiceExportRows(failedInvoices *dtos.BulkUploadFailedInvoicesDto) []dtos.BulkUploadFailedInvoiceExportRowDto {
	rows := make([]dtos.BulkUploadFailedInvoiceExportRowDto, 0, len(failedInvoices.FailedInvoices))
	for _, failedInvoice := range failedInvoices.FailedInvoices {
		rows = append(rows, dtos.BulkUploadFailedInvoiceExportRowDto{
			InvoiceNumber: failedInvoice.InvoiceNumber,
			Stage:         failedInvoice.Stage,
			Reason:        failedInvoice.Reason,
		})
	}
	return rows
}

func ExportBulkUploadFailedInvoicesCSV(failedInvoices *dtos.BulkUploadFailedInvoicesDto) ([]byte, error) {
	buffer := &bytes.Buffer{}
	writer := csv.NewWriter(buffer)

	if err := writer.Write([]string{"invoice_number", "stage", "reason"}); err != nil {
		return nil, fmt.Errorf("failed to write csv header: %w", err)
	}

	for _, row := range BuildBulkUploadFailedInvoiceExportRows(failedInvoices) {
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

func ExportBulkUploadFailedInvoicesExcel(failedInvoices *dtos.BulkUploadFailedInvoicesDto) ([]byte, error) {
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

	for rowIndex, row := range BuildBulkUploadFailedInvoiceExportRows(failedInvoices) {
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

func ExportBulkUploadFailedInvoices(failedInvoices *dtos.BulkUploadFailedInvoicesDto, format string) ([]byte, string, string, error) {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "", "csv":
		data, err := ExportBulkUploadFailedInvoicesCSV(failedInvoices)
		if err != nil {
			return nil, "", "", err
		}
		return data, "text/csv; charset=utf-8", "csv", nil
	case "excel", "xlsx":
		data, err := ExportBulkUploadFailedInvoicesExcel(failedInvoices)
		if err != nil {
			return nil, "", "", err
		}
		return data, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", "xlsx", nil
	default:
		return nil, "", "", fmt.Errorf("unsupported export format: %s", format)
	}
}
