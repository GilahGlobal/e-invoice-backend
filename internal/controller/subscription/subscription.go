package subscription

import (
	"einvoice-access-point/internal/dtos"
	subscriptionService "einvoice-access-point/internal/services/subscription"
	"einvoice-access-point/pkg/database"
	"einvoice-access-point/pkg/middleware"
	"einvoice-access-point/pkg/utility"
	"net/http"
	"strconv"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
)

type Controller struct {
	Db        *database.Database
	TestDb    *database.Database
	Validator *validator.Validate
	Logger    *utility.Logger
}

// GetPlans godoc
// @Summary List Subscription Plans
// @Description Retrieves all available subscription plans
// @Tags Subscription
// @Accept json
// @Produce json
// @Param is_sandbox query string true "Use sandbox database (true/false)"
// @Success 200 {object} dtos.SubscriptionPlansResponseDto "Plans fetched successfully"
// @Failure 400 {object} models.Response "Bad request"
// @Failure 422 {object} models.Response "Unprocessable entity"
// @Failure 500 {object} models.Response "Internal server error"
// @Router /subscription/plans [get]
func (base *Controller) GetPlans(c *fiber.Ctx) error {
	var query dtos.SubscriptionPlanQueryDto
	if err := c.QueryParser(&query); err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusBadRequest, "error", "Failed to parse query params", err.Error(), nil)
		return c.Status(fiber.StatusBadRequest).JSON(rd)
	}

	if err := base.Validator.Struct(&query); err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusUnprocessableEntity, "error", "Validation failed", utility.ValidationResponse(err, base.Validator), nil)
		return c.Status(fiber.StatusUnprocessableEntity).JSON(rd)
	}

	isSandbox, err := strconv.ParseBool(query.IsSandbox)
	if err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusBadRequest, "error", "is_sandbox must be true or false", err.Error(), nil)
		return c.Status(fiber.StatusBadRequest).JSON(rd)
	}
	db := middleware.GetDatabaseInstance(isSandbox, base.Db, base.TestDb)

	plans, err := subscriptionService.ListPlans(db)
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusInternalServerError, "error", "failed to fetch plans", err.Error(), nil)
		return c.Status(http.StatusInternalServerError).JSON(rd)
	}

	rd := utility.BuildSuccessResponse(fiber.StatusOK, "plans fetched successfully", plans)
	return c.Status(fiber.StatusOK).JSON(rd)
}

// CreatePlan godoc
// @Summary Create Subscription Plan
// @Description Creates a subscription plan in the specified environment database
// @Tags Subscription
// @Accept json
// @Produce json
// @Param data body dtos.CreateSubscriptionPlanDto true "Create plan request payload"
// @Success 201 {object} dtos.CreateSubscriptionPlanResponseDto "Plan created successfully"
// @Failure 400 {object} models.Response "Bad request"
// @Failure 422 {object} models.Response "Unprocessable entity"
// @Failure 500 {object} models.Response "Internal server error"
// @Router /subscription/plans [post]
func (base *Controller) CreatePlan(c *fiber.Ctx) error {
	var req dtos.CreateSubscriptionPlanDto
	if err := c.BodyParser(&req); err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusBadRequest, "error", "Failed to parse request body", err.Error(), nil)
		return c.Status(fiber.StatusBadRequest).JSON(rd)
	}

	if err := base.Validator.Struct(&req); err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusUnprocessableEntity, "error", "Validation failed", utility.ValidationResponse(err, base.Validator), nil)
		return c.Status(fiber.StatusUnprocessableEntity).JSON(rd)
	}

	db := middleware.GetDatabaseInstance(*req.IsSandbox, base.Db, base.TestDb)

	createdPlan, err := subscriptionService.CreatePlan(req, db)
	if err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusBadRequest, "error", err.Error(), err, nil)
		return c.Status(fiber.StatusBadRequest).JSON(rd)
	}

	rd := utility.BuildSuccessResponse(fiber.StatusCreated, "plan created successfully", fiber.Map{
		"is_sandbox": *req.IsSandbox,
		"plan":       createdPlan,
	})
	return c.Status(fiber.StatusCreated).JSON(rd)
}
