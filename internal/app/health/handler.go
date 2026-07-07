package health

import (
	"einvoice-access-point/internal/apperror"
	"einvoice-access-point/internal/data/database"
	"einvoice-access-point/internal/data/entities"
	"einvoice-access-point/internal/utility"
	"fmt"

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

func NewHandler(svc *Service, validator *validator.Validate, logger *utility.Logger, db, testDb *database.Database) *Handler {
	return &Handler{
		svc:       svc,
		Validator: validator,
		Logger:    logger,
		Db:        db,
		TestDb:    testDb,
	}
}

// Post godoc
// @Summary      Health check (POST)
// @Description  Accepts a ping message and validates connectivity.
// @Tags         Health
// @Accept       json
// @Produce      json
// @Param        request  body      entities.Ping  true  "Ping request payload"
// @Success      200      {object}  entities.Response "ping successful"
// @Failure      400      {object}  entities.Response "invalid request or validation failed"
// @Failure      500      {object}  entities.Response "ping failed"
// @Router       /health [post]
func (h *Handler) Post(c *fiber.Ctx) error {
	var req entities.Ping

	if err := c.BodyParser(&req); err != nil {
		return apperror.New(fiber.StatusBadRequest, "error", "Failed to parse request body", err, nil)
	}

	if err := h.Validator.Struct(&req); err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusBadRequest, "error", "Validation failed", utility.ValidationResponse(err, h.Validator), nil)
		return c.Status(fiber.StatusBadRequest).JSON(rd)
	}

	if !h.svc.ReturnTrue() {
		return apperror.New(fiber.StatusInternalServerError, "error", "ping failed", fmt.Errorf("ping failed"), nil)
	}

	h.Logger.Info("ping successful")
	rd := utility.BuildSuccessResponse(fiber.StatusOK, "ping successful", req.Message)
	return c.Status(fiber.StatusOK).JSON(rd)
}

// Get godoc
// @Summary      Health check (GET)
// @Description  Performs a basic service availability check.
// @Tags         Health
// @Produce      json
// @Success      200  {object}  entities.Response "ping successful"
// @Failure      400  {object}  entities.Response "ping failed"
// @Router       /health [get]
func (h *Handler) Get(c *fiber.Ctx) error {
	if !h.svc.ReturnTrue() {
		return apperror.New(fiber.StatusInternalServerError, "error", "ping failed", fmt.Errorf("ping failed"), nil)
	}

	h.Logger.Info("ping successful")
	rd := utility.BuildSuccessResponse(fiber.StatusOK, "ping successful", nil)
	return c.Status(fiber.StatusOK).JSON(rd)
}

// FirsHealthCheck godoc
// @Summary      FIRS API Health Check
// @Description  Calls external FIRS API to confirm connectivity.
// @Tags         Health
// @Produce      json
// @Success      200  {object}  entities.Response "ping successful"
// @Failure      400  {object}  entities.Response "invalid response from FIRS API"
// @Router       /health/firs [get]
func (h *Handler) FirsHealthCheck(c *fiber.Ctx) error {
	respData, errDetails, err := h.svc.FirsApiHealthCheck()
	if err != nil {
		return apperror.New(fiber.StatusBadRequest, "error", err.Error(), errDetails, nil)
	}

	h.Logger.Info("ping successfully")
	rd := utility.BuildSuccessResponse(fiber.StatusOK, "ping successfully", respData)
	return c.Status(fiber.StatusOK).JSON(rd)
}
