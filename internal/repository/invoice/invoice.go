package invoice

import (
	"einvoice-access-point/pkg/database"
	"einvoice-access-point/pkg/models"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"strconv"
	"time"

	"gorm.io/gorm"
)

func GenerateUniqueInvoiceID(businessID string, db *gorm.DB) string {
	var lastInvoice models.Invoice
	var newInvoiceNumber string

	err := db.Where("business_id = ?", businessID).
		Order("invoice_number DESC").
		First(&lastInvoice).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			newInvoiceNumber = "INV00001"
		} else {
			log.Println("Error fetching last invoice:", err)
			return ""
		}
	} else {
		lastNumber, _ := strconv.Atoi(lastInvoice.InvoiceNumber[3:])
		newInvoiceNumber = fmt.Sprintf("INV%05d", lastNumber+1)
	}

	return newInvoiceNumber
}

func CreateInvoice(db database.DatabaseManager, invoice *models.Invoice) error {
	return db.DB().Create(invoice).Error
}

func FindInvoiceByNumber(db database.DatabaseManager, invoiceNumber string) (*models.Invoice, error) {
	var invoice models.Invoice
	err := db.DB().Where("invoice_number = ?", invoiceNumber).First(&invoice).Error
	if err != nil {
		return nil, err
	}
	return &invoice, nil
}

func FindInvoiceByNumberAndBusinessID(db database.DatabaseManager, invoiceNumber string, businessID string) (*models.Invoice, error) {
	var invoice models.Invoice
	err := db.DB().Where("invoice_number = ? AND business_id = ?", invoiceNumber, businessID).First(&invoice).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &invoice, nil
}

func UpdateInvoiceStatus(db database.DatabaseManager, invoice *models.Invoice, step string, status string) error {
	var history []models.StatusHistoryEntry

	if len(invoice.StatusHistory) > 0 {
		_ = json.Unmarshal(invoice.StatusHistory, &history)
	}

	for i := range history {
		if history[i].Step == step {
			history[i].Status = status
			history[i].Timestamp = time.Now()
			break
		}
	}

	historyJSON, _ := json.Marshal(history)
	invoice.StatusHistory = historyJSON
	invoice.CurrentStatus = step

	return db.DB().Save(invoice).Error
}

func UpdateInvoiceIRN(db database.DatabaseManager, invoice *models.Invoice, irn string) error {
	invoice.IRN = irn
	return db.DB().Save(invoice).Error
}

func FindMinimalInvoicesByBusinessID(db database.DatabaseManager, businessID string, pagination database.Pagination) ([]models.MinimalInvoiceDTO, database.PaginationResponse, error) {
	var result []models.MinimalInvoiceDTO

	if pagination.Page <= 0 {
		pagination.Page = 1
	}
	if pagination.Limit <= 0 {
		pagination.Limit = 20
	}

	var totalCount int64
	if err := db.DB().Model(&models.Invoice{}).Where("business_id = ? AND deleted_at IS NULL", businessID).Count(&totalCount).Error; err != nil {
		return nil, database.PaginationResponse{
			CurrentPage:     pagination.Page,
			PageCount:       0,
			TotalPagesCount: 0,
		}, err
	}

	totalPages := int(math.Ceil(float64(totalCount) / float64(pagination.Limit)))
	offset := (pagination.Page - 1) * pagination.Limit

	query := `
	SELECT 
		id,
		invoice_number,
		irn,
		platform,
		current_status,
		CASE
			WHEN current_status IN ('signed_invoice', 'transmitted_invoice')
				THEN 'partial_success'
			ELSE (
				SELECT COALESCE(entry->>'status', 'pending')
				FROM jsonb_array_elements(status_history) AS entry
				WHERE entry->>'step' = invoices.current_status
				ORDER BY entry->>'timestamp' DESC
				LIMIT 1
			)
		END AS status_text,
		created_at
	FROM invoices
	WHERE business_id = ? AND deleted_at IS NULL
	ORDER BY created_at DESC
	LIMIT ? OFFSET ?;
	`

	if err := db.DB().Raw(query, businessID, pagination.Limit, offset).Scan(&result).Error; err != nil {
		return nil, database.PaginationResponse{
			CurrentPage:     pagination.Page,
			PageCount:       0,
			TotalPagesCount: totalPages,
		}, err
	}

	return result, database.PaginationResponse{
		CurrentPage:     pagination.Page,
		PageCount:       len(result),
		TotalPagesCount: totalPages,
	}, nil
}

func FindInvoiceByBusinessAndID(db database.DatabaseManager, businessID, invoiceID string) (*models.Invoice, error) {
	var invoice models.Invoice
	if err := db.DB().
		Where("business_id = ? AND id = ?", businessID, invoiceID).
		First(&invoice).Error; err != nil {
		return nil, err
	}
	return &invoice, nil
}

func DeleteInvoiceByBusinessAndID(db database.DatabaseManager, businessID, invoiceID string) error {
	result := db.DB().
		Where("business_id = ? AND id = ?", businessID, invoiceID).
		Delete(&models.Invoice{})

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("invoice not found")
	}

	return nil
}

func UpdateInvoice(db database.DatabaseManager, invoiceNumber string, invoiceData []byte) error {
	result := db.DB().Model(&models.Invoice{}).Where("invoice_number = ?", invoiceNumber).Update("invoice_data", invoiceData)
	return result.Error
}

func CreateBulkUploadLog(db database.DatabaseManager, payload *models.BulkUpload) error {
	return db.DB().Create(payload).Error
}

func GetBulkUploadLogByFileKey(db database.DatabaseManager, fileKey, businessID string) (*models.BulkUpload, error) {
	var bulkUpload models.BulkUpload
	err := db.DB().Where("file_key = ? AND business_id = ?", fileKey, businessID).First(&bulkUpload).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &bulkUpload, nil
}

func UpdateBulkUploadLog(db database.DatabaseManager, fileKey, businessID string, payload *models.BulkUpload) error {

	result := db.DB().Model(&models.BulkUpload{}).Where("file_key = ? AND business_id = ?", fileKey, businessID).Updates(payload)
	return result.Error
}

func FindBulkUploadLogsByBusinessID(db database.DatabaseManager, businessID string, pagination database.Pagination) ([]models.BulkUpload, database.PaginationResponse, error) {
	var result []models.BulkUpload

	if pagination.Page <= 0 {
		pagination.Page = 1
	}
	if pagination.Limit <= 0 {
		pagination.Limit = 20
	}

	var totalCount int64
	if err := db.DB().Model(&models.BulkUpload{}).Where("business_id = ?", businessID).Count(&totalCount).Error; err != nil {
		return nil, database.PaginationResponse{
			CurrentPage:     pagination.Page,
			PageCount:       0,
			TotalPagesCount: 0,
		}, err
	}

	totalPages := int(math.Ceil(float64(totalCount) / float64(pagination.Limit)))
	offset := (pagination.Page - 1) * pagination.Limit

	if err := db.DB().
		Where("business_id = ?", businessID).
		Order("created_at DESC").
		Limit(pagination.Limit).
		Offset(offset).
		Find(&result).Error; err != nil {
		return nil, database.PaginationResponse{
			CurrentPage:     pagination.Page,
			PageCount:       0,
			TotalPagesCount: totalPages,
		}, err
	}

	return result, database.PaginationResponse{
		CurrentPage:     pagination.Page,
		PageCount:       len(result),
		TotalPagesCount: totalPages,
	}, nil
}
