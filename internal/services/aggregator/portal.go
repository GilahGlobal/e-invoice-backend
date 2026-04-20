package aggregator

import (
	"einvoice-access-point/internal/dtos"
	aggregatorRepo "einvoice-access-point/internal/repository/aggregator"
	planRepo "einvoice-access-point/internal/repository/plan"
	subscriptionRepo "einvoice-access-point/internal/repository/subscription"
	"einvoice-access-point/pkg/database"
	inst "einvoice-access-point/pkg/dbinit"
	"einvoice-access-point/pkg/models"
	"einvoice-access-point/pkg/utility"
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

func ListBusinesses(aggregatorID string, page, size int, search string, db *gorm.DB) ([]dtos.AggregatorBusinessDetailDto, *database.PaginationResponse, error) {
	businesses, total, err := aggregatorRepo.GetAcceptedBusinesses(db, aggregatorID, page, size, search)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to fetch businesses: %w", err)
	}

	result := make([]dtos.AggregatorBusinessDetailDto, 0, len(businesses))
	for _, b := range businesses {
		result = append(result, dtos.AggregatorBusinessDetailDto{
			ID:          b.ID,
			Name:        b.Name,
			Email:       b.Email,
			CompanyName: b.CompanyName,
			TIN:         b.TIN,
			PhoneNumber: b.PhoneNumber,
			ServiceID:   b.ServiceID,
		})
	}

	return result, buildPagination(page, size, total), nil
}

func GetBusinessDetail(aggregatorID, businessID string, db *gorm.DB) (*dtos.AggregatorBusinessFullDetailDto, int, error) {
	business, err := aggregatorRepo.GetBusinessByIDForAggregator(db, aggregatorID, businessID)
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to fetch business: %w", err)
	}
	if business == nil {
		return nil, http.StatusNotFound, fmt.Errorf("business not found or not managed by this aggregator")
	}

	result := &dtos.AggregatorBusinessFullDetailDto{
		ID:          business.ID,
		Name:        business.Name,
		Email:       business.Email,
		CompanyName: business.CompanyName,
		TIN:         business.TIN,
		PhoneNumber: business.PhoneNumber,
		ServiceID:   business.ServiceID,
	}

	// Fetch subscription info (best-effort, won't fail the request)
	pdb := inst.InitDB(db, false)
	subscription, _ := subscriptionRepo.GetLatestSubscriptionByBusinessAndAggregator(pdb, businessID, aggregatorID)
	if subscription != nil {
		subInfo := &dtos.BusinessSubscriptionInfoDto{
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
			plan, _ := planRepo.GetPlanByID(subscription.PlanID, pdb)
			if plan != nil {
				subInfo.PlanAmount = plan.Amount
				subInfo.BillingCycleDays = plan.BillingCycle
			}
		}

		result.Subscription = subInfo
	}

	// Fetch usage stats (best-effort)
	totalInvoices, totalBulkUploads, _ := aggregatorRepo.GetBusinessStatsForAggregator(db, aggregatorID, businessID)
	result.TotalInvoicesUploaded = totalInvoices
	result.TotalBulkUploads = totalBulkUploads

	return result, http.StatusOK, nil
}

func RemoveBusiness(aggregatorID, businessID string, db *gorm.DB) (int, error) {
	business, err := aggregatorRepo.GetBusinessByIDForAggregator(db, aggregatorID, businessID)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to fetch business: %w", err)
	}
	if business == nil {
		return http.StatusNotFound, fmt.Errorf("business not found or not managed by this aggregator")
	}

	// Unlink business
	if err := db.Model(&models.Business{}).Where("id = ?", businessID).
		Update("aggregator_id", nil).Error; err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to unlink business: %w", err)
	}

	// Revoke the accepted invitation
	db.Model(&models.AggregatorInvitation{}).
		Where("business_id = ? AND aggregator_id = ? AND status = ?", businessID, aggregatorID, models.InvitationStatusAccepted).
		Update("status", models.InvitationStatusRevoked)

	// Log activity
	aggregatorRepo.CreateActivityLog(&models.AggregatorActivityLog{
		ID:           utility.GenerateUUID(),
		AggregatorID: aggregatorID,
		BusinessID:   businessID,
		Action:       models.ActivityBusinessRemoved,
		Details:      fmt.Sprintf("Removed business %s", business.CompanyName),
	}, db)

	return http.StatusOK, nil
}

func ListInvoicesByBusiness(aggregatorID, businessID string, page, size int, db *gorm.DB) ([]models.MinimalInvoiceDTO, *database.PaginationResponse, error) {
	business, err := aggregatorRepo.GetBusinessByIDForAggregator(db, aggregatorID, businessID)
	if err != nil || business == nil {
		return nil, nil, fmt.Errorf("business not found or not managed by this aggregator")
	}

	invoices, total, err := aggregatorRepo.GetInvoicesByAggregatorAndBusiness(db, aggregatorID, businessID, page, size)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to fetch invoices: %w", err)
	}

	result := make([]models.MinimalInvoiceDTO, 0, len(invoices))
	for _, inv := range invoices {
		result = append(result, models.MinimalInvoiceDTO{
			ID:            inv.ID,
			InvoiceNumber: inv.InvoiceNumber,
			IRN:           inv.IRN,
			Platform:      inv.Platform,
			CurrentStatus: inv.CurrentStatus,
			CreatedAt:     inv.CreatedAt,
		})
	}

	return result, buildPagination(page, size, total), nil
}

func ListAllInvoices(aggregatorID string, page, size int, db *gorm.DB) ([]models.MinimalInvoiceDTO, *database.PaginationResponse, error) {
	invoices, total, err := aggregatorRepo.GetAllInvoicesByAggregator(db, aggregatorID, page, size)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to fetch invoices: %w", err)
	}

	result := make([]models.MinimalInvoiceDTO, 0, len(invoices))
	for _, inv := range invoices {
		result = append(result, models.MinimalInvoiceDTO{
			ID:            inv.ID,
			InvoiceNumber: inv.InvoiceNumber,
			IRN:           inv.IRN,
			Platform:      inv.Platform,
			CurrentStatus: inv.CurrentStatus,
			CreatedAt:     inv.CreatedAt,
		})
	}

	return result, buildPagination(page, size, total), nil
}

func ListBulkUploadsByBusiness(aggregatorID, businessID string, page, size int, db *gorm.DB) ([]models.BulkUpload, *database.PaginationResponse, error) {
	business, err := aggregatorRepo.GetBusinessByIDForAggregator(db, aggregatorID, businessID)
	if err != nil || business == nil {
		return nil, nil, fmt.Errorf("business not found or not managed by this aggregator")
	}

	uploads, total, err := aggregatorRepo.GetBulkUploadsByAggregatorAndBusiness(db, aggregatorID, businessID, page, size)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to fetch bulk uploads: %w", err)
	}

	return uploads, buildPagination(page, size, total), nil
}

func ListAllBulkUploads(aggregatorID string, page, size int, db *gorm.DB) ([]models.BulkUpload, *database.PaginationResponse, error) {
	uploads, total, err := aggregatorRepo.GetAllBulkUploadsByAggregator(db, aggregatorID, page, size)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to fetch bulk uploads: %w", err)
	}

	return uploads, buildPagination(page, size, total), nil
}

func GetDashboard(aggregatorID string, db *gorm.DB) (*dtos.AggregatorDashboardDto, error) {
	totalBiz, pendingInvites, totalInvoices, totalBulkUploads, err := aggregatorRepo.GetDashboardStats(db, aggregatorID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch dashboard stats: %w", err)
	}

	return &dtos.AggregatorDashboardDto{
		TotalBusinesses:    totalBiz,
		PendingInvitations: pendingInvites,
		TotalInvoices:      totalInvoices,
		TotalBulkUploads:   totalBulkUploads,
	}, nil
}

func GetActivityLog(aggregatorID string, page, size int, db *gorm.DB) ([]dtos.AggregatorActivityLogDto, *database.PaginationResponse, error) {
	logs, total, err := aggregatorRepo.GetActivityLogs(db, aggregatorID, page, size)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to fetch activity logs: %w", err)
	}

	result := make([]dtos.AggregatorActivityLogDto, 0, len(logs))
	for _, l := range logs {
		result = append(result, dtos.AggregatorActivityLogDto{
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
