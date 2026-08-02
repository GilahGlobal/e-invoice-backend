package aggregator

import (
	bulkUploadPkg "einvoice-access-point/internal/app/bulk_upload"
	"einvoice-access-point/internal/data/entities"
	"einvoice-access-point/internal/data/repositories"

	"crypto/sha256"
	"einvoice-access-point/internal/common"
	"encoding/hex"
	"errors"
	"strings"

	"einvoice-access-point/internal/data/database"
	"einvoice-access-point/internal/data/dbinit"
	"einvoice-access-point/internal/utility"
	"fmt"
	"math"
	"net/http"
	"time"

	"gorm.io/gorm"
)

func buildPagination(page, size int, total int64) *database.PaginationResponse {
	return &database.PaginationResponse{
		CurrentPage:     page,
		PageCount:       size,
		TotalPagesCount: int(math.Ceil(float64(total) / float64(size))),
	}
}

func (s *Service) ListBusinesses(aggregatorID string, page, size int, search string, db *gorm.DB) ([]AggregatorBusinessDetailDto, *database.PaginationResponse, error) {
	businesses, total, err := s.repo.GetAcceptedBusinesses(db, aggregatorID, page, size, search)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to fetch businesses: %w", err)
	}

	result := make([]AggregatorBusinessDetailDto, 0, len(businesses))
	for _, b := range businesses {
		result = append(result, AggregatorBusinessDetailDto{
			ID:          b.ID,
			Name:        b.Name,
			Email:       b.Email,
			CompanyName: b.CompanyName,
			TIN:         b.TIN,
			PhoneNumber: b.PhoneNumber,
			ServiceID:   b.ServiceID,
			BusinessID:  b.BusinessID,
			KeysSet:     b.KeysSet,
		})
	}

	return result, buildPagination(page, size, total), nil
}

func (s *Service) GetBusinessDetail(aggregatorID, businessID string, db *gorm.DB) (*AggregatorBusinessFullDetailDto, int, error) {
	pdb := dbinit.InitDB(db, false)
	bRepo := repositories.NewBusinessRepository(pdb, pdb)
	business, err := bRepo.GetBusinessByIDForAggregator(pdb, aggregatorID, businessID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, http.StatusNotFound, fmt.Errorf("business not found or not managed by this aggregator")
		}
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to fetch business: %w", err)
	}
	if business == nil {
		return nil, http.StatusNotFound, fmt.Errorf("business not found or not managed by this aggregator")
	}

	result := &AggregatorBusinessFullDetailDto{
		ID:          business.ID,
		Name:        business.Name,
		Email:       business.Email,
		CompanyName: business.CompanyName,
		TIN:         business.TIN,
		PhoneNumber: business.PhoneNumber,
		ServiceID:   business.ServiceID,
		BusinessID:  business.BusinessID,
		KeysSet:     business.KeysSet,
	}

	// Fetch subscription info (best-effort, won't fail the request)
	subscription, _, _ := s.subSvc.RequireAggregatorBusinessSubscription(db, aggregatorID, businessID)
	if subscription != nil {
		subInfo := &BusinessSubscriptionInfoDto{
			IsActive:          subscription.IsActive,
			PlanID:            subscription.PlanID,
			PlanName:          subscription.Plan,
			TotalInvoices:     subscription.TotalInvoices,
			UsedInvoices:      subscription.UsedInvoices,
			RemainingInvoices: subscription.RemainingInvoices,
			NextBillingDate:   subscription.NextBillingDate.Format(time.RFC3339),
		}

		// Enrich with plan details
		if subscription.PlanID != "" {
			plan, found, _ := s.subSvc.GetPlanByID(subscription.PlanID, db)
			if found && plan != nil {
				subInfo.PlanAmount = plan.Amount
				subInfo.BillingCycleDays = plan.BillingCycle
			}
		}

		result.Subscription = subInfo
	}

	// Fetch usage stats (best-effort)
	totalInvoices, totalBulkUploads, _ := s.repo.GetBusinessStatsForAggregator(db, aggregatorID, businessID)
	result.TotalInvoicesUploaded = totalInvoices
	result.TotalBulkUploads = totalBulkUploads

	return result, http.StatusOK, nil
}

func (s *Service) RemoveBusiness(aggregatorID, businessID string, db *gorm.DB) (int, error) {
	pdb := dbinit.InitDB(db, false)
	bRepo := repositories.NewBusinessRepository(pdb, pdb)
	business, err := bRepo.GetBusinessByIDForAggregator(pdb, aggregatorID, businessID)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to fetch business: %w", err)
	}
	if business == nil {
		return http.StatusNotFound, fmt.Errorf("business not found or not managed by this aggregator")
	}

	// Unlink business
	if err := db.Model(&entities.Business{}).Where("id = ?", businessID).
		Update("aggregator_id", nil).Error; err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to unlink business: %w", err)
	}

	// Revoke the accepted invitation
	db.Model(&entities.AggregatorInvitation{}).
		Where("business_id = ? AND aggregator_id = ? AND status = ?", businessID, aggregatorID, entities.InvitationStatusAccepted).
		Update("status", entities.InvitationStatusRevoked)

	// Log activity
	s.repo.CreateActivityLog(&entities.AggregatorActivityLog{
		ID:           utility.GenerateUUID(),
		AggregatorID: aggregatorID,
		BusinessID:   businessID,
		Action:       entities.ActivityBusinessRemoved,
		Details:      fmt.Sprintf("Removed business %s", business.CompanyName),
	}, db)

	return http.StatusOK, nil
}

func (s *Service) ListInvoicesByBusiness(aggregatorID, businessID string, page, size int, db *gorm.DB) ([]entities.MinimalInvoiceDTO, *database.PaginationResponse, error) {
	pdb := dbinit.InitDB(db, false)
	bRepo := repositories.NewBusinessRepository(pdb, pdb)
	business, err := bRepo.GetBusinessByIDForAggregator(pdb, aggregatorID, businessID)
	if err != nil || business == nil {
		return nil, nil, fmt.Errorf("business not found or not managed by this aggregator")
	}

	invoices, total, err := s.repo.GetInvoicesByAggregatorAndBusiness(db, aggregatorID, businessID, page, size)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to fetch invoices: %w", err)
	}

	result := make([]entities.MinimalInvoiceDTO, 0, len(invoices))
	for _, inv := range invoices {
		result = append(result, entities.MinimalInvoiceDTO{
			ID:            inv.ID,
			InvoiceNumber: inv.InvoiceNumber,
			IRN:           inv.IRN,
			Platform:      inv.Platform,
			CurrentStatus: inv.CurrentStatus,
			PaymentStatus: inv.PaymentStatus,
			StatusText:    inv.CurrentStatus,
			QrCodeBmpUrl:  inv.QrCodeBmpUrl,
			QrCode:        inv.QrCode,
			CreatedAt:     inv.CreatedAt,
		})
	}

	return result, buildPagination(page, size, total), nil
}

func (s *Service) ListAllInvoices(aggregatorID string, page, size int, db *gorm.DB) ([]entities.MinimalInvoiceDTO, *database.PaginationResponse, error) {
	invoices, total, err := s.repo.GetAllInvoicesByAggregator(db, aggregatorID, page, size)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to fetch invoices: %w", err)
	}

	result := make([]entities.MinimalInvoiceDTO, 0, len(invoices))
	for _, inv := range invoices {
		result = append(result, entities.MinimalInvoiceDTO{
			ID:            inv.ID,
			InvoiceNumber: inv.InvoiceNumber,
			IRN:           inv.IRN,
			Platform:      inv.Platform,
			CurrentStatus: inv.CurrentStatus,
			PaymentStatus: inv.PaymentStatus,
			StatusText:    inv.CurrentStatus,
			QrCodeBmpUrl:  inv.QrCodeBmpUrl,
			QrCode:        inv.QrCode,
			CreatedAt:     inv.CreatedAt,
		})
	}

	return result, buildPagination(page, size, total), nil
}

func (s *Service) ListBulkUploadsByBusiness(aggregatorID, businessID string, page, size int, db *gorm.DB) ([]entities.BulkUpload, *database.PaginationResponse, error) {
	pdb := dbinit.InitDB(db, false)
	bRepo := repositories.NewBusinessRepository(pdb, pdb)
	business, err := bRepo.GetBusinessByIDForAggregator(pdb, aggregatorID, businessID)
	if err != nil || business == nil {
		return nil, nil, fmt.Errorf("business not found or not managed by this aggregator")
	}

	repo := repositories.NewBulkUploadRepository(pdb, pdb)
	bulkUploadSvc := bulkUploadPkg.NewService(repo)
	uploads, pagination, err := bulkUploadSvc.GetBulkUploadLogsByBusinessID(db, businessID, page, size)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to fetch bulk uploads: %w", err)
	}

	return uploads, &pagination, nil
}

func (s *Service) ListAllBulkUploads(aggregatorID string, page, size int, db *gorm.DB) ([]entities.BulkUpload, *database.PaginationResponse, error) {
	uploads, total, err := s.repo.GetAllBulkUploadsByAggregator(db, aggregatorID, page, size)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to fetch bulk uploads: %w", err)
	}

	return uploads, buildPagination(page, size, total), nil
}

func (s *Service) GetBulkUploadFailedInvoices(aggregatorID, bulkUploadID string, db *gorm.DB) (*BulkUploadFailedInvoicesDto, int, error) {
	bulkUpload, err := s.repo.GetBulkUploadByIDForAggregator(db, aggregatorID, bulkUploadID)
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to fetch bulk upload: %w", err)
	}
	if bulkUpload == nil {
		return nil, http.StatusNotFound, fmt.Errorf("bulk upload not found or not uploaded by this aggregator")
	}

	pdb := dbinit.InitDB(db, false)
	repo := repositories.NewBulkUploadRepository(pdb, pdb)
	bulkUploadSvc := bulkUploadPkg.NewService(repo)
	failedInvoices, err := bulkUploadSvc.BuildBulkUploadFailedInvoicesResponse(db, bulkUpload)
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}

	return failedInvoices, http.StatusOK, nil
}

func (s *Service) GetDashboard(aggregatorID string, db *gorm.DB) (*AggregatorDashboardDto, error) {
	totalBiz, pendingInvites, totalInvoices, totalBulkUploads, err := s.repo.GetDashboardStats(db, aggregatorID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch dashboard stats: %w", err)
	}

	return &AggregatorDashboardDto{
		TotalBusinesses:    totalBiz,
		PendingInvitations: pendingInvites,
		TotalInvoices:      totalInvoices,
		TotalBulkUploads:   totalBulkUploads,
	}, nil
}

func (s *Service) GetActivityLog(aggregatorID string, page, size int, db *gorm.DB) ([]AggregatorActivityLogDto, *database.PaginationResponse, error) {
	logs, total, err := s.repo.GetActivityLogs(db, aggregatorID, page, size)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to fetch activity logs: %w", err)
	}

	result := make([]AggregatorActivityLogDto, 0, len(logs))
	for _, l := range logs {
		result = append(result, AggregatorActivityLogDto{
			ID:           l.ID,
			AggregatorID: l.AggregatorID,
			BusinessID:   l.BusinessID,
			Action:       l.Action,
			Details:      l.Details,
			CreatedAt:    l.CreatedAt.Format(time.RFC3339),
		})
	}

	return result, buildPagination(page, size, total), nil
}

func (s *Service) ListAllTransactions(aggregatorID string, page, size int, db *gorm.DB) ([]TransactionDto, *database.PaginationResponse, error) {
	transactions, total, err := s.repo.GetTransactionsByAggregator(db, aggregatorID, page, size)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to fetch transactions: %w", err)
	}

	result := make([]TransactionDto, 0, len(transactions))
	businessNameCache := make(map[string]string)

	for _, t := range transactions {
		businessName := businessNameCache[t.BusinessID]
		if businessName == "" {
			pdb := dbinit.InitDB(db, false)
			bRepo := repositories.NewBusinessRepository(pdb, pdb)
			biz, err := bRepo.GetBusinessByIDForAggregator(pdb, aggregatorID, t.BusinessID)
			if err == nil && biz != nil {
				businessName = biz.CompanyName
				if businessName == "" {
					businessName = biz.Name
				}
				businessNameCache[t.BusinessID] = businessName
			}
		}

		result = append(result, TransactionDto{
			ID:           t.ID,
			BusinessID:   t.BusinessID,
			BusinessName: businessName,
			AggregatorID: t.AggregatorID,
			Reference:    t.Reference,
			Provider:     t.Provider,
			Status:       string(t.Status),
			Amount:       t.Amount,
			Currency:     t.Currency,
			PlanID:       t.PlanID,
			Plan:         t.Plan,
			CreatedAt:    t.CreatedAt.Format(time.RFC3339),
			UpdatedAt:    t.UpdatedAt.Format(time.RFC3339),
		})
	}

	return result, buildPagination(page, size, total), nil
}

func (s *Service) UpdateBusinessSetup(db *gorm.DB, businessID string, req AggregatorUpdateBusinessSetupDto) error {
	updates := make(map[string]interface{})

	if req.ServiceID != nil {
		updates["service_id"] = *req.ServiceID
	}
	if req.BusinessID != nil {
		updates["business_id"] = *req.BusinessID
	}

	if len(updates) == 0 {
		return nil
	}

	if err := db.Model(&entities.Business{}).Where("id = ?", businessID).Updates(updates).Error; err != nil {
		return fmt.Errorf("failed to update business setup: %w", err)
	}

	return nil
}

func (s *Service) LogActivity(db *gorm.DB, aggregatorID, businessID, action, details string) {
	s.repo.CreateActivityLog(&entities.AggregatorActivityLog{
		ID:           utility.GenerateUUID(),
		AggregatorID: aggregatorID,
		BusinessID:   businessID,
		Action:       action,
		Details:      details,
	}, db)
}

func (s *Service) CreateBusiness(db *gorm.DB, req CreateBusinessDto, aggregatorID string) error {
	serverSecret := s.cfg.Server.Secret
	email := strings.ToLower(req.Email)
	name := strings.Title(strings.ToLower(req.Name))

	var existingBusiness entities.Business
	if err := db.Where("email = ?", email).First(&existingBusiness).Error; err == nil {
		return fmt.Errorf("business already exists with the given email")
	}

	if err := db.Where("LOWER(company_name) = LOWER(?)", req.CompanyName).First(&existingBusiness).Error; err == nil {
		return fmt.Errorf("business already exists with the given company name")
	}

	password, err := utility.HashPassword(req.Password)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	apiKey, err := utility.GenerateSecureToken(32, serverSecret)
	if err != nil {
		return fmt.Errorf("failed to generate api key: %w", err)
	}

	encryptedAPIKey, err := common.EncryptAES(apiKey)
	if err != nil {
		return fmt.Errorf("failed to encrypt API key: %w", err)
	}

	apiKeyHash := sha256.Sum256([]byte(apiKey))
	apiKeyHashStr := hex.EncodeToString(apiKeyHash[:])

	business := entities.Business{
		ID:            utility.GenerateUUID(),
		Name:          name,
		Email:         email,
		Password:      password,
		APIKey:        common.EncryptedString(encryptedAPIKey),
		APIKeyHash:    apiKeyHashStr,
		CompanyName:   req.CompanyName,
		PhoneNumber:   req.PhoneNumber,
		TIN:           req.TIN,
		IsAggregator:  false,
		EmailVerified: true,
		AggregatorID:  &aggregatorID,
	}

	if err := db.Create(&business).Error; err != nil {
		return fmt.Errorf("failed to create business: %w", err)
	}

	return nil
}

func (s *Service) UpdateBusinessProfile(db *gorm.DB, businessID string, req UpdateBusinessProfileDto, aggregatorID string) error {
	var business entities.Business
	if err := db.Where("id = ? AND aggregator_id = ?", businessID, aggregatorID).First(&business).Error; err != nil {
		return fmt.Errorf("business not found or not managed by this aggregator")
	}

	encryptedPublicKey, err := common.EncryptAES(req.IRNPublicKey)
	if err != nil {
		return fmt.Errorf("failed to encrypt IRN public key: %w", err)
	}

	encryptedCertificate, err := common.EncryptAES(req.IRNCertificate)
	if err != nil {
		return fmt.Errorf("failed to encrypt IRN certificate: %w", err)
	}

	updateData := map[string]interface{}{
		"service_id":      req.ServiceID,
		"irn_public_key":  common.EncryptedString(encryptedPublicKey),
		"irn_certificate": common.EncryptedString(encryptedCertificate),
		"keys_set":        true,
	}

	if err := db.Model(&business).Updates(updateData).Error; err != nil {
		return fmt.Errorf("failed to update business profile: %w", err)
	}

	return nil
}

func (s *Service) GetInvoiceDetail(aggregatorID, invoiceID string, db *gorm.DB) (*entities.Invoice, error) {
	var invoice entities.Invoice
	if err := db.Where("id = ? AND aggregator_id = ?", invoiceID, aggregatorID).First(&invoice).Error; err != nil {
		return nil, fmt.Errorf("invoice not found or does not belong to this aggregator")
	}
	return &invoice, nil
}
