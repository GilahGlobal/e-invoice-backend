package aggregator

import (
	"net/http"

	"einvoice-access-point/internal/dtos"
	"einvoice-access-point/internal/services/subscription"
	"einvoice-access-point/pkg/middleware"
	"einvoice-access-point/pkg/utility"

	"github.com/gofiber/fiber/v2"
)

// GetPlans godoc
// @Summary List Aggregator Subscription Plans
// @Description Retrieves active plans available for aggregator subscriptions
// @Tags Aggregator Subscription
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} dtos.SubscriptionPlansResponseDto "Plans fetched successfully"
// @Failure 401 {object} models.Response "Unauthorized"
// @Failure 500 {object} models.Response "Internal server error"
// @Router /aggregator/subscription/plans [get]
func (base *Controller) GetPlans(c *fiber.Ctx) error {
	userDetails, err := middleware.GetUserDetails(c)
	if err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusUnauthorized, "error", "Unauthorized", err, nil)
		return c.Status(fiber.StatusUnauthorized).JSON(rd)
	}

	db := middleware.GetDatabaseInstance(userDetails.IsSandbox, base.Db, base.TestDB)

	plans, err := subscription.ListActivePlans(db)
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusInternalServerError, "error", "failed to fetch plans", err.Error(), nil)
		return c.Status(http.StatusInternalServerError).JSON(rd)
	}

	rd := utility.BuildSuccessResponse(http.StatusOK, "plans fetched successfully", plans)
	return c.Status(http.StatusOK).JSON(rd)
}

// Subscribe godoc
// @Summary Subscribe A Managed Business To A Plan
// @Description Initializes a Paystack transaction for an aggregator-managed business subscription plan
// @Tags Aggregator Subscription
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param data body dtos.AggregatorSubscribeRequestDto true "Subscribe request payload"
// @Success 200 {object} dtos.AggregatorSubscribeResponseDto "Subscription initialized successfully"
// @Failure 400 {object} models.Response "Bad request"
// @Failure 401 {object} models.Response "Unauthorized"
// @Failure 422 {object} models.Response "Unprocessable entity"
// @Failure 502 {object} models.Response "Bad gateway"
// @Router /aggregator/subscription/subscribe [post]
func (base *Controller) Subscribe(c *fiber.Ctx) error {
	userDetails, err := middleware.GetUserDetails(c)
	if err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusUnauthorized, "error", "Unauthorized", err, nil)
		return c.Status(fiber.StatusUnauthorized).JSON(rd)
	}

	var req dtos.AggregatorSubscribeRequestDto
	if err := c.BodyParser(&req); err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusBadRequest, "error", "Failed to parse request body", err.Error(), nil)
		return c.Status(fiber.StatusBadRequest).JSON(rd)
	}

	if err := base.Validator.Struct(&req); err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusUnprocessableEntity, "error", "Validation failed", utility.ValidationResponse(err, base.Validator), nil)
		return c.Status(fiber.StatusUnprocessableEntity).JSON(rd)
	}

	db := middleware.GetDatabaseInstance(userDetails.IsSandbox, base.Db, base.TestDB)

	respData, code, err := subscription.SubscribeBusinessToPlan(req, userDetails.ID, userDetails.IsSandbox, db)
	if err != nil {
		rd := utility.BuildErrorResponse(code, "error", err.Error(), err, nil)
		return c.Status(code).JSON(rd)
	}

	rd := utility.BuildSuccessResponse(http.StatusOK, "subscription initialized successfully", respData)
	return c.Status(code).JSON(rd)
}
