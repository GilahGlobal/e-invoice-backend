package repositories

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"strconv"
	"time"

	"einvoice-access-point/internal/data/database"
	"einvoice-access-point/internal/data/entities"

	"gorm.io/gorm"
)

type InvoiceRepository struct {
	db     database.DatabaseManager
	testDB database.DatabaseManager
}

func NewInvoiceRepository(db, testDB database.DatabaseManager) *InvoiceRepository {
	return &InvoiceRepository{
		db:     db,
		testDB: testDB,
	}
}

func (r *InvoiceRepository) GenerateUniqueInvoiceID(businessID string, db *gorm.DB) string {
	var lastInvoice entities.Invoice
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

func (r *InvoiceRepository) CreateInvoice(db database.DatabaseManager, invoice *entities.Invoice) error {
	return db.DB().Create(invoice).Error
}

func (r *InvoiceRepository) FindInvoiceByNumber(db database.DatabaseManager, invoiceNumber string) (*entities.Invoice, error) {
	var invoice entities.Invoice
	err := db.DB().Where("invoice_number = ?", invoiceNumber).First(&invoice).Error
	if err != nil {
		return nil, err
	}
	return &invoice, nil
}

func (r *InvoiceRepository) FindInvoiceByNumberAndBusinessID(db database.DatabaseManager, invoiceNumber string, businessID string) (*entities.Invoice, error) {
	var invoice entities.Invoice
	err := db.DB().Where("invoice_number = ? AND business_id = ?", invoiceNumber, businessID).First(&invoice).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &invoice, err
}

func (r *InvoiceRepository) FindInvoiceByIRNAndBusinessID(db database.DatabaseManager, irn string, businessID string) (*entities.Invoice, error) {
	var invoice entities.Invoice
	err := db.DB().Where("irn = ? AND business_id = ?", irn, businessID).First(&invoice).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &invoice, err
}

func (r *InvoiceRepository) UpdateInvoiceStatus(db database.DatabaseManager, invoice *entities.Invoice, step string, status string) error {
	var history []entities.StatusHistoryEntry

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

func (r *InvoiceRepository) UpdateInvoiceIRN(db database.DatabaseManager, invoice *entities.Invoice, irn string) error {
	invoice.IRN = irn
	return db.DB().Save(invoice).Error
}

func (r *InvoiceRepository) FindMinimalInvoicesByBusinessID(db database.DatabaseManager, businessID string, pagination database.Pagination) ([]entities.MinimalInvoiceDTO, database.PaginationResponse, error) {
	var result []entities.MinimalInvoiceDTO

	if pagination.Page <= 0 {
		pagination.Page = 1
	}
	if pagination.Limit <= 0 {
		pagination.Limit = 20
	}

	var totalCount int64
	if err := db.DB().Model(&entities.Invoice{}).Where("business_id = ? AND deleted_at IS NULL", businessID).Count(&totalCount).Error; err != nil {
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
		payment_status,
		qr_code_bmp_url,
		encrypted_irn AS qr_code,
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

func (r *InvoiceRepository) FindInvoiceByBusinessAndID(db database.DatabaseManager, businessID, invoiceID string) (*entities.Invoice, error) {
	var invoice entities.Invoice
	if err := db.DB().
		Where("business_id = ? AND id = ?", businessID, invoiceID).
		First(&invoice).Error; err != nil {
		return nil, err
	}
	return &invoice, nil
}

func (r *InvoiceRepository) DeleteInvoiceByBusinessAndID(db database.DatabaseManager, businessID, invoiceID string) error {
	result := db.DB().
		Where("business_id = ? AND id = ?", businessID, invoiceID).
		Delete(&entities.Invoice{})

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("invoice not found")
	}

	return nil
}

func (r *InvoiceRepository) UpdateInvoice(db database.DatabaseManager, invoiceNumber string, invoiceData []byte) error {
	result := db.DB().Model(&entities.Invoice{}).Where("invoice_number = ?", invoiceNumber).Update("invoice_data", invoiceData)
	return result.Error
}

func (r *InvoiceRepository) UpdateInvoiceDataByID(db database.DatabaseManager, invoiceID string, invoiceData []byte) error {
	result := db.DB().Model(&entities.Invoice{}).Where("id = ?", invoiceID).Update("invoice_data", invoiceData)
	return result.Error
}

func (r *InvoiceRepository) SaveInvoice(db database.DatabaseManager, invoice *entities.Invoice) error {
	return db.DB().Save(invoice).Error
}

func (r *InvoiceRepository) GetInvoiceStats(db *gorm.DB, businessID *string, aggregatorID *string) (*entities.InvoiceStatsResponseData, error) {
	var monthlyResults []entities.MonthlyInvoiceStatsDto

	query := `
	SELECT 
		TO_CHAR(created_at, 'YYYYMM') AS month,
		COUNT(*) AS total_invoices,
		SUM(CASE WHEN current_status = 'confirmed_invoice' THEN 1 ELSE 0 END) AS successful_invoices,
		SUM(CASE WHEN current_status IN ('signed_invoice', 'transmitted_invoice') THEN 1 ELSE 0 END) AS partial_invoices,
		SUM(CASE 
			WHEN current_status NOT IN ('confirmed_invoice', 'signed_invoice', 'transmitted_invoice') 
			AND (
				SELECT COALESCE(entry->>'status', 'pending')
				FROM jsonb_array_elements(status_history) AS entry
				WHERE entry->>'step' = invoices.current_status
				ORDER BY entry->>'timestamp' DESC
				LIMIT 1
			) = 'failed' THEN 1 ELSE 0 END) AS failed_invoices
	FROM invoices
	WHERE deleted_at IS NULL
	`

	args := []interface{}{}

	if businessID != nil && *businessID != "" {
		query += " AND business_id = ?"
		args = append(args, *businessID)
	}

	if aggregatorID != nil && *aggregatorID != "" {
		query += " AND aggregator_id = ?"
		args = append(args, *aggregatorID)
	}

	query += " GROUP BY TO_CHAR(created_at, 'YYYYMM') ORDER BY month DESC;"

	if err := db.Raw(query, args...).Scan(&monthlyResults).Error; err != nil {
		return nil, err
	}

	totalStats := entities.InvoiceStatsDto{}
	for _, m := range monthlyResults {
		totalStats.TotalInvoices += m.TotalInvoices
		totalStats.SuccessfulInvoices += m.SuccessfulInvoices
		totalStats.PartialInvoices += m.PartialInvoices
		totalStats.FailedInvoices += m.FailedInvoices
	}

	return &entities.InvoiceStatsResponseData{
		Total:   totalStats,
		Monthly: monthlyResults,
	}, nil
}
