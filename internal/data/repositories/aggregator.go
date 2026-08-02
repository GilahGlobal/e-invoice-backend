package repositories

import (
	"errors"
	"strings"

	"einvoice-access-point/internal/data/database"
	"einvoice-access-point/internal/data/entities"

	"gorm.io/gorm"
)

type AggregatorRepository struct {
	prodDb database.DatabaseManager
	testDb database.DatabaseManager
}

func NewAggregatorRepository(prodDb, testDb database.DatabaseManager) *AggregatorRepository {
	return &AggregatorRepository{prodDb: prodDb, testDb: testDb}
}

func (r *AggregatorRepository) GetAggregatorByID(db *gorm.DB, id string) (*entities.Business, error) {
	var aggregator entities.Business
	err := db.Where("id = ?", id).First(&aggregator).Error
	if err != nil {
		return nil, err
	}
	return &aggregator, nil
}

func (r *AggregatorRepository) ListAllAggregators(db *gorm.DB, search string, page, size int) ([]entities.Business, int64, error) {
	var aggregators []entities.Business
	var total int64

	query := db.Model(&entities.Business{}).Where("is_aggregator = ? AND email_verified = ?", true, true)

	if search != "" {
		searchPattern := "%" + strings.ToLower(search) + "%"
		query = query.Where("LOWER(name) LIKE ? OR LOWER(company_name) LIKE ? OR LOWER(email) LIKE ?", searchPattern, searchPattern, searchPattern)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * size
	if err := query.Offset(offset).Limit(size).Order("created_at DESC").Find(&aggregators).Error; err != nil {
		return nil, 0, err
	}

	return aggregators, total, nil
}

func (r *AggregatorRepository) CreateInvitation(invitation *entities.AggregatorInvitation, db *gorm.DB) error {
	return db.Create(invitation).Error
}

func (r *AggregatorRepository) GetInvitationByID(db *gorm.DB, id string) (*entities.AggregatorInvitation, error) {
	var invitation entities.AggregatorInvitation
	err := db.Where("id = ?", id).First(&invitation).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &invitation, nil
}

func (r *AggregatorRepository) GetInvitationByToken(db *gorm.DB, token string) (*entities.AggregatorInvitation, error) {
	var invitation entities.AggregatorInvitation
	err := db.Preload("Business").Where("invite_token = ?", token).First(&invitation).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &invitation, nil
}

func (r *AggregatorRepository) UpdateInvitation(invitation *entities.AggregatorInvitation, db *gorm.DB) error {
	return db.Save(invitation).Error
}

func (r *AggregatorRepository) ListPendingInvitationsByAggregator(db *gorm.DB, aggregatorID string) ([]entities.AggregatorInvitation, error) {
	var invitations []entities.AggregatorInvitation
	err := db.Where("aggregator_id = ? AND status = ?", aggregatorID, entities.InvitationStatusPending).
		Order("created_at DESC").
		Find(&invitations).Error
	return invitations, err
}

func (r *AggregatorRepository) ListInvitationsByBusiness(db *gorm.DB, businessID string) ([]entities.AggregatorInvitation, error) {
	var invitations []entities.AggregatorInvitation
	err := db.Where("business_id = ?", businessID).
		Order("created_at DESC").
		Find(&invitations).Error
	return invitations, err
}

func (r *AggregatorRepository) CheckExistingActiveInvitation(db *gorm.DB, businessID, aggregatorID string) (*entities.AggregatorInvitation, error) {
	var invitation entities.AggregatorInvitation
	err := db.Where("business_id = ? AND aggregator_id = ? AND status IN ?", businessID, aggregatorID,
		[]string{entities.InvitationStatusPending, entities.InvitationStatusAccepted}).
		First(&invitation).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &invitation, nil
}

func (r *AggregatorRepository) CheckBusinessHasAggregator(db *gorm.DB, businessID string) (bool, error) {
	var count int64
	err := db.Model(&entities.AggregatorInvitation{}).
		Where("business_id = ? AND status = ?", businessID, entities.InvitationStatusAccepted).
		Count(&count).Error
	return count > 0, err
}

func (r *AggregatorRepository) GetAcceptedBusinesses(db *gorm.DB, aggregatorID string, page, size int, search string) ([]entities.Business, int64, error) {
	var businesses []entities.Business
	var total int64

	query := db.Model(&entities.Business{}).Where("aggregator_id = ?", aggregatorID)

	if search != "" {
		searchPattern := "%" + strings.ToLower(search) + "%"
		query = query.Where("LOWER(name) LIKE ? OR LOWER(company_name) LIKE ? OR LOWER(email) LIKE ?", searchPattern, searchPattern, searchPattern)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * size
	if err := query.Offset(offset).Limit(size).Order("created_at DESC").Find(&businesses).Error; err != nil {
		return nil, 0, err
	}

	return businesses, total, nil
}

func (r *AggregatorRepository) GetBusinessStatsForAggregator(db *gorm.DB, aggregatorID, businessID string) (totalInvoices, totalBulkUploads int64, err error) {
	invoiceQuery := db.Model(&entities.Invoice{}).Where("aggregator_id = ? AND business_id = ?", aggregatorID, businessID)
	bulkUploadQuery := db.Model(&entities.BulkUpload{}).Where("aggregator_id = ? AND business_id = ?", aggregatorID, businessID)

	var lastTx entities.Transaction
	txErr := db.Where("business_id = ? AND aggregator_id = ? AND status = ?", businessID, aggregatorID, "success").
		Order("updated_at desc").
		First(&lastTx).Error

	if txErr == nil {
		invoiceQuery = invoiceQuery.Where("created_at >= ?", lastTx.UpdatedAt)
		bulkUploadQuery = bulkUploadQuery.Where("created_at >= ?", lastTx.UpdatedAt)
	}

	if err = invoiceQuery.Count(&totalInvoices).Error; err != nil {
		return
	}
	if err = bulkUploadQuery.Count(&totalBulkUploads).Error; err != nil {
		return
	}
	return
}

func (r *AggregatorRepository) GetInvoicesByAggregatorAndBusiness(db *gorm.DB, aggregatorID, businessID string, page, size int) ([]entities.Invoice, int64, error) {
	var invoices []entities.Invoice
	var total int64

	query := db.Model(&entities.Invoice{}).Where("aggregator_id = ? AND business_id = ?", aggregatorID, businessID)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * size
	if err := query.Offset(offset).Limit(size).Order("created_at DESC").Find(&invoices).Error; err != nil {
		return nil, 0, err
	}

	return invoices, total, nil
}

func (r *AggregatorRepository) GetAllInvoicesByAggregator(db *gorm.DB, aggregatorID string, page, size int) ([]entities.Invoice, int64, error) {
	var invoices []entities.Invoice
	var total int64

	query := db.Model(&entities.Invoice{}).Where("aggregator_id = ?", aggregatorID)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * size
	if err := query.Offset(offset).Limit(size).Order("created_at DESC").Find(&invoices).Error; err != nil {
		return nil, 0, err
	}

	return invoices, total, nil
}

func (r *AggregatorRepository) GetBulkUploadsByAggregatorAndBusiness(db *gorm.DB, aggregatorID, businessID string, page, size int) ([]entities.BulkUpload, int64, error) {
	var uploads []entities.BulkUpload
	var total int64

	query := db.Model(&entities.BulkUpload{}).Where("aggregator_id = ? AND business_id = ?", aggregatorID, businessID)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * size
	if err := query.Offset(offset).Limit(size).Order("created_at DESC").Find(&uploads).Error; err != nil {
		return nil, 0, err
	}

	return uploads, total, nil
}

func (r *AggregatorRepository) GetAllBulkUploadsByAggregator(db *gorm.DB, aggregatorID string, page, size int) ([]entities.BulkUpload, int64, error) {
	var uploads []entities.BulkUpload
	var total int64

	query := db.Model(&entities.BulkUpload{}).Where("aggregator_id = ?", aggregatorID)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * size
	if err := query.Offset(offset).Limit(size).Order("created_at DESC").Find(&uploads).Error; err != nil {
		return nil, 0, err
	}

	return uploads, total, nil
}

func (r *AggregatorRepository) GetBulkUploadByIDForAggregator(db *gorm.DB, aggregatorID, bulkUploadID string) (*entities.BulkUpload, error) {
	var bulkUpload entities.BulkUpload
	err := db.Where("id = ? AND aggregator_id = ?", bulkUploadID, aggregatorID).First(&bulkUpload).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &bulkUpload, nil
}

func (r *AggregatorRepository) CreateActivityLog(log *entities.AggregatorActivityLog, db *gorm.DB) error {
	return db.Create(log).Error
}

func (r *AggregatorRepository) GetActivityLogs(db *gorm.DB, aggregatorID string, page, size int) ([]entities.AggregatorActivityLog, int64, error) {
	var logs []entities.AggregatorActivityLog
	var total int64

	query := db.Model(&entities.AggregatorActivityLog{}).Where("aggregator_id = ?", aggregatorID)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * size
	if err := query.Offset(offset).Limit(size).Order("created_at DESC").Find(&logs).Error; err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}

func (r *AggregatorRepository) GetDashboardStats(db *gorm.DB, aggregatorID string) (totalBiz, pendingInvites, totalInvoices, totalBulkUploads int64, err error) {
	if err = db.Model(&entities.Business{}).Where("aggregator_id = ?", aggregatorID).Count(&totalBiz).Error; err != nil {
		return
	}
	if err = db.Model(&entities.AggregatorInvitation{}).Where("aggregator_id = ? AND status = ?", aggregatorID, entities.InvitationStatusPending).Count(&pendingInvites).Error; err != nil {
		return
	}
	if err = db.Model(&entities.Invoice{}).Where("aggregator_id = ?", aggregatorID).Count(&totalInvoices).Error; err != nil {
		return
	}
	if err = db.Model(&entities.BulkUpload{}).Where("aggregator_id = ?", aggregatorID).Count(&totalBulkUploads).Error; err != nil {
		return
	}
	return
}

func (r *AggregatorRepository) GetTransactionsByAggregator(db *gorm.DB, aggregatorID string, page, size int) ([]entities.Transaction, int64, error) {
	var transactions []entities.Transaction
	var total int64

	query := db.Model(&entities.Transaction{}).Where("aggregator_id = ? AND status = ?", aggregatorID, entities.TransactionStatusSuccess)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * size
	if err := query.Offset(offset).Limit(size).Order("created_at DESC").Find(&transactions).Error; err != nil {
		return nil, 0, err
	}

	return transactions, total, nil
}
