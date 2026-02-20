package health

import (
	"einvoice-access-point/internal/services/ping"
	"einvoice-access-point/pkg/database"
	"einvoice-access-point/pkg/models"
	"einvoice-access-point/pkg/utility"
	"fmt"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
)

type Controller struct {
	Db        *database.Database
	TestDb    *database.Database
	Validator *validator.Validate
	Logger    *utility.Logger
}

// Post godoc
// @Summary      Health check (POST)
// @Description  Accepts a ping message and validates connectivity.
// @Tags         Health
// @Accept       json
// @Produce      json
// @Param        request  body      models.Ping  true  "Ping request payload"
// @Success      200      {object}  models.Response "ping successful"
// @Failure      400      {object}  models.Response "invalid request or validation failed"
// @Failure      500      {object}  models.Response "ping failed"
// @Router       /health [post]
func (base *Controller) Post(c *fiber.Ctx) error {
	var req models.Ping

	if err := c.BodyParser(&req); err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusBadRequest, "error", "Failed to parse request body", err, nil)
		return c.Status(fiber.StatusBadRequest).JSON(rd)
	}

	if err := base.Validator.Struct(&req); err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusBadRequest, "error", "Validation failed", utility.ValidationResponse(err, base.Validator), nil)
		return c.Status(fiber.StatusBadRequest).JSON(rd)
	}

	if !ping.ReturnTrue() {
		rd := utility.BuildErrorResponse(fiber.StatusInternalServerError, "error", "ping failed", fmt.Errorf("ping failed"), nil)
		return c.Status(fiber.StatusInternalServerError).JSON(rd)
	}

	base.Logger.Info("ping successful")
	rd := utility.BuildSuccessResponse(fiber.StatusOK, "ping successful", req.Message)
	return c.Status(fiber.StatusOK).JSON(rd)
}

// Get godoc
// @Summary      Health check (GET)
// @Description  Performs a basic service availability check.
// @Tags         Health
// @Produce      json
// @Success      200  {object}  models.Response "ping successful"
// @Failure      400  {object}  models.Response "ping failed"
// @Router       /health [get]
func (base *Controller) Get(c *fiber.Ctx) error {
	if !ping.ReturnTrue() {
		rd := utility.BuildErrorResponse(fiber.StatusInternalServerError, "error", "ping failed", fmt.Errorf("ping failed"), nil)
		return c.Status(fiber.StatusInternalServerError).JSON(rd)
	}

	base.Logger.Info("ping successful")
	rd := utility.BuildSuccessResponse(fiber.StatusOK, "ping successful", nil)
	return c.Status(fiber.StatusOK).JSON(rd)
}

// FirsHealthCheck godoc
// @Summary      FIRS API Health Check
// @Description  Calls external FIRS API to confirm connectivity.
// @Tags         Health
// @Produce      json
// @Success      200  {object}  models.Response "ping successful"
// @Failure      400  {object}  models.Response "invalid response from FIRS API"
// @Router       /health/firs [get]
func (base *Controller) FirsHealthCheck(c *fiber.Ctx) error {
	respData, errDetails, err := ping.FirsApiHealthCheck()
	if err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusBadRequest, "error", err.Error(), errDetails, nil)
		return c.Status(fiber.StatusBadRequest).JSON(rd)
	}

	base.Logger.Info("ping successfully")
	rd := utility.BuildSuccessResponse(fiber.StatusOK, "ping successfully", respData)
	return c.Status(fiber.StatusOK).JSON(rd)
}
