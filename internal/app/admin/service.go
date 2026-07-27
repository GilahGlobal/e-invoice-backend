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

type Service interface {
	RegisterAdmin(req AdminRegisterDto, isSandbox bool) (int, error)
	LoginAdmin(req AdminLoginRequestDto, isSandbox bool) (map[string]interface{}, int, error)
	GetBusinesses(db database.DatabaseManager, search string, page, size int) ([]AdminBusinessResponseDto, *database.PaginationResponse, error)
	GetAggregators(db database.DatabaseManager, search string, page, size int) ([]AdminBusinessResponseDto, *database.PaginationResponse, error)
	GetInvoicesByBusiness(db database.DatabaseManager, businessID string, page, size int) ([]entities.MinimalInvoiceDTO, database.PaginationResponse, error)
	GetInvoicesByAggregator(db database.DatabaseManager, aggregatorID string, page, size int) ([]entities.Invoice, *database.PaginationResponse, error)
	GetTransactions(db database.DatabaseManager, page, size int) ([]AdminTransactionDto, *database.PaginationResponse, error)
	GetInvoiceStats(db database.DatabaseManager) (*entities.InvoiceStatsResponseData, error)
	GetBusinessStats(db database.DatabaseManager) (AdminBusinessStatsDto, error)
}

type service struct {
	adminRepo        repositories.AdminRepository
	authRepo         repositories.AuthRepository
	businessRepo     repositories.BusinessRepository
	aggregatorRepo   repositories.AggregatorRepository
	invoiceRepo      repositories.InvoiceRepository
	subscriptionRepo repositories.SubscriptionRepository
	cfg              *config.Configuration
}

func NewServiceWithDB(prodDB, testDB database.DatabaseManager, cfg *config.Configuration) Service {
	adminRepo := repositories.NewAdminRepository()
	authRepo := repositories.NewAuthRepository(prodDB, testDB)
	businessRepo := repositories.NewBusinessRepository(prodDB, testDB)
	aggregatorRepo := repositories.NewAggregatorRepository(prodDB, testDB)
	invoiceRepo := repositories.NewInvoiceRepository(prodDB, testDB)
	subscriptionRepo := repositories.NewSubscriptionRepository(prodDB, testDB)

	return &service{
		adminRepo:        adminRepo,
		authRepo:         authRepo,
		businessRepo:     businessRepo,
		aggregatorRepo:   aggregatorRepo,
		invoiceRepo:      invoiceRepo,
		subscriptionRepo: subscriptionRepo,
		cfg:              cfg,
	}
}

func (s *service) RegisterAdmin(req AdminRegisterDto, isSandbox bool) (int, error) {
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

	admin := entities.Admin{
		ID:       utility.GenerateUUID(),
		Name:     req.Name,
		Email:    email,
		Password: passwordHash,
		Role:     req.Role,
	}

	err = s.adminRepo.CreateAdmin(&admin, db)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to create admin: %w", err)
	}

	go resend_email.SendAdminCreatedEmail(email, req.Password)

	return http.StatusCreated, nil
}

func (s *service) LoginAdmin(req AdminLoginRequestDto, isSandbox bool) (map[string]interface{}, int, error) {
	db := s.authRepo.GetDB(isSandbox)

	email := strings.ToLower(req.Email)
	admin, err := s.adminRepo.GetAdminByEmail(db, email)
	if err != nil || admin == nil {
		return nil, http.StatusBadRequest, errors.New("invalid credentials")
	}

	if !utility.CompareHash(req.Password, admin.Password) {
		return nil, http.StatusBadRequest, errors.New("invalid credentials")
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
			ID:    admin.ID,
			Name:  admin.Name,
			Email: admin.Email,
			Role:  admin.Role,
		},
		"access_token": tokenData.AccessToken,
	}

	return responseData, http.StatusOK, nil
}

func mapBusinessesToDto(businesses []entities.Business) []AdminBusinessResponseDto {
	var dtos []AdminBusinessResponseDto
	for _, b := range businesses {
		var serviceID, businessID string
		if b.ServiceID != nil {
			serviceID = *b.ServiceID
		}
		if b.BusinessID != nil {
			businessID = *b.BusinessID
		}
		dtos = append(dtos, AdminBusinessResponseDto{
			ID:                b.ID,
			Name:              b.Name,
			ServiceID:         serviceID,
			TIN:               b.TIN,
			CreatedAt:         b.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			Email:             b.Email,
			BusinessID:        businessID,
			PhoneNumber:       b.PhoneNumber,
			CompanyName:       b.CompanyName,
			BmpUploadSelected: b.BmpUploadSelected,
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

func (s *service) GetBusinesses(db database.DatabaseManager, search string, page, size int) ([]AdminBusinessResponseDto, *database.PaginationResponse, error) {
	businesses, total, err := s.businessRepo.ListAllBusinesses(db, search, page, size)
	if err != nil {
		return nil, nil, err
	}
	return mapBusinessesToDto(businesses), buildPagination(page, size, total), nil
}

func (s *service) GetAggregators(db database.DatabaseManager, search string, page, size int) ([]AdminBusinessResponseDto, *database.PaginationResponse, error) {
	aggregators, total, err := s.aggregatorRepo.ListAllAggregators(db.DB(), search, page, size)
	if err != nil {
		return nil, nil, err
	}
	return mapBusinessesToDto(aggregators), buildPagination(page, size, total), nil
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

func (s *service) GetInvoicesByBusiness(db database.DatabaseManager, businessID string, page, size int) ([]entities.MinimalInvoiceDTO, database.PaginationResponse, error) {
	paginationQuery := database.Pagination{
		Page:  page,
		Limit: size,
	}
	return s.invoiceRepo.FindMinimalInvoicesByBusinessID(db, businessID, paginationQuery)
}

func (s *service) GetInvoicesByAggregator(db database.DatabaseManager, aggregatorID string, page, size int) ([]entities.Invoice, *database.PaginationResponse, error) {
	invoices, total, err := s.aggregatorRepo.GetAllInvoicesByAggregator(db.DB(), aggregatorID, page, size)
	if err != nil {
		return nil, nil, err
	}
	return invoices, buildPagination(page, size, total), nil
}

func (s *service) GetTransactions(db database.DatabaseManager, page, size int) ([]AdminTransactionDto, *database.PaginationResponse, error) {
	transactions, total, err := s.subscriptionRepo.ListAllTransactions(db, page, size)
	if err != nil {
		return nil, nil, err
	}
	return mapTransactionsToDto(transactions), buildPagination(page, size, total), nil
}

func (s *service) GetInvoiceStats(db database.DatabaseManager) (*entities.InvoiceStatsResponseData, error) {
	return s.invoiceRepo.GetInvoiceStats(db.DB(), nil, nil)
}

func (s *service) GetBusinessStats(db database.DatabaseManager) (AdminBusinessStatsDto, error) {
	var dto AdminBusinessStatsDto
	totalBiz, totalAgg, err := s.businessRepo.GetSystemBusinessStats(db)
	if err != nil {
		return dto, err
	}
	dto.TotalBusinesses = totalBiz
	dto.TotalAggregators = totalAgg
	return dto, nil
}
