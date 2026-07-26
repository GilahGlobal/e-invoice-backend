package admin

import (
	"strconv"

	"einvoice-access-point/internal/apperror"
	"einvoice-access-point/internal/data/dbinit"
	"einvoice-access-point/internal/middleware"
	"einvoice-access-point/internal/utility"

	"github.com/gofiber/fiber/v2"
)

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
