package admin

import (
	"einvoice-access-point/internal/apperror"
	"einvoice-access-point/internal/config"
	"einvoice-access-point/internal/data/database"
	"einvoice-access-point/internal/data/dbinit"
	"einvoice-access-point/internal/middleware"
	"einvoice-access-point/internal/utility"
	"strconv"

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
