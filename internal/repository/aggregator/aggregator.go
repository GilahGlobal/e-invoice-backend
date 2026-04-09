package aggregator

import (
	"einvoice-access-point/pkg/models"
	"errors"
	"strings"

	"gorm.io/gorm"
)

// =====================
// Aggregator CRUD
// =====================

func CreateAggregator(aggregator *models.Aggregator, db *gorm.DB) error {
	return db.Create(aggregator).Error
}

func GetAggregatorByEmail(db *gorm.DB, email string) (*models.Aggregator, error) {
	var aggregator models.Aggregator
	err := db.Where("email = ?", strings.ToLower(email)).First(&aggregator).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &aggregator, nil
}

func GetAggregatorByID(db *gorm.DB, id string) (*models.Aggregator, error) {
	var aggregator models.Aggregator
	err := db.Where("id = ?", id).First(&aggregator).Error
	if err != nil {
		return nil, err
	}
	return &aggregator, nil
}

func UpdateAggregator(aggregator *models.Aggregator, db *gorm.DB) error {
	return db.Save(aggregator).Error
}

func CheckAggregatorEmailExists(db *gorm.DB, email string) bool {
	var count int64
	db.Model(&models.Aggregator{}).Where("email = ?", strings.ToLower(email)).Count(&count)
	return count > 0
}

func CheckAggregatorCompanyExists(db *gorm.DB, companyName string) bool {
	var count int64
	db.Model(&models.Aggregator{}).Where("LOWER(company_name) = LOWER(?)", companyName).Count(&count)
	return count > 0
}

// ListAllAggregators returns a paginated list of active aggregators for business browsing
func ListAllAggregators(db *gorm.DB, search string, page, size int) ([]models.Aggregator, int64, error) {
	var aggregators []models.Aggregator
	var total int64

	query := db.Model(&models.Aggregator{}).Where("is_active = ? AND email_verified = ?", true, true)

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

// =====================
// Invitations
// =====================

func CreateInvitation(invitation *models.AggregatorInvitation, db *gorm.DB) error {
	return db.Create(invitation).Error
}

func GetInvitationByID(db *gorm.DB, id string) (*models.AggregatorInvitation, error) {
	var invitation models.AggregatorInvitation
	err := db.Preload("Business").Preload("Aggregator").Where("id = ?", id).First(&invitation).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &invitation, nil
}

func GetInvitationByToken(db *gorm.DB, token string) (*models.AggregatorInvitation, error) {
	var invitation models.AggregatorInvitation
	err := db.Preload("Business").Preload("Aggregator").Where("invite_token = ?", token).First(&invitation).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &invitation, nil
}

func UpdateInvitation(invitation *models.AggregatorInvitation, db *gorm.DB) error {
	return db.Save(invitation).Error
}

// ListPendingInvitationsByAggregator returns all pending invitations for an aggregator
func ListPendingInvitationsByAggregator(db *gorm.DB, aggregatorID string) ([]models.AggregatorInvitation, error) {
	var invitations []models.AggregatorInvitation
	err := db.Preload("Business").
		Where("aggregator_id = ? AND status = ?", aggregatorID, models.InvitationStatusPending).
		Order("created_at DESC").
		Find(&invitations).Error
	return invitations, err
}

// ListInvitationsByBusiness returns all invitations sent by a business
func ListInvitationsByBusiness(db *gorm.DB, businessID string) ([]models.AggregatorInvitation, error) {
	var invitations []models.AggregatorInvitation
	err := db.Preload("Aggregator").
		Where("business_id = ?", businessID).
		Order("created_at DESC").
		Find(&invitations).Error
	return invitations, err
}

// CheckExistingActiveInvitation checks if there's already an active (pending/accepted) invitation
func CheckExistingActiveInvitation(db *gorm.DB, businessID, aggregatorID string) (*models.AggregatorInvitation, error) {
	var invitation models.AggregatorInvitation
	err := db.Where("business_id = ? AND aggregator_id = ? AND status IN ?", businessID, aggregatorID,
		[]string{models.InvitationStatusPending, models.InvitationStatusAccepted}).
		First(&invitation).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &invitation, nil
}

// CheckBusinessHasAggregator checks if a business already has a linked aggregator
func CheckBusinessHasAggregator(db *gorm.DB, businessID string) (bool, error) {
	var count int64
	err := db.Model(&models.AggregatorInvitation{}).
		Where("business_id = ? AND status = ?", businessID, models.InvitationStatusAccepted).
		Count(&count).Error
	return count > 0, err
}

// =====================
// Business Management
// =====================

// GetAcceptedBusinesses returns paginated businesses that accepted invitations from this aggregator
func GetAcceptedBusinesses(db *gorm.DB, aggregatorID string, page, size int, search string) ([]models.Business, int64, error) {
	var businesses []models.Business
	var total int64

	query := db.Model(&models.Business{}).Where("aggregator_id = ?", aggregatorID)

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

// GetBusinessByIDForAggregator fetches a business that belongs to the aggregator
func GetBusinessByIDForAggregator(db *gorm.DB, aggregatorID, businessID string) (*models.Business, error) {
	var business models.Business
	err := db.Where("id = ? AND aggregator_id = ?", businessID, aggregatorID).First(&business).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &business, nil
}

// =====================
// Invoices & Bulk Uploads for Aggregator
// =====================

// GetInvoicesByAggregatorAndBusiness returns invoices uploaded by this aggregator for a specific business
func GetInvoicesByAggregatorAndBusiness(db *gorm.DB, aggregatorID, businessID string, page, size int) ([]models.Invoice, int64, error) {
	var invoices []models.Invoice
	var total int64

	query := db.Model(&models.Invoice{}).Where("aggregator_id = ? AND business_id = ?", aggregatorID, businessID)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * size
	if err := query.Offset(offset).Limit(size).Order("created_at DESC").Find(&invoices).Error; err != nil {
		return nil, 0, err
	}

	return invoices, total, nil
}

// GetAllInvoicesByAggregator returns all invoices uploaded by this aggregator across all businesses
func GetAllInvoicesByAggregator(db *gorm.DB, aggregatorID string, page, size int) ([]models.Invoice, int64, error) {
	var invoices []models.Invoice
	var total int64

	query := db.Model(&models.Invoice{}).Where("aggregator_id = ?", aggregatorID)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * size
	if err := query.Offset(offset).Limit(size).Order("created_at DESC").Find(&invoices).Error; err != nil {
		return nil, 0, err
	}

	return invoices, total, nil
}

// GetBulkUploadsByAggregatorAndBusiness returns bulk uploads by this aggregator for a specific business
func GetBulkUploadsByAggregatorAndBusiness(db *gorm.DB, aggregatorID, businessID string, page, size int) ([]models.BulkUpload, int64, error) {
	var uploads []models.BulkUpload
	var total int64

	query := db.Model(&models.BulkUpload{}).Where("aggregator_id = ? AND business_id = ?", aggregatorID, businessID)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * size
	if err := query.Offset(offset).Limit(size).Order("created_at DESC").Find(&uploads).Error; err != nil {
		return nil, 0, err
	}

	return uploads, total, nil
}

// GetAllBulkUploadsByAggregator returns all bulk uploads by this aggregator
func GetAllBulkUploadsByAggregator(db *gorm.DB, aggregatorID string, page, size int) ([]models.BulkUpload, int64, error) {
	var uploads []models.BulkUpload
	var total int64

	query := db.Model(&models.BulkUpload{}).Where("aggregator_id = ?", aggregatorID)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * size
	if err := query.Offset(offset).Limit(size).Order("created_at DESC").Find(&uploads).Error; err != nil {
		return nil, 0, err
	}

	return uploads, total, nil
}

// =====================
// Activity Log
// =====================

func CreateActivityLog(log *models.AggregatorActivityLog, db *gorm.DB) error {
	return db.Create(log).Error
}

func GetActivityLogs(db *gorm.DB, aggregatorID string, page, size int) ([]models.AggregatorActivityLog, int64, error) {
	var logs []models.AggregatorActivityLog
	var total int64

	query := db.Model(&models.AggregatorActivityLog{}).Where("aggregator_id = ?", aggregatorID)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * size
	if err := query.Offset(offset).Limit(size).Order("created_at DESC").Find(&logs).Error; err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}

// =====================
// Dashboard Stats
// =====================

func GetDashboardStats(db *gorm.DB, aggregatorID string) (totalBiz, pendingInvites, totalInvoices, totalBulkUploads int64, err error) {
	if err = db.Model(&models.Business{}).Where("aggregator_id = ?", aggregatorID).Count(&totalBiz).Error; err != nil {
		return
	}
	if err = db.Model(&models.AggregatorInvitation{}).Where("aggregator_id = ? AND status = ?", aggregatorID, models.InvitationStatusPending).Count(&pendingInvites).Error; err != nil {
		return
	}
	if err = db.Model(&models.Invoice{}).Where("aggregator_id = ?", aggregatorID).Count(&totalInvoices).Error; err != nil {
		return
	}
	if err = db.Model(&models.BulkUpload{}).Where("aggregator_id = ?", aggregatorID).Count(&totalBulkUploads).Error; err != nil {
		return
	}
	return
}
