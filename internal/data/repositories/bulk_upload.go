package repositories

import (
	"errors"
	"math"

	"einvoice-access-point/internal/data/database"
	"einvoice-access-point/internal/data/entities"

	"gorm.io/gorm"
)

type BulkUploadRepository interface {
	CreateBulkUploadLog(db database.DatabaseManager, payload *entities.BulkUpload) error
	GetBulkUploadLogByID(db database.DatabaseManager, id, businessID string) (*entities.BulkUpload, error)
	UpdateBulkUploadLog(db database.DatabaseManager, bulkID, businessID string, payload *entities.BulkUpload) error
	FindBulkUploadLogsByBusinessID(db database.DatabaseManager, businessID string, pagination database.Pagination) ([]entities.BulkUpload, database.PaginationResponse, error)
	FindInvoiceByNumberAndBusinessID(db database.DatabaseManager, invoiceNumber string, businessID string) (*entities.Invoice, error)
}

type bulkUploadRepository struct {
	db     database.DatabaseManager
	testDB database.DatabaseManager
}

func NewBulkUploadRepository(db, testDB database.DatabaseManager) BulkUploadRepository {
	return &bulkUploadRepository{
		db:     db,
		testDB: testDB,
	}
}

func (r *bulkUploadRepository) CreateBulkUploadLog(db database.DatabaseManager, payload *entities.BulkUpload) error {
	return db.DB().Create(payload).Error
}

func (r *bulkUploadRepository) GetBulkUploadLogByID(db database.DatabaseManager, id, businessID string) (*entities.BulkUpload, error) {
	var bulkUpload entities.BulkUpload
	err := db.DB().Where("id = ? AND business_id = ?", id, businessID).First(&bulkUpload).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &bulkUpload, nil
}

func (r *bulkUploadRepository) UpdateBulkUploadLog(db database.DatabaseManager, bulkID, businessID string, payload *entities.BulkUpload) error {
	result := db.DB().Model(&entities.BulkUpload{}).Where("id = ? AND business_id = ?", bulkID, businessID).Updates(payload)
	return result.Error
}

func (r *bulkUploadRepository) FindBulkUploadLogsByBusinessID(db database.DatabaseManager, businessID string, pagination database.Pagination) ([]entities.BulkUpload, database.PaginationResponse, error) {
	var result []entities.BulkUpload

	if pagination.Page <= 0 {
		pagination.Page = 1
	}
	if pagination.Limit <= 0 {
		pagination.Limit = 20
	}

	var totalCount int64
	if err := db.DB().Model(&entities.BulkUpload{}).Where("business_id = ?", businessID).Count(&totalCount).Error; err != nil {
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

func (r *bulkUploadRepository) FindInvoiceByNumberAndBusinessID(db database.DatabaseManager, invoiceNumber string, businessID string) (*entities.Invoice, error) {
	var invoice entities.Invoice
	err := db.DB().Where("invoice_number = ? AND business_id = ?", invoiceNumber, businessID).First(&invoice).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &invoice, nil
}
