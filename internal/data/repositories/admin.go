package repositories

import (
	"einvoice-access-point/internal/data/database"
	"einvoice-access-point/internal/data/entities"
	"errors"

	"gorm.io/gorm"
)

type AdminRepository struct{}

func NewAdminRepository() *AdminRepository {
	return &AdminRepository{}
}

func (r *AdminRepository) CreateAdmin(admin *entities.Admin, db database.DatabaseManager) error {
	return db.CreateOneRecord(admin)
}

func (r *AdminRepository) GetAdminByEmail(db database.DatabaseManager, email string) (*entities.Admin, error) {
	var admin entities.Admin
	err, nilErr := db.SelectOneFromDb(&admin, "email = ?", email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		return nil, err
	}
	if nilErr != nil {
		return nil, nilErr
	}
	return &admin, nil
}

func (r *AdminRepository) GetAdminByID(db database.DatabaseManager, id string) (*entities.Admin, error) {
	var admin entities.Admin
	err, nilErr := db.SelectOneFromDb(&admin, "id = ?", id)
	if err != nil {
		return nil, err
	}
	if nilErr != nil {
		return nil, nilErr
	}
	return &admin, nil
}

func (r *AdminRepository) CountAdmins(db database.DatabaseManager) (int64, error) {
	var count int64
	err := db.DB().Model(&entities.Admin{}).Count(&count).Error
	if err != nil {
		return 0, err
	}
	return count, nil
}

func (r *AdminRepository) GetAllRoles(db database.DatabaseManager) ([]entities.Role, error) {
	var roles []entities.Role
	err := db.SelectAllFromDb("created_at asc", "", &roles, nil)
	if err != nil {
		return nil, err
	}
	return roles, nil
}

type AdminOverviewStatsResult struct {
	TotalInvoices          int64
	SuccessInvoices        int64
	PartialSuccessInvoices int64
	FailedInvoices         int64
	TotalCompanies         int64
	TotalApiCalls          int64
	NewRegistrations       int64
}

func (r *AdminRepository) GetOverviewStats(db *gorm.DB, startDate, endDate string) (AdminOverviewStatsResult, error) {
	var result AdminOverviewStatsResult

	// Invoices
	invoiceQuery := db.Table("invoices")
	if startDate != "" && endDate != "" {
		invoiceQuery = invoiceQuery.Where("created_at >= ? AND created_at <= ?", startDate, endDate)
	}
	invoiceQuery = invoiceQuery.Joins(`LEFT JOIN LATERAL (
		SELECT entry->>'status' AS current_step_status
		FROM jsonb_array_elements(invoices.status_history) AS entry
		WHERE entry->>'step' = invoices.current_status
		ORDER BY (entry->>'timestamp')::timestamptz DESC
		LIMIT 1
	) AS current_step ON true`)

	invoiceQuery.Select(`
		COUNT(*) as total_invoices,
		COALESCE(SUM(CASE WHEN current_status = 'confirmed_invoice' AND current_step_status = 'success' THEN 1 ELSE 0 END), 0) as success_invoices,
		COALESCE(SUM(CASE WHEN (current_status = 'confirmed_invoice' AND current_step_status != 'success') OR current_status = 'transmitted_invoice' OR (current_status = 'signed_invoice' AND current_step_status = 'success') THEN 1 ELSE 0 END), 0) as partial_success_invoices,
		COALESCE(SUM(CASE WHEN (current_status = 'signed_invoice' AND current_step_status != 'success') OR current_status NOT IN ('confirmed_invoice', 'signed_invoice', 'transmitted_invoice') THEN 1 ELSE 0 END), 0) as failed_invoices
	`).Scan(&result)

	// Companies
	companyQuery := db.Model(&entities.Business{})
	if startDate != "" && endDate != "" {
		companyQuery = companyQuery.Where("created_at >= ? AND created_at <= ?", startDate, endDate)
	}
	companyQuery.Count(&result.TotalCompanies)

	// API Calls
	apiCallQuery := db.Model(&entities.ApiLog{})
	if startDate != "" && endDate != "" {
		apiCallQuery = apiCallQuery.Where("created_at >= ? AND created_at <= ?", startDate, endDate)
	}
	apiCallQuery.Count(&result.TotalApiCalls)

	// New Registrations (Businesses created in timeframe)
	// Since TotalCompanies already counts this if timeframe is applied, NewRegistrations = TotalCompanies in that timeframe
	result.NewRegistrations = result.TotalCompanies

	return result, nil
}

type AdminDailyInvoiceStatsResult struct {
	Date                   string
	SuccessInvoices        int64
	PartialSuccessInvoices int64
	FailedInvoices         int64
}

func (r *AdminRepository) GetDailyInvoiceStatsByBusiness(db *gorm.DB, businessID string, startDate string) ([]AdminDailyInvoiceStatsResult, error) {
	var results []AdminDailyInvoiceStatsResult

	// Using PostgreSQL DATE function for grouping
	query := `
		SELECT 
			DATE(invoices.created_at) as date,
			COALESCE(SUM(CASE WHEN current_status = 'confirmed_invoice' AND current_step_status = 'success' THEN 1 ELSE 0 END), 0) as success_invoices,
			COALESCE(SUM(CASE WHEN (current_status = 'confirmed_invoice' AND current_step_status != 'success') OR current_status = 'transmitted_invoice' OR (current_status = 'signed_invoice' AND current_step_status = 'success') THEN 1 ELSE 0 END), 0) as partial_success_invoices,
			COALESCE(SUM(CASE WHEN (current_status = 'signed_invoice' AND current_step_status != 'success') OR current_status NOT IN ('confirmed_invoice', 'signed_invoice', 'transmitted_invoice') THEN 1 ELSE 0 END), 0) as failed_invoices
		FROM invoices 
		LEFT JOIN LATERAL (
			SELECT entry->>'status' AS current_step_status
			FROM jsonb_array_elements(invoices.status_history) AS entry
			WHERE entry->>'step' = invoices.current_status
			ORDER BY (entry->>'timestamp')::timestamptz DESC
			LIMIT 1
		) AS current_step ON true
		WHERE business_id = ? AND invoices.created_at >= ?
		GROUP BY DATE(invoices.created_at)
		ORDER BY DATE(invoices.created_at) DESC
	`

	err := db.Raw(query, businessID, startDate).Scan(&results).Error
	return results, err
}
