package admin

import (
	"einvoice-access-point/internal/apperror"
	"einvoice-access-point/internal/config"
	"einvoice-access-point/internal/data/database"
	"einvoice-access-point/internal/data/dbinit"
	"einvoice-access-point/internal/middleware"
	"einvoice-access-point/internal/utility"
	"strconv"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
)

type Handler struct {
	validator *validator.Validate
	logger    *utility.Logger
	svc       *Service
	Db        *database.Database
	TestDB    *database.Database
}

func NewHandler(validator *validator.Validate, logger *utility.Logger, cfg *config.Configuration, db, testDb *database.Database) *Handler {
	prodDB := dbinit.InitDB(db.Postgresql.DB(), false)
	testDB := dbinit.InitDB(testDb.Postgresql.DB(), false)
	svc := NewServiceWithDB(prodDB, testDB, cfg)
	return &Handler{
		validator: validator,
		logger:    logger,
		svc:       svc,
		Db:        db,
		TestDB:    testDb,
	}
}

// @Summary Retrieve all roles
// @Description Retrieve a list of all roles available for admin users
// @Tags Admin Roles
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} RoleListResponseDto "Roles retrieved successfully"
// @Failure 401 {object} apperror.AppError "Unauthorized"
// @Failure 500 {object} apperror.AppError "Internal server error"
// @Router /admin/roles [get]
func (h *Handler) GetRoles(c *fiber.Ctx) error {
	rawDb, err := middleware.GetDatabase(c)
	if err != nil {
		return apperror.New(fiber.StatusInternalServerError, "error", err.Error(), err, nil)
	}
	db := dbinit.InitDB(rawDb, false)

	roles, err := h.svc.GetRoles(db)
	if err != nil {
		return apperror.New(fiber.StatusInternalServerError, "error", "Failed to retrieve roles", err, nil)
	}

	var rolesData []RoleResponseDto
	for _, role := range roles {
		rolesData = append(rolesData, RoleResponseDto{
			ID:          role.ID,
			Name:        role.Name,
			Description: role.Description,
		})
	}

	rd := utility.BuildSuccessResponse(fiber.StatusOK, "success", "Roles retrieved successfully", rolesData)
	return c.Status(fiber.StatusOK).JSON(rd)
}

// @Summary Register Admin
// @Description Register a new admin (Requires SuperAdmin privileges)
// @Tags Admin Auth
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param data body AdminRegisterDto true "Register request payload"
// @Success 201 {object} utility.Response "Admin created successfully"
// @Failure 400 {object} apperror.AppError "Bad request, validation failed"
// @Failure 401 {object} apperror.AppError "Unauthorized"
// @Failure 403 {object} apperror.AppError "Forbidden"
// @Failure 500 {object} apperror.AppError "Internal server error"
// @Router /admin/auth/register [post]
func (h *Handler) Register(c *fiber.Ctx) error {
	var req AdminRegisterDto
	if err := c.BodyParser(&req); err != nil {
		return apperror.New(fiber.StatusBadRequest, "error", "Failed to parse request body", err, nil)
	}

	if err := h.validator.Struct(&req); err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusUnprocessableEntity, "error", "Validation failed", utility.ValidationResponse(err, h.validator), nil)
		return c.Status(fiber.StatusUnprocessableEntity).JSON(rd)
	}

	claims, ok := c.Locals("adminClaims").(*middleware.AdminDataClaims)
	if !ok {
		return apperror.New(fiber.StatusUnauthorized, "error", "Unauthorized", nil, nil)
	}

	isSandbox := claims.IsSandbox

	code, err := h.svc.RegisterAdmin(req, isSandbox)
	if err != nil {
		return apperror.New(code, "error", err.Error(), err, nil)
	}

	h.logger.Info("Admin created successfully")
	rd := utility.BuildSuccessResponse(fiber.StatusCreated, "Admin created successfully", nil)
	return c.Status(code).JSON(rd)
}

// @Summary Login Admin
// @Description Login for admins
// @Tags Admin Auth
// @Accept json
// @Produce json
// @Param data body AdminLoginRequestDto true "Login request payload"
// @Success 200 {object} AdminLoginResponseDto "Login successful"
// @Failure 400 {object} apperror.AppError "Bad request, validation failed"
// @Failure 500 {object} apperror.AppError "Internal server error"
// @Router /admin/auth/login [post]
func (h *Handler) Login(c *fiber.Ctx) error {
	var req AdminLoginRequestDto
	if err := c.BodyParser(&req); err != nil {
		return apperror.New(fiber.StatusBadRequest, "error", "Failed to parse request body", err, nil)
	}

	if err := h.validator.Struct(&req); err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusUnprocessableEntity, "error", "Validation failed", utility.ValidationResponse(err, h.validator), nil)
		return c.Status(fiber.StatusUnprocessableEntity).JSON(rd)
	}

	respData, code, err := h.svc.LoginAdmin(req, req.IsSandbox)
	if err != nil {
		return apperror.New(code, "error", err.Error(), err, nil)
	}

	h.logger.Info("Admin login successful")
	rd := utility.BuildSuccessResponse(fiber.StatusOK, "Admin login successful", respData)
	return c.Status(code).JSON(rd)
}

func getPagination(c *fiber.Ctx) (int, int) {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	size, _ := strconv.Atoi(c.Query("size", "20"))
	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = 20
	}
	return page, size
}

// GetBusinesses godoc
// @Summary List Businesses
// @Description Returns a paginated list of all businesses in the system.
// @Tags Admin Queries
// @Accept json
// @Produce json
// @Param page query int false "Page number"
// @Param size query int false "Page size"
// @Param search query string false "Search query"
// @Success 200 {object} AdminBusinessListResponseDto
// @Failure 500 {object} apperror.AppError
// @Router /admin/businesses [get]
func (h *Handler) GetBusinesses(c *fiber.Ctx) error {
	rawDb, err := middleware.GetDatabase(c)
	if err != nil {
		return apperror.New(fiber.StatusInternalServerError, "error", err.Error(), err, nil)
	}
	db := dbinit.InitDB(rawDb, false)

	search := c.Query("search", "")
	page, size := getPagination(c)

	businesses, pagination, err := h.svc.GetBusinesses(db, search, page, size)
	if err != nil {
		return apperror.New(fiber.StatusInternalServerError, "error", err.Error(), err, nil)
	}

	rd := utility.BuildSuccessResponse(fiber.StatusOK, "Businesses retrieved successfully", businesses, pagination)
	return c.Status(fiber.StatusOK).JSON(rd)
}

// GetAggregators godoc
// @Summary List Aggregators
// @Description Returns a paginated list of all aggregators in the system.
// @Tags Admin Queries
// @Accept json
// @Produce json
// @Param page query int false "Page number"
// @Param size query int false "Page size"
// @Param search query string false "Search query"
// @Success 200 {object} AdminAggregatorListResponseDto
// @Failure 500 {object} apperror.AppError
// @Router /admin/aggregators [get]
func (h *Handler) GetAggregators(c *fiber.Ctx) error {
	rawDb, err := middleware.GetDatabase(c)
	if err != nil {
		return apperror.New(fiber.StatusInternalServerError, "error", err.Error(), err, nil)
	}
	db := dbinit.InitDB(rawDb, false)

	search := c.Query("search", "")
	page, size := getPagination(c)

	aggregators, pagination, err := h.svc.GetAggregators(db, search, page, size)
	if err != nil {
		return apperror.New(fiber.StatusInternalServerError, "error", err.Error(), err, nil)
	}

	rd := utility.BuildSuccessResponse(fiber.StatusOK, "Aggregators retrieved successfully", aggregators, pagination)
	return c.Status(fiber.StatusOK).JSON(rd)
}

// GetInvoicesByBusiness godoc
// @Summary List Invoices by Business
// @Description Returns a paginated list of invoices for a specific business ID.
// @Tags Admin Queries
// @Accept json
// @Produce json
// @Param id path string true "Business ID"
// @Param page query int false "Page number"
// @Param size query int false "Page size"
// @Success 200 {object} AdminInvoiceListResponseDto
// @Failure 500 {object} apperror.AppError
// @Router /admin/businesses/invoices/{id} [get]
func (h *Handler) GetInvoicesByBusiness(c *fiber.Ctx) error {
	rawDb, err := middleware.GetDatabase(c)
	if err != nil {
		return apperror.New(fiber.StatusInternalServerError, "error", err.Error(), err, nil)
	}
	db := dbinit.InitDB(rawDb, false)

	businessID := c.Params("id")
	page, size := getPagination(c)

	invoices, pagination, err := h.svc.GetInvoicesByBusiness(db, businessID, page, size)
	if err != nil {
		return apperror.New(fiber.StatusInternalServerError, "error", err.Error(), err, nil)
	}

	rd := utility.BuildSuccessResponse(fiber.StatusOK, "Invoices retrieved successfully", invoices, pagination)
	return c.Status(fiber.StatusOK).JSON(rd)
}

// GetInvoicesByAggregator godoc
// @Summary List Invoices by Aggregator
// @Description Returns a paginated list of invoices for a specific aggregator ID.
// @Tags Admin Queries
// @Accept json
// @Produce json
// @Param id path string true "Aggregator ID"
// @Param page query int false "Page number"
// @Param size query int false "Page size"
// @Success 200 {object} AdminAggregatorInvoiceListResponseDto
// @Failure 500 {object} apperror.AppError
// @Router /admin/aggregators/invoices/{id} [get]
func (h *Handler) GetInvoicesByAggregator(c *fiber.Ctx) error {
	rawDb, err := middleware.GetDatabase(c)
	if err != nil {
		return apperror.New(fiber.StatusInternalServerError, "error", err.Error(), err, nil)
	}
	db := dbinit.InitDB(rawDb, false)

	aggregatorID := c.Params("id")
	page, size := getPagination(c)

	invoices, pagination, err := h.svc.GetInvoicesByAggregator(db, aggregatorID, page, size)
	if err != nil {
		return apperror.New(fiber.StatusInternalServerError, "error", err.Error(), err, nil)
	}

	rd := utility.BuildSuccessResponse(fiber.StatusOK, "Invoices retrieved successfully", invoices, pagination)
	return c.Status(fiber.StatusOK).JSON(rd)
}

// GetTransactions godoc
// @Summary List Transactions
// @Description Returns a paginated list of all subscription transactions in the system.
// @Tags Admin Queries
// @Accept json
// @Produce json
// @Param page query int false "Page number"
// @Param size query int false "Page size"
// @Success 200 {object} AdminTransactionListResponseDto
// @Failure 500 {object} apperror.AppError
// @Router /admin/transactions [get]
func (h *Handler) GetTransactions(c *fiber.Ctx) error {
	rawDb, err := middleware.GetDatabase(c)
	if err != nil {
		return apperror.New(fiber.StatusInternalServerError, "error", err.Error(), err, nil)
	}
	db := dbinit.InitDB(rawDb, false)

	page, size := getPagination(c)

	transactions, pagination, err := h.svc.GetTransactions(db, page, size)
	if err != nil {
		return apperror.New(fiber.StatusInternalServerError, "error", err.Error(), err, nil)
	}

	rd := utility.BuildSuccessResponse(fiber.StatusOK, "Transactions retrieved successfully", transactions, pagination)
	return c.Status(fiber.StatusOK).JSON(rd)
}

// GetInvoiceStats godoc
// @Summary System Invoice Stats
// @Description Returns system-wide invoice statistics
// @Tags Admin Queries
// @Accept json
// @Produce json
// @Success 200 {object} AdminInvoiceStatsResponseDto
// @Failure 500 {object} apperror.AppError
// @Router /admin/stats/invoices [get]
func (h *Handler) GetInvoiceStats(c *fiber.Ctx) error {
	rawDb, err := middleware.GetDatabase(c)
	if err != nil {
		return apperror.New(fiber.StatusInternalServerError, "error", err.Error(), err, nil)
	}
	db := dbinit.InitDB(rawDb, false)

	stats, err := h.svc.GetInvoiceStats(db)
	if err != nil {
		return apperror.New(fiber.StatusInternalServerError, "error", err.Error(), err, nil)
	}

	rd := utility.BuildSuccessResponse(fiber.StatusOK, "Invoice stats retrieved successfully", stats)
	return c.Status(fiber.StatusOK).JSON(rd)
}

// GetBusinessStats godoc
// @Summary System Business Stats
// @Description Returns system-wide business statistics
// @Tags Admin Queries
// @Accept json
// @Produce json
// @Success 200 {object} AdminBusinessStatsResponseDto
// @Failure 500 {object} apperror.AppError
// @Router /admin/stats/businesses [get]
func (h *Handler) GetBusinessStats(c *fiber.Ctx) error {
	rawDb, err := middleware.GetDatabase(c)
	if err != nil {
		return apperror.New(fiber.StatusInternalServerError, "error", err.Error(), err, nil)
	}
	db := dbinit.InitDB(rawDb, false)

	stats, err := h.svc.GetBusinessStats(db)
	if err != nil {
		return apperror.New(fiber.StatusInternalServerError, "error", err.Error(), err, nil)
	}

	rd := utility.BuildSuccessResponse(fiber.StatusOK, "Business stats retrieved successfully", stats)
	return c.Status(fiber.StatusOK).JSON(rd)
}

// GetOverviewStats godoc
// @Summary Admin Overview Stats
// @Description Returns overview stats for invoices, companies, API calls, and registrations based on timeframe.
// @Tags Admin Queries
// @Accept json
// @Produce json
// @Param timeframe query string false "Timeframe: today, 7_days, 30_days, custom"
// @Param start_date query string false "Custom start date (YYYY-MM-DD)"
// @Param end_date query string false "Custom end date (YYYY-MM-DD)"
// @Success 200 {object} AdminOverviewStatsResponseDto
// @Failure 500 {object} apperror.AppError
// @Router /admin/stats/overview [get]
func (h *Handler) GetOverviewStats(c *fiber.Ctx) error {
	rawDb, err := middleware.GetDatabase(c)
	if err != nil {
		return apperror.New(fiber.StatusInternalServerError, "error", err.Error(), err, nil)
	}
	db := dbinit.InitDB(rawDb, false)

	timeframe := c.Query("timeframe", "30_days") // default to 30_days
	customStartDate := c.Query("start_date", "")
	customEndDate := c.Query("end_date", "")

	stats, err := h.svc.GetOverviewStats(db, timeframe, customStartDate, customEndDate)
	if err != nil {
		return apperror.New(fiber.StatusInternalServerError, "error", err.Error(), err, nil)
	}

	rd := utility.BuildSuccessResponse(fiber.StatusOK, "Overview stats retrieved successfully", stats)
	return c.Status(fiber.StatusOK).JSON(rd)
}

// CreateBusiness godoc
// @Summary Create Business
// @Description Creates a new business (admin only).
// @Tags Admin Operations
// @Accept json
// @Produce json
// @Param data body AdminCreateBusinessDto true "Business request payload"
// @Success 201 {object} utility.Response "Business created successfully"
// @Failure 400 {object} apperror.AppError
// @Failure 422 {object} apperror.AppError
// @Failure 500 {object} apperror.AppError
// @Router /admin/businesses [post]
func (h *Handler) CreateBusiness(c *fiber.Ctx) error {
	var req AdminCreateBusinessDto
	if err := c.BodyParser(&req); err != nil {
		return apperror.New(fiber.StatusBadRequest, "error", "Failed to parse request body", err, nil)
	}

	if err := h.validator.Struct(&req); err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusUnprocessableEntity, "error", "Validation failed", utility.ValidationResponse(err, h.validator), nil)
		return c.Status(fiber.StatusUnprocessableEntity).JSON(rd)
	}

	claims, ok := c.Locals("adminClaims").(*middleware.AdminDataClaims)
	if !ok {
		return apperror.New(fiber.StatusUnauthorized, "error", "Unauthorized", nil, nil)
	}

	err := h.svc.CreateBusiness(req, claims.IsSandbox)
	if err != nil {
		if strings.Contains(err.Error(), "already exists") {
			return apperror.New(fiber.StatusBadRequest, "error", err.Error(), nil, nil)
		}
		return apperror.New(fiber.StatusInternalServerError, "error", err.Error(), err, nil)
	}

	rd := utility.BuildSuccessResponse(fiber.StatusCreated, "Business created successfully", nil)
	return c.Status(fiber.StatusCreated).JSON(rd)
}

// GetBusinessDailyInvoiceStats godoc
// @Summary Business Daily Invoice Stats
// @Description Returns daily invoice stats (last 14 days) and aggregated stats for a specific business.
// @Tags Admin Queries
// @Accept json
// @Produce json
// @Param id path string true "Business ID"
// @Success 200 {object} AdminBusinessDailyStatsResponseDto
// @Failure 500 {object} apperror.AppError
// @Router /admin/businesses/stats{id} [get]
func (h *Handler) GetBusinessDailyInvoiceStats(c *fiber.Ctx) error {
	rawDb, err := middleware.GetDatabase(c)
	if err != nil {
		return apperror.New(fiber.StatusInternalServerError, "error", err.Error(), err, nil)
	}
	db := dbinit.InitDB(rawDb, false)

	businessID := c.Params("id")
	stats, err := h.svc.GetBusinessDailyInvoiceStatsDto(db, businessID)
	if err != nil {
		return apperror.New(fiber.StatusInternalServerError, "error", err.Error(), err, nil)
	}

	rd := utility.BuildSuccessResponse(fiber.StatusOK, "Business daily invoice stats retrieved successfully", stats)
	return c.Status(fiber.StatusOK).JSON(rd)
}

// UpdateBusiness godoc
// @Summary Update Business
// @Description Updates business details and status.
// @Tags Admin Operations
// @Accept json
// @Produce json
// @Param id path string true "Business ID"
// @Param data body AdminUpdateBusinessDto true "Business update payload"
// @Success 200 {object} utility.Response "Business updated successfully"
// @Failure 400 {object} apperror.AppError
// @Failure 500 {object} apperror.AppError
// @Router /admin/businesses/{id} [put]
func (h *Handler) UpdateBusiness(c *fiber.Ctx) error {
	rawDb, err := middleware.GetDatabase(c)
	if err != nil {
		return apperror.New(fiber.StatusInternalServerError, "error", err.Error(), err, nil)
	}
	db := dbinit.InitDB(rawDb, false)

	businessID := c.Params("id")
	var req AdminUpdateBusinessDto
	if err := c.BodyParser(&req); err != nil {
		return apperror.New(fiber.StatusBadRequest, "error", "Failed to parse request body", err, nil)
	}

	err = h.svc.UpdateBusiness(db, businessID, req)
	if err != nil {
		return apperror.New(fiber.StatusInternalServerError, "error", err.Error(), err, nil)
	}

	rd := utility.BuildSuccessResponse(fiber.StatusOK, "Business updated successfully", nil)
	return c.Status(fiber.StatusOK).JSON(rd)
}

// GetBusinessAggregatorInfo godoc
// @Summary Business Aggregator Info
// @Description Returns current aggregator ID and relationship history for a business.
// @Tags Admin Queries
// @Accept json
// @Produce json
// @Param id path string true "Business ID"
// @Success 200 {object} AdminBusinessAggregatorInfoResponse
// @Failure 500 {object} apperror.AppError
// @Router /admin/businesses/aggregator/{id} [get]
func (h *Handler) GetBusinessAggregatorInfo(c *fiber.Ctx) error {
	rawDb, err := middleware.GetDatabase(c)
	if err != nil {
		return apperror.New(fiber.StatusInternalServerError, "error", err.Error(), err, nil)
	}
	db := dbinit.InitDB(rawDb, false)

	businessID := c.Params("id")
	info, err := h.svc.GetBusinessAggregatorInfo(db, businessID)
	if err != nil {
		return apperror.New(fiber.StatusInternalServerError, "error", err.Error(), err, nil)
	}

	rd := utility.BuildSuccessResponse(fiber.StatusOK, "Business aggregator info retrieved successfully", info)
	return c.Status(fiber.StatusOK).JSON(rd)
}

// CreateAggregator godoc
// @Summary Create Aggregator
// @Description Creates a new aggregator (admin only).
// @Tags Admin Operations
// @Accept json
// @Produce json
// @Param data body AdminCreateAggregatorDto true "Aggregator request payload"
// @Success 201 {object} utility.Response "Aggregator created successfully"
// @Failure 400 {object} apperror.AppError
// @Failure 422 {object} apperror.AppError
// @Failure 500 {object} apperror.AppError
// @Router /admin/aggregators [post]
func (h *Handler) CreateAggregator(c *fiber.Ctx) error {
	var req AdminCreateAggregatorDto
	if err := c.BodyParser(&req); err != nil {
		return apperror.New(fiber.StatusBadRequest, "error", "Failed to parse request body", err, nil)
	}

	if err := h.validator.Struct(&req); err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusUnprocessableEntity, "error", "Validation failed", utility.ValidationResponse(err, h.validator), nil)
		return c.Status(fiber.StatusUnprocessableEntity).JSON(rd)
	}

	claims, ok := c.Locals("adminClaims").(*middleware.AdminDataClaims)
	if !ok {
		return apperror.New(fiber.StatusUnauthorized, "error", "Unauthorized", nil, nil)
	}

	err := h.svc.CreateAggregator(req, claims.IsSandbox)
	if err != nil {
		if strings.Contains(err.Error(), "already exists") {
			return apperror.New(fiber.StatusBadRequest, "error", err.Error(), nil, nil)
		}
		return apperror.New(fiber.StatusInternalServerError, "error", err.Error(), err, nil)
	}

	rd := utility.BuildSuccessResponse(fiber.StatusCreated, "Aggregator created successfully", nil)
	return c.Status(fiber.StatusCreated).JSON(rd)
}

// UpdateAggregator godoc
// @Summary Update Aggregator
// @Description Updates aggregator details and status.
// @Tags Admin Operations
// @Accept json
// @Produce json
// @Param id path string true "Aggregator ID"
// @Param data body AdminUpdateBusinessDto true "Aggregator update payload"
// @Success 200 {object} utility.Response "Aggregator updated successfully"
// @Failure 400 {object} apperror.AppError
// @Failure 500 {object} apperror.AppError
// @Router /admin/aggregators/{id} [put]
func (h *Handler) UpdateAggregator(c *fiber.Ctx) error {
	rawDb, err := middleware.GetDatabase(c)
	if err != nil {
		return apperror.New(fiber.StatusInternalServerError, "error", err.Error(), err, nil)
	}
	db := dbinit.InitDB(rawDb, false)

	aggregatorID := c.Params("id")
	var req AdminUpdateBusinessDto
	if err := c.BodyParser(&req); err != nil {
		return apperror.New(fiber.StatusBadRequest, "error", "Failed to parse request body", err, nil)
	}

	err = h.svc.UpdateAggregator(db, aggregatorID, req)
	if err != nil {
		return apperror.New(fiber.StatusInternalServerError, "error", err.Error(), err, nil)
	}

	rd := utility.BuildSuccessResponse(fiber.StatusOK, "Aggregator updated successfully", nil)
	return c.Status(fiber.StatusOK).JSON(rd)
}

// GetAggregatorInfo godoc
// @Summary Aggregator Info
// @Description Returns stats and associated companies for an aggregator.
// @Tags Admin Queries
// @Accept json
// @Produce json
// @Param id path string true "Aggregator ID"
// @Success 200 {object} AdminAggregatorInfoResponse
// @Failure 500 {object} apperror.AppError
// @Router /admin/aggregators/{id} [get]
func (h *Handler) GetAggregatorInfo(c *fiber.Ctx) error {
	rawDb, err := middleware.GetDatabase(c)
	if err != nil {
		return apperror.New(fiber.StatusInternalServerError, "error", err.Error(), err, nil)
	}
	db := dbinit.InitDB(rawDb, false)

	aggregatorID := c.Params("id")
	info, err := h.svc.GetAggregatorInfo(db, aggregatorID)
	if err != nil {
		return apperror.New(fiber.StatusInternalServerError, "error", err.Error(), err, nil)
	}

	rd := utility.BuildSuccessResponse(fiber.StatusOK, "Aggregator info retrieved successfully", info)
	return c.Status(fiber.StatusOK).JSON(rd)
}

// GetAggregatorInvitations godoc
// @Summary Aggregator Invitations
// @Description Returns pending invitations sent by an aggregator.
// @Tags Admin Queries
// @Accept json
// @Produce json
// @Param id path string true "Aggregator ID"
// @Success 200 {object} AdminAggregatorInvitationsResponse
// @Failure 500 {object} apperror.AppError
// @Router /admin/aggregators/invitations/{id} [get]
func (h *Handler) GetAggregatorInvitations(c *fiber.Ctx) error {
	rawDb, err := middleware.GetDatabase(c)
	if err != nil {
		return apperror.New(fiber.StatusInternalServerError, "error", err.Error(), err, nil)
	}
	db := dbinit.InitDB(rawDb, false)

	aggregatorID := c.Params("id")
	invitations, err := h.svc.GetAggregatorInvitations(db, aggregatorID)
	if err != nil {
		return apperror.New(fiber.StatusInternalServerError, "error", err.Error(), err, nil)
	}

	rd := utility.BuildSuccessResponse(fiber.StatusOK, "Aggregator invitations retrieved successfully", invitations)
	return c.Status(fiber.StatusOK).JSON(rd)
}
