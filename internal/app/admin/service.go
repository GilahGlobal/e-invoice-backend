package admin

import (
	"einvoice-access-point/internal/app/invoice"
	"einvoice-access-point/internal/config"
	"einvoice-access-point/internal/data/database"
	"einvoice-access-point/internal/data/entities"
	"einvoice-access-point/internal/data/repositories"
	"einvoice-access-point/internal/middleware"
	"einvoice-access-point/internal/pkg/resend_email"
	"einvoice-access-point/internal/utility"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type Service struct {
	adminRepo        *repositories.AdminRepository
	authRepo         *repositories.AuthRepository
	businessRepo     *repositories.BusinessRepository
	aggregatorRepo   *repositories.AggregatorRepository
	invoiceRepo      *repositories.InvoiceRepository
	subscriptionRepo *repositories.SubscriptionRepository
	cfg              *config.Configuration
}

func NewServiceWithDB(prodDB, testDB database.DatabaseManager, cfg *config.Configuration) *Service {
	adminRepo := repositories.NewAdminRepository()
	authRepo := repositories.NewAuthRepository(prodDB, testDB)
	businessRepo := repositories.NewBusinessRepository(prodDB, testDB)
	aggregatorRepo := repositories.NewAggregatorRepository(prodDB, testDB)
	invoiceRepo := repositories.NewInvoiceRepository(prodDB, testDB)
	subscriptionRepo := repositories.NewSubscriptionRepository(prodDB, testDB)

	return &Service{
		adminRepo:        adminRepo,
		authRepo:         authRepo,
		businessRepo:     businessRepo,
		aggregatorRepo:   aggregatorRepo,
		invoiceRepo:      invoiceRepo,
		subscriptionRepo: subscriptionRepo,
		cfg:              cfg,
	}
}

func (s *Service) RegisterAdmin(req AdminRegisterDto, isSandbox bool) (int, error) {
	db := s.authRepo.GetDB(isSandbox)

	email := strings.ToLower(req.Email)
	existingAdmin, _ := s.adminRepo.GetAdminByEmail(db, email)
	if existingAdmin != nil {
		return http.StatusBadRequest, errors.New("admin with this email already exists")
	}

	passwordHash, err := utility.HashPassword(req.Password)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to hash password: %w", err)
	}

	var roleCount int64
	if err := db.DB().Model(&entities.Role{}).Where("id = ?", req.RoleID).Count(&roleCount).Error; err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to verify role: %w", err)
	}
	if roleCount == 0 {
		return http.StatusBadRequest, errors.New("invalid role_id provided")
	}

	admin := entities.Admin{
		ID:       utility.GenerateUUID(),
		Name:     req.Name,
		Email:    email,
		Password: passwordHash,
		RoleID:   req.RoleID,
	}

	err = s.adminRepo.CreateAdmin(&admin, db)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to create admin: %w", err)
	}

	go resend_email.SendAdminCreatedEmail(email, req.Password)

	return http.StatusCreated, nil
}

func (s *Service) GetRoles(db database.DatabaseManager) ([]entities.Role, error) {
	roles, err := s.adminRepo.GetAllRoles(db)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve roles: %v", err)
	}
	return roles, nil
}

func (s *Service) LoginAdmin(req AdminLoginRequestDto, isSandbox bool) (map[string]interface{}, int, error) {
	db := s.authRepo.GetDB(isSandbox)

	email := strings.ToLower(req.Email)
	admin, err := s.adminRepo.GetAdminByEmail(db, email)
	if err != nil || admin == nil {
		return nil, http.StatusBadRequest, errors.New("invalid credentials")
	}

	if !utility.CompareHash(req.Password, admin.Password) {
		return nil, http.StatusBadRequest, errors.New("invalid credentials")
	}

	if err := db.DB().Preload("Role").Where("id = ?", admin.ID).First(&admin).Error; err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to load admin role: %w", err)
	}

	tokenData, err := middleware.CreateAdminToken(*admin, isSandbox)
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to create token: %w", err)
	}

	tokens := map[string]string{
		"access_token": tokenData.AccessToken,
		"exp":          strconv.Itoa(int(tokenData.ExpiresAt.Unix())),
	}

	accessToken := entities.AccessToken{ID: tokenData.AccessUuid, OwnerID: admin.ID}
	err = s.authRepo.CreateAccessToken(&accessToken, isSandbox, tokens)
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to save token session: %w", err)
	}

	responseData := map[string]interface{}{
		"data": AdminResponse{
			ID:     admin.ID,
			Name:   admin.Name,
			Email:  admin.Email,
			RoleID: admin.RoleID.String(),
			Role:   admin.Role.Name,
		},
		"access_token": tokenData.AccessToken,
	}

	return responseData, http.StatusOK, nil
}

func mapBusinessesToDto(businesses []repositories.AdminBusinessQueryResult) []AdminBusinessResponseDto {
	var dtos []AdminBusinessResponseDto
	for _, b := range businesses {
		var serviceID, businessID string
		if b.ServiceID != nil {
			serviceID = *b.ServiceID
		}
		if b.BusinessID != nil {
			businessID = *b.BusinessID
		}

		lastUpload := ""
		if b.LastInvoiceUploadedAt != nil {
			lastUpload = *b.LastInvoiceUploadedAt
		}

		dtos = append(dtos, AdminBusinessResponseDto{
			ID:                    b.ID,
			Name:                  b.Name,
			ServiceID:             serviceID,
			TIN:                   b.TIN,
			Industry:              b.Industry,
			CreatedAt:             b.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			Email:                 b.Email,
			BusinessID:            businessID,
			PhoneNumber:           b.PhoneNumber,
			CompanyName:           b.CompanyName,
			BmpUploadSelected:     b.BmpUploadSelected,
			SubscribedPlan:        b.SubscribedPlan,
			TotalInvoicesUploaded: b.TotalInvoicesUploaded,
			Status:                b.AccStatus,
			LastInvoiceUploadedAt: lastUpload,
			AggregatorID:          b.AggregatorID,
			IsAggregator:          b.IsAggregator,
		})
	}
	return dtos
}

func buildPagination(page, size int, total int64) *database.PaginationResponse {
	return &database.PaginationResponse{
		CurrentPage:     page,
		PageCount:       size,
		TotalPagesCount: int(math.Ceil(float64(total) / float64(size))),
	}
}

func (s *Service) GetBusinesses(db database.DatabaseManager, search string, page, size int) ([]AdminBusinessResponseDto, *database.PaginationResponse, error) {
	businesses, total, err := s.businessRepo.ListAllBusinesses(db, search, page, size)
	if err != nil {
		return nil, nil, err
	}
	return mapBusinessesToDto(businesses), buildPagination(page, size, total), nil
}

func mapAggregatorsToDto(aggregators []repositories.AdminAggregatorQueryResult) []AdminAggregatorResponseDto {
	var dtos []AdminAggregatorResponseDto
	for _, a := range aggregators {
		lastUpload := ""
		if a.LastInvoiceUploadedAt != nil {
			lastUpload = *a.LastInvoiceUploadedAt
		}
		dtos = append(dtos, AdminAggregatorResponseDto{
			ID:                    a.ID,
			CompanyName:           a.CompanyName,
			Email:                 a.Email,
			TIN:                   a.TIN,
			Industry:              a.Industry,
			SubscribedPlan:        a.SubscribedPlan,
			CompaniesManaged:      a.CompaniesManaged,
			TotalInvoicesManaged:  a.TotalInvoicesManaged,
			LastInvoiceUploadedAt: lastUpload,
			Status:                a.AccStatus,
			CreatedAt:             a.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		})
	}
	return dtos
}

func (s *Service) GetAggregators(db database.DatabaseManager, search string, page, size int) ([]AdminAggregatorResponseDto, *database.PaginationResponse, error) {
	aggregators, total, err := s.aggregatorRepo.ListAllAggregators(db.DB(), search, page, size)
	if err != nil {
		return nil, nil, err
	}
	return mapAggregatorsToDto(aggregators), buildPagination(page, size, total), nil
}

func mapTransactionsToDto(transactions []entities.Transaction) []AdminTransactionDto {
	var dtos []AdminTransactionDto
	for _, txn := range transactions {
		dtos = append(dtos, AdminTransactionDto{
			ID:           txn.ID,
			BusinessID:   txn.BusinessID,
			AggregatorID: txn.AggregatorID,
			Reference:    txn.Reference,
			Provider:     txn.Provider,
			Status:       string(txn.Status),
			Amount:       txn.Amount,
			Currency:     txn.Currency,
			PlanID:       txn.PlanID,
			Plan:         txn.Plan,
			CreatedAt:    txn.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		})
	}
	return dtos
}

func (s *Service) GetInvoicesByBusiness(db database.DatabaseManager, businessID string, page, size int) ([]entities.MinimalInvoiceDTO, database.PaginationResponse, error) {
	paginationQuery := database.Pagination{
		Page:  page,
		Limit: size,
	}
	return s.invoiceRepo.FindMinimalInvoicesByBusinessID(db, businessID, paginationQuery)
}

func (s *Service) GetInvoicesByAggregator(db database.DatabaseManager, aggregatorID string, page, size int) ([]entities.Invoice, *database.PaginationResponse, error) {
	invoices, total, err := s.aggregatorRepo.GetAllInvoicesByAggregator(db.DB(), aggregatorID, page, size)
	if err != nil {
		return nil, nil, err
	}
	return invoices, buildPagination(page, size, total), nil
}

func (s *Service) GetTransactions(db database.DatabaseManager, page, size int) ([]AdminTransactionDto, *database.PaginationResponse, error) {
	transactions, total, err := s.subscriptionRepo.ListAllTransactions(db, page, size)
	if err != nil {
		return nil, nil, err
	}
	return mapTransactionsToDto(transactions), buildPagination(page, size, total), nil
}

func (s *Service) GetInvoiceStats(db database.DatabaseManager) (*entities.InvoiceStatsResponseData, error) {
	return s.invoiceRepo.GetInvoiceStats(db.DB(), nil, nil)
}

func (s *Service) GetBusinessStats(db database.DatabaseManager) (AdminBusinessStatsDto, error) {
	var dto AdminBusinessStatsDto
	totalBiz, totalAgg, err := s.businessRepo.GetSystemBusinessStats(db)
	if err != nil {
		return dto, err
	}
	dto.TotalBusinesses = totalBiz
	dto.TotalAggregators = totalAgg
	return dto, nil
}

func (s *Service) GetOverviewStats(db database.DatabaseManager, timeframe string, customStartDate string, customEndDate string) (*AdminOverviewStatsDto, error) {
	var startDate, endDate string

	switch timeframe {
	case "today":
		startDate = utility.GetTodayStart()
		endDate = utility.GetTodayEnd()
	case "7_days":
		startDate = utility.GetPastDaysStart(7)
		endDate = utility.GetTodayEnd()
	case "30_days":
		startDate = utility.GetPastDaysStart(30)
		endDate = utility.GetTodayEnd()
	case "custom":
		startDate = customStartDate
		endDate = customEndDate
	}

	result, err := s.adminRepo.GetOverviewStats(db.DB(), startDate, endDate)
	if err != nil {
		return nil, err
	}

	return &AdminOverviewStatsDto{
		TotalInvoices:          result.TotalInvoices,
		SuccessInvoices:        result.SuccessInvoices,
		PartialSuccessInvoices: result.PartialSuccessInvoices,
		FailedInvoices:         result.FailedInvoices,
		TotalCompanies:         result.TotalCompanies,
		TotalApiCalls:          result.TotalApiCalls,
		NewRegistrations:       result.NewRegistrations,
	}, nil
}

func (s *Service) CreateBusiness(req AdminCreateBusinessDto, isSandbox bool) error {
	db := s.authRepo.GetDB(isSandbox)

	exists, err := s.businessRepo.CheckBusinessExistsByEmailOrCompanyName(db, req.Email, req.CompanyName)
	if err != nil {
		return fmt.Errorf("failed to check existing business: %w", err)
	}
	if exists {
		return errors.New("business with this email or company name already exists")
	}

	generatedPassword := utility.RandomString(12)
	passwordHash, err := utility.HashPassword(generatedPassword)
	if err != nil {
		return err
	}

	business := entities.Business{
		ID:          utility.GenerateUUID(),
		CompanyName: req.CompanyName,
		Email:       strings.ToLower(req.Email),
		TIN:         req.TIN,
		Industry:    req.Industry,
		PhoneNumber: req.PhoneNumber,
		Password:    passwordHash,
		AccStatus:   0, // Active
	}

	if err := s.businessRepo.CreateBusiness(&business, db); err != nil {
		return err
	}

	go resend_email.SendAdminCreatedEmail(business.Email, generatedPassword)

	return nil
}

func (s *Service) UpdateBusiness(db database.DatabaseManager, businessID string, req AdminUpdateBusinessDto) error {
	business, err := s.businessRepo.GetBusinessByIDForAdmin(db, businessID)
	if err != nil {
		return err
	}

	if req.CompanyName != "" {
		business.CompanyName = req.CompanyName
	}
	if req.TIN != "" {
		business.TIN = req.TIN
	}
	if req.Industry != "" {
		business.Industry = req.Industry
	}
	if req.PhoneNumber != "" {
		business.PhoneNumber = req.PhoneNumber
	}
	if req.AccStatus != nil {
		business.AccStatus = *req.AccStatus
	}

	return s.businessRepo.UpdateAUser(business, db)
}

func (s *Service) GetBusinessDailyInvoiceStats(db database.DatabaseManager, businessID string) ([]repositories.AdminDailyInvoiceStatsResult, error) {
	startDate := utility.GetPastDaysStart(14)
	return s.adminRepo.GetDailyInvoiceStatsByBusiness(db.DB(), businessID, startDate)
}

type AdminDailyInvoiceStatsResult struct {
	Date                   string `json:"date"`
	SuccessInvoices        int64  `json:"success_invoices"`
	PartialSuccessInvoices int64  `json:"partial_success_invoices"`
	FailedInvoices         int64  `json:"failed_invoices"`
}

func mapDailyInvoiceStatsToDto(stats []repositories.AdminDailyInvoiceStatsResult) []AdminDailyInvoiceStatsDto {
	var dtos []AdminDailyInvoiceStatsDto
	for _, s := range stats {
		dtos = append(dtos, AdminDailyInvoiceStatsDto{
			Date:                   s.Date,
			SuccessInvoices:        s.SuccessInvoices,
			PartialSuccessInvoices: s.PartialSuccessInvoices,
			FailedInvoices:         s.FailedInvoices,
		})
	}
	return dtos
}

func (s *Service) GetBusinessDailyInvoiceStatsDto(db database.DatabaseManager, businessID string) ([]AdminDailyInvoiceStatsDto, error) {
	stats, err := s.adminRepo.GetDailyInvoiceStatsByBusiness(db.DB(), businessID, utility.GetPastDaysStart(14))
	if err != nil {
		return nil, err
	}
	return mapDailyInvoiceStatsToDto(stats), nil
}

func (s *Service) CreateAggregator(req AdminCreateAggregatorDto, isSandbox bool) error {
	db := s.authRepo.GetDB(isSandbox)

	exists, err := s.businessRepo.CheckBusinessExistsByEmailOrCompanyName(db, req.Email, req.CompanyName)
	if err != nil {
		return fmt.Errorf("failed to check existing aggregator: %w", err)
	}
	if exists {
		return errors.New("aggregator with this email or company name already exists")
	}

	generatedPassword := utility.RandomString(12)
	passwordHash, err := utility.HashPassword(generatedPassword)
	if err != nil {
		return err
	}

	aggregator := entities.Business{
		ID:           utility.GenerateUUID(),
		CompanyName:  req.CompanyName,
		Email:        strings.ToLower(req.Email),
		TIN:          req.TIN,
		Industry:     req.Industry,
		PhoneNumber:  req.PhoneNumber,
		Password:     passwordHash,
		IsAggregator: true,
		AccStatus:    0, // Active
	}

	if err := s.businessRepo.CreateBusiness(&aggregator, db); err != nil {
		return err
	}

	go resend_email.SendAdminCreatedEmail(aggregator.Email, generatedPassword)

	return nil
}

func (s *Service) UpdateAggregator(db database.DatabaseManager, aggregatorID string, req AdminUpdateBusinessDto) error {
	// Re-using the exact same logic as UpdateBusiness because Aggregator is just a Business with is_aggregator=true
	return s.UpdateBusiness(db, aggregatorID, req)
}

func (s *Service) GetAggregatorInfo(db database.DatabaseManager, aggregatorID string) (*AdminAggregatorInfoResponseDto, error) {
	// First fetch the base aggregator business with stats
	aggregators, _, err := s.aggregatorRepo.ListAllAggregators(db.DB(), "", 1, 1)
	if err != nil {
		return nil, err
	}

	// We have to filter by ID since ListAllAggregators doesn't take an ID
	var targetAgg *repositories.AdminAggregatorQueryResult
	for _, agg := range aggregators {
		if agg.ID == aggregatorID {
			targetAgg = &agg
			break
		}
	}

	// If not in the first page, let's fetch it via single query (we should write a specific query for this ideally)
	// but to save time let's fetch directly.
	if targetAgg == nil {
		business, err := s.businessRepo.GetBusinessByIDForAdmin(db, aggregatorID)
		if err != nil {
			return nil, err
		}
		targetAgg = &repositories.AdminAggregatorQueryResult{
			Business: *business,
		}
		// Populate stats manually
		_, pendingInvites, totalInvoices, _, _ := s.aggregatorRepo.GetDashboardStats(db.DB(), aggregatorID)
		targetAgg.TotalInvoicesManaged = totalInvoices
		_ = pendingInvites // we'll use this below
	}

	_, pendingInvites, _, _, _ := s.aggregatorRepo.GetDashboardStats(db.DB(), aggregatorID)

	aggDto := mapAggregatorsToDto([]repositories.AdminAggregatorQueryResult{*targetAgg})[0]

	companyStats, err := s.aggregatorRepo.GetAggregatorCompanyStats(db.DB(), aggregatorID)
	if err != nil {
		return nil, err
	}

	var companies []AggregatorCompanyDto
	for _, c := range companyStats {
		companies = append(companies, AggregatorCompanyDto{
			ID:               c.ID,
			CompanyName:      c.CompanyName,
			TIN:              c.TIN,
			InvoicesUploaded: c.InvoicesUploaded,
		})
	}

	return &AdminAggregatorInfoResponseDto{
		Aggregator: aggDto,
		Stats: AggregatorStatsDto{
			CompaniesManaged:   targetAgg.CompaniesManaged,
			InvoicesUploaded:   targetAgg.TotalInvoicesManaged,
			PendingInvitations: pendingInvites,
		},
		Companies: companies,
	}, nil
}

func (s *Service) GetAggregatorInvitations(db database.DatabaseManager, aggregatorID string) ([]AdminAggregatorInvitationDto, error) {
	invitations, err := s.aggregatorRepo.ListPendingInvitationsForAdmin(db.DB(), aggregatorID)
	if err != nil {
		return nil, err
	}

	var dtos []AdminAggregatorInvitationDto
	for _, i := range invitations {
		dtos = append(dtos, AdminAggregatorInvitationDto{
			ID:          i.ID,
			CompanyName: i.CompanyName,
			Industry:    i.Industry,
			Status:      i.Status,
			CreatedAt:   i.CreatedAt,
		})
	}
	return dtos, nil
}

type AdminBusinessAggregatorHistoryDto struct {
	ID           string `json:"id"`
	AggregatorID string `json:"aggregator_id"`
	Action       string `json:"action"`
	CreatedAt    string `json:"created_at"`
}

type AdminBusinessAggregatorInfoResponseDto struct {
	CurrentAggregatorID string                              `json:"current_aggregator_id"`
	History             []AdminBusinessAggregatorHistoryDto `json:"history"`
}

func (s *Service) GetBusinessAggregatorInfo(db database.DatabaseManager, businessID string) (*AdminBusinessAggregatorInfoResponseDto, error) {
	business, err := s.businessRepo.GetBusinessByIDForAdmin(db, businessID)
	if err != nil {
		return nil, err
	}

	histories, err := s.businessRepo.GetBusinessAggregatorHistory(db, businessID)
	if err != nil {
		return nil, err
	}

	var dtos []AdminBusinessAggregatorHistoryDto
	for _, h := range histories {
		dtos = append(dtos, AdminBusinessAggregatorHistoryDto{
			ID:           h.ID.String(),
			AggregatorID: h.AggregatorID,
			Action:       h.Action,
			CreatedAt:    h.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		})
	}

	currentAggregatorID := ""
	if business.AggregatorID != nil {
		currentAggregatorID = *business.AggregatorID
	}

	return &AdminBusinessAggregatorInfoResponseDto{
		CurrentAggregatorID: currentAggregatorID,
		History:             dtos,
	}, nil
}

func (s *Service) GetBusinessInfo(db database.DatabaseManager, businessID string) (*AdminBusinessInfoResponseDto, error) {
	businessQuery, err := s.businessRepo.GetBusinessDetailsByIDForAdmin(db, businessID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("business not found")
		}
		return nil, err
	}

	businessDto := mapBusinessesToDto([]repositories.AdminBusinessQueryResult{*businessQuery})[0]

	var aggDto *AdminAggregatorResponseDto
	if businessQuery.AggregatorID != nil && *businessQuery.AggregatorID != "" {
		agg, err := s.aggregatorRepo.GetAggregatorByID(db.DB(), *businessQuery.AggregatorID)
		if err == nil && agg != nil {
			_, _, totalInvoices, _, _ := s.aggregatorRepo.GetDashboardStats(db.DB(), agg.ID)
			aggDto = &AdminAggregatorResponseDto{
				ID:                   agg.ID,
				CompanyName:          agg.CompanyName,
				Email:                agg.Email,
				TIN:                  agg.TIN,
				Industry:             agg.Industry,
				TotalInvoicesManaged: totalInvoices,
				Status:               agg.AccStatus,
				CreatedAt:            agg.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			}
		}
	}

	return &AdminBusinessInfoResponseDto{
		Business:   businessDto,
		Aggregator: aggDto,
	}, nil
}

// GetBusinessInvoiceDetail fetches the full details of a specific invoice for a specific business.
// The businessID parameter is the business UUID (id).
// The invoiceID parameter can be the invoice UUID (id), invoice_number, or irn.
func (s *Service) GetBusinessInvoiceDetail(db database.DatabaseManager, businessID, invoiceID string) (*entities.Invoice, error) {
	business, err := s.businessRepo.GetBusinessByID(db, businessID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("business not found")
		}
		return nil, err
	}

	invoice, err := s.invoiceRepo.FindInvoiceByBusinessAndIdentifier(db, business.ID, invoiceID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("invoice not found for this business")
		}
		return nil, err
	}
	return invoice, nil
}

// DownloadInvoice downloads and decrypts an invoice by invoice ID, invoice number, or IRN.
func (s *Service) DownloadInvoice(db database.DatabaseManager, identifier string, isSandbox bool) (*string, *string, error) {
	irn := strings.TrimSpace(identifier)

	// Attempt to locate invoice in DB by id, invoice_number, or irn
	inv, err := s.invoiceRepo.FindInvoiceByIdentifier(db, identifier)
	if err == nil && inv != nil {
		if strings.TrimSpace(inv.IRN) != "" {
			irn = strings.TrimSpace(inv.IRN)
		} else {
			errDetails := "invoice does not have an IRN (it may not have been confirmed or transmitted yet)"
			return nil, &errDetails, errors.New(errDetails)
		}
	}

	if irn == "" {
		errDetails := "IRN is required to download invoice"
		return nil, &errDetails, errors.New(errDetails)
	}

	return invoice.DownloadAndDecryptInvoice(irn, isSandbox)
}

func mapSubscriptionItemToDto(item repositories.AdminSubscriptionItem, now time.Time) AdminSubscriptionResponseDto {
	daysUntilExpiry := int(math.Ceil(item.NextBillingDate.Sub(now).Hours() / 24))
	statusLabel := "active"
	if !item.IsActive || item.NextBillingDate.Before(now) {
		statusLabel = "expired"
	} else if daysUntilExpiry <= 7 {
		statusLabel = "expiring_soon"
	}

	lastReminder := ""
	if item.LastReminderSentAt != nil {
		lastReminder = item.LastReminderSentAt.Format(time.RFC3339)
	}

	return AdminSubscriptionResponseDto{
		ID:                 item.ID,
		BusinessID:         item.BusinessID,
		BusinessName:       item.BusinessName,
		BusinessEmail:      item.BusinessEmail,
		BusinessCompany:    item.BusinessCompany,
		BusinessTIN:        item.BusinessTIN,
		BusinessPhone:      item.BusinessPhone,
		AggregatorID:       item.AggregatorID,
		AggregatorName:     item.AggregatorName,
		AggregatorEmail:    item.AggregatorEmail,
		AggregatorCompany:  item.AggregatorCompany,
		PlanID:             item.PlanID,
		Plan:               item.Plan,
		IsActive:           item.IsActive,
		TotalInvoices:      item.TotalInvoices,
		RemainingInvoices:  item.RemainingInvoices,
		UsedInvoices:       item.UsedInvoices,
		NextBillingDate:    item.NextBillingDate.Format(time.RFC3339),
		DaysUntilExpiry:    daysUntilExpiry,
		StatusLabel:        statusLabel,
		RemindersSentCount: item.RemindersSentCount,
		LastReminderSentAt: lastReminder,
		CreatedAt:          item.CreatedAt.Format(time.RFC3339),
		UpdatedAt:          item.UpdatedAt.Format(time.RFC3339),
	}
}

func (s *Service) GetSubscriptions(db database.DatabaseManager, filter repositories.AdminSubscriptionFilter, page, size int) ([]AdminSubscriptionResponseDto, *database.PaginationResponse, error) {
	items, total, err := s.subscriptionRepo.ListSubscriptions(db, filter, page, size)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list subscriptions: %w", err)
	}

	now := time.Now().UTC()
	dtos := make([]AdminSubscriptionResponseDto, 0, len(items))
	for _, item := range items {
		dtos = append(dtos, mapSubscriptionItemToDto(item, now))
	}

	return dtos, buildPagination(page, size, total), nil
}

func (s *Service) GetSubscriptionByID(db database.DatabaseManager, subscriptionID string) (*AdminSubscriptionDetailDto, error) {
	item, err := s.subscriptionRepo.GetSubscriptionByIDWithDetails(db, subscriptionID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch subscription: %w", err)
	}
	if item == nil {
		return nil, fmt.Errorf("subscription not found")
	}

	now := time.Now().UTC()
	baseDto := mapSubscriptionItemToDto(*item, now)

	reminderLogs, err := s.subscriptionRepo.GetReminderLogsBySubscriptionID(db, subscriptionID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch reminder logs: %w", err)
	}

	var transactions []entities.Transaction
	_ = db.DB().Where("business_id = ?", item.BusinessID).Order("created_at desc").Limit(20).Find(&transactions).Error

	return &AdminSubscriptionDetailDto{
		AdminSubscriptionResponseDto: baseDto,
		ReminderLogs:                 reminderLogs,
		Transactions:                 transactions,
	}, nil
}

func (s *Service) GetSubscriptionStats(db database.DatabaseManager) (*AdminSubscriptionStatsDto, error) {
	stats, err := s.subscriptionRepo.GetSubscriptionStats(db)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch subscription stats: %w", err)
	}

	return &AdminSubscriptionStatsDto{
		TotalSubscriptions:   stats.TotalSubscriptions,
		ActiveSubscriptions:  stats.ActiveSubscriptions,
		ExpiringIn3Days:      stats.ExpiringIn3Days,
		ExpiringIn7Days:      stats.ExpiringIn7Days,
		ExpiringIn14Days:     stats.ExpiringIn14Days,
		ExpiredSubscriptions: stats.ExpiredSubscriptions,
	}, nil
}

func mapPlanToAdminDto(plan entities.SubscriptionPlan, activeCount, totalCount int64) AdminPlanResponseDto {
	return AdminPlanResponseDto{
		ID:                     plan.ID,
		Name:                   plan.Name,
		Amount:                 plan.Amount,
		IsActive:               plan.IsActive,
		TotalInvoices:          plan.TotalInvoices,
		BillingCycle:           plan.BillingCycle,
		ReminderDaysBefore:     plan.GetReminderDays(),
		ActiveSubscribersCount: activeCount,
		TotalSubscribersCount:  totalCount,
		CreatedAt:              plan.CreatedAt.Format(time.RFC3339),
		UpdatedAt:              plan.UpdatedAt.Format(time.RFC3339),
	}
}

func (s *Service) GetSubscriptionPlans(db database.DatabaseManager) ([]AdminPlanResponseDto, error) {
	items, err := s.subscriptionRepo.GetPlansWithSubscriberCounts(db)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch subscription plans: %w", err)
	}

	result := make([]AdminPlanResponseDto, 0, len(items))
	for _, it := range items {
		result = append(result, mapPlanToAdminDto(it.SubscriptionPlan, it.ActiveSubscribersCount, it.TotalSubscribersCount))
	}
	return result, nil
}

func (s *Service) GetSubscriptionPlanByID(db database.DatabaseManager, planID string) (*AdminPlanResponseDto, error) {
	plan, err := s.subscriptionRepo.GetPlanByID(planID, db)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch plan: %w", err)
	}
	if plan == nil {
		return nil, fmt.Errorf("subscription plan not found")
	}

	activeCount, _ := s.subscriptionRepo.CountSubscriptionsByPlanID(plan.ID, true, db)
	totalCount, _ := s.subscriptionRepo.CountSubscriptionsByPlanID(plan.ID, false, db)

	dto := mapPlanToAdminDto(*plan, activeCount, totalCount)
	return &dto, nil
}

func (s *Service) CreateSubscriptionPlan(db database.DatabaseManager, req AdminCreatePlanDto) (*AdminPlanResponseDto, error) {
	planName := strings.TrimSpace(req.Name)
	if planName == "" {
		return nil, fmt.Errorf("plan name is required")
	}

	existingPlan, err := s.subscriptionRepo.GetPlanByName(planName, db)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing plan: %w", err)
	}
	if existingPlan != nil {
		return nil, fmt.Errorf("a subscription plan with name %q already exists", planName)
	}

	reminderDays := req.ReminderDaysBefore
	if len(reminderDays) == 0 {
		reminderDays = []int{7, 3, 1}
	} else {
		seen := make(map[int]bool)
		var cleaned []int
		for _, d := range reminderDays {
			if d > 0 && !seen[d] {
				seen[d] = true
				cleaned = append(cleaned, d)
			}
		}
		if len(cleaned) == 0 {
			cleaned = []int{7, 3, 1}
		} else {
			sort.Sort(sort.Reverse(sort.IntSlice(cleaned)))
		}
		reminderDays = cleaned
	}
	reminderDaysBytes, _ := json.Marshal(reminderDays)

	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	plan := &entities.SubscriptionPlan{
		ID:                 utility.GenerateUUID(),
		Name:               planName,
		Amount:             req.Amount,
		IsActive:           isActive,
		TotalInvoices:      req.TotalInvoices,
		BillingCycle:       req.BillingCycle,
		ReminderDaysBefore: datatypes.JSON(reminderDaysBytes),
	}

	if err := s.subscriptionRepo.CreatePlan(plan, db); err != nil {
		return nil, fmt.Errorf("failed to create subscription plan: %w", err)
	}

	dto := mapPlanToAdminDto(*plan, 0, 0)
	return &dto, nil
}

func (s *Service) UpdateSubscriptionPlan(db database.DatabaseManager, planID string, req AdminUpdatePlanDto) (*AdminPlanResponseDto, error) {
	plan, err := s.subscriptionRepo.GetPlanByID(planID, db)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch plan: %w", err)
	}
	if plan == nil {
		return nil, fmt.Errorf("subscription plan not found")
	}

	if req.Name != nil && strings.TrimSpace(*req.Name) != "" {
		newName := strings.TrimSpace(*req.Name)
		if !strings.EqualFold(newName, plan.Name) {
			existing, err := s.subscriptionRepo.GetPlanByName(newName, db)
			if err != nil {
				return nil, fmt.Errorf("failed to check plan name uniqueness: %w", err)
			}
			if existing != nil && existing.ID != plan.ID {
				return nil, fmt.Errorf("a subscription plan with name %q already exists", newName)
			}
			plan.Name = newName
		}
	}

	if req.Amount != nil && *req.Amount > 0 {
		plan.Amount = *req.Amount
	}
	if req.TotalInvoices != nil && *req.TotalInvoices > 0 {
		plan.TotalInvoices = *req.TotalInvoices
	}
	if req.BillingCycle != nil && *req.BillingCycle > 0 {
		plan.BillingCycle = *req.BillingCycle
	}
	if req.IsActive != nil {
		plan.IsActive = *req.IsActive
	}

	if len(req.ReminderDaysBefore) > 0 {
		seen := make(map[int]bool)
		var cleaned []int
		for _, d := range req.ReminderDaysBefore {
			if d > 0 && !seen[d] {
				seen[d] = true
				cleaned = append(cleaned, d)
			}
		}
		if len(cleaned) > 0 {
			sort.Sort(sort.Reverse(sort.IntSlice(cleaned)))
			b, _ := json.Marshal(cleaned)
			plan.ReminderDaysBefore = datatypes.JSON(b)
		}
	}

	if err := s.subscriptionRepo.UpdatePlan(plan, db); err != nil {
		return nil, fmt.Errorf("failed to update subscription plan: %w", err)
	}

	activeCount, _ := s.subscriptionRepo.CountSubscriptionsByPlanID(plan.ID, true, db)
	totalCount, _ := s.subscriptionRepo.CountSubscriptionsByPlanID(plan.ID, false, db)

	dto := mapPlanToAdminDto(*plan, activeCount, totalCount)
	return &dto, nil
}

func (s *Service) ToggleSubscriptionPlanStatus(db database.DatabaseManager, planID string) (string, error) {
	plan, err := s.subscriptionRepo.GetPlanByID(planID, db)
	if err != nil {
		return "", fmt.Errorf("failed to fetch plan: %w", err)
	}
	if plan == nil {
		return "", fmt.Errorf("subscription plan not found")
	}

	targetStatus := !plan.IsActive
	if err := s.subscriptionRepo.UpdatePlanStatus(plan.ID, targetStatus, db); err != nil {
		return "", fmt.Errorf("failed to update plan status: %w", err)
	}

	action := "activated"
	if !targetStatus {
		action = "deactivated"
	}
	return action, nil
}

func (s *Service) DeleteSubscriptionPlan(db database.DatabaseManager, planID string, force bool) error {
	plan, err := s.subscriptionRepo.GetPlanByID(planID, db)
	if err != nil {
		return fmt.Errorf("failed to fetch plan: %w", err)
	}
	if plan == nil {
		return fmt.Errorf("subscription plan not found")
	}

	activeCount, err := s.subscriptionRepo.CountSubscriptionsByPlanID(plan.ID, true, db)
	if err != nil {
		return fmt.Errorf("failed to verify active subscriptions: %w", err)
	}

	if activeCount > 0 && !force {
		return fmt.Errorf("cannot delete plan %q: %d active subscription(s) are currently running on this plan. Deactivate the plan instead so existing subscriptions can complete their cycle", plan.Name, activeCount)
	}

	if err := s.subscriptionRepo.DeletePlan(plan.ID, db); err != nil {
		return fmt.Errorf("failed to delete subscription plan: %w", err)
	}

	return nil
}
