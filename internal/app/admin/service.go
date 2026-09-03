package admin

import (
	"einvoice-access-point/internal/config"
	"einvoice-access-point/internal/data/database"
	"einvoice-access-point/internal/data/entities"
	"einvoice-access-point/internal/data/repositories"
	"einvoice-access-point/internal/middleware"
	"einvoice-access-point/internal/pkg/resend_email"
	"einvoice-access-point/internal/utility"
	"errors"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
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
