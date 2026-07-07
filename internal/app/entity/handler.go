package entity

import (
	"einvoice-access-point/internal/apperror"
	"einvoice-access-point/internal/data/database"
	"einvoice-access-point/internal/data/entities"
	"einvoice-access-point/internal/middleware"
	"einvoice-access-point/internal/pkg/firs_models"
	"einvoice-access-point/internal/utility"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
)

type Handler struct {
	svc       *Service
	Db        *database.Database
	TestDb    *database.Database
	Validator *validator.Validate
	Logger    *utility.Logger
}

func NewHandler(validator *validator.Validate, logger *utility.Logger, db, testDb *database.Database) *Handler {
	svc := NewService()
	return &Handler{
		svc:       svc,
		Validator: validator,
		Logger:    logger,
		Db:        db,
		TestDb:    testDb,
	}
}

// @Summary      Get Entities
// @Description  Retrieve a paginated list of entities
// @Tags         Entity
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        query  query     entities.PaginationQuery  false  "Pagination and sorting"
// @Success      200 {object} entities.Response "Entities retrieved successfully"
// @Failure      400 {object} entities.Response "Bad request"
// @Failure      401 {object} entities.Response "Unauthorized"
// @Failure      500 {object} entities.Response "Internal server error"
// @Router       /entity [get]
func (h *Handler) GetEntities(c *fiber.Ctx) error {
	userDetails, err := middleware.GetUserDetails(c)
	var query entities.PaginationQuery
	if err := c.QueryParser(&query); err != nil {
		return apperror.New(fiber.StatusBadRequest, "error", "Invalid query parameters", err, nil)
	}

	queries := h.svc.FetchQueryItems(query)

	respData, errDetails, err := h.svc.GetEntities(queries, userDetails.IsSandbox)
	if err != nil {
		return apperror.New(fiber.StatusBadRequest, "error", err.Error(), errDetails, nil)
	}

	h.Logger.Info("Entities gotten successfully")
	rd := utility.BuildSuccessResponse(fiber.StatusOK, "Entities gotten successfully", respData)
	return c.Status(fiber.StatusOK).JSON(rd)
}

// @Summary      Get Entity by ID
// @Description  Retrieve details of a specific entity using its ID
// @Tags         Entity
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        entity_id   path      string  true  "Entity ID"
// @Success      200 {object} entities.Response "Entity retrieved successfully"
// @Failure      400 {object} entities.Response "Bad request"
// @Failure      401 {object} entities.Response "Unauthorized"
// @Failure      404 {object} entities.Response "Entity not found"
// @Failure      500 {object} entities.Response "Internal server error"
// @Router       /entity/{entity_id} [get]
func (h *Handler) GetEntity(c *fiber.Ctx) error {
	userDetails, err := middleware.GetUserDetails(c)
	entityId := c.Params("entity_id")
	if entityId == "" {
		return apperror.New(fiber.StatusBadRequest, "error", "entity id is required", nil, nil)
	}

	respData, errDetails, err := h.svc.GetEntity(entityId, userDetails.IsSandbox)
	if err != nil {
		return apperror.New(fiber.StatusBadRequest, "error", err.Error(), errDetails, nil)
	}

	h.Logger.Info("successfully")
	rd := utility.BuildSuccessResponse(fiber.StatusOK, "successfully", respData)
	return c.Status(fiber.StatusOK).JSON(rd)
}

// @Summary      Verify TIN
// @Description  Verify a taxpayer identification number (TIN)
// @Tags         Entity
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        data  body      firs_models.VerifyTinData  true  "TIN verification request"
// @Success      200 {object} entities.Response "TIN verified successfully"
// @Failure      400 {object} entities.Response "Bad request"
// @Failure      401 {object} entities.Response "Unauthorized"
// @Failure      422 {object} entities.Response "Validation failed"
// @Failure      500 {object} entities.Response "Internal server error"
// @Router       /entity/verify-tin [post]
func (h *Handler) VerifyTin(c *fiber.Ctx) error {
	var req firs_models.VerifyTinData

	err := c.BodyParser(&req)
	if err != nil {
		return apperror.New(fiber.StatusBadRequest, "error", "Failed to parse request body", err, nil)
	}

	err = h.Validator.Struct(&req)
	if err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusUnprocessableEntity, "error", "Validation failed", utility.ValidationResponse(err, h.Validator), nil)
		return c.Status(fiber.StatusUnprocessableEntity).JSON(rd)
	}

	respData, errDetails, err := h.svc.VerifyTin(req.TIN)
	if err != nil {
		return apperror.New(fiber.StatusBadRequest, "error", err.Error(), errDetails, nil)
	}

	h.Logger.Info("successfully")
	rd := utility.BuildSuccessResponse(fiber.StatusOK, "successfully", respData)
	return c.Status(fiber.StatusOK).JSON(rd)
}

// @Summary      Post VAT Payment
// @Description  Submit VAT payment details to FIRS
// @Tags         Entity
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        data  body      firs_models.FirsTransactionVatPayload  true  "VAT payment request payload"
// @Success      200 {object} entities.Response "VAT payment processed successfully"
// @Failure      400 {object} entities.Response "Bad request"
// @Failure      401 {object} entities.Response "Unauthorized"
// @Failure      422 {object} entities.Response "Validation failed"
// @Failure      500 {object} entities.Response "Internal server error"
// @Router       /entity/vat-payment [post]
func (h *Handler) PostVatPayment(c *fiber.Ctx) error {
	var req firs_models.FirsTransactionVatPayload

	err := c.BodyParser(&req)
	if err != nil {
		return apperror.New(fiber.StatusBadRequest, "error", "Failed to parse request body", err, nil)
	}

	err = h.Validator.Struct(&req)
	if err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusUnprocessableEntity, "error", "Validation failed", utility.ValidationResponse(err, h.Validator), nil)
		return c.Status(fiber.StatusUnprocessableEntity).JSON(rd)
	}

	respData, errDetails, err := h.svc.PostVatPayment(req)
	if err != nil {
		return apperror.New(fiber.StatusBadRequest, "error", err.Error(), errDetails, nil)
	}

	h.Logger.Info("successfully")
	rd := utility.BuildSuccessResponse(fiber.StatusOK, "successfully", respData)
	return c.Status(fiber.StatusOK).JSON(rd)
}
