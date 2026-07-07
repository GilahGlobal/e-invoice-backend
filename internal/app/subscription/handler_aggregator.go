package subscription

import (
	"einvoice-access-point/internal/apperror"
	"einvoice-access-point/internal/data/database"
	"einvoice-access-point/internal/data/dbinit"
	"einvoice-access-point/internal/middleware"
	"einvoice-access-point/internal/utility"
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
)

type AggregatorHandler struct {
	svc       *Service
	Validator *validator.Validate
	Logger    *utility.Logger
	Db        *database.Database
	TestDB    *database.Database
}

func NewAggregatorHandler(validator *validator.Validate, logger *utility.Logger, db, testDb *database.Database) *AggregatorHandler {
	prodDB := dbinit.InitDB(db.Postgresql.DB(), false)
	testDBConn := dbinit.InitDB(testDb.Postgresql.DB(), false)
	svc := NewServiceWithDB(prodDB, testDBConn)
	return &AggregatorHandler{
		svc:       svc,
		Validator: validator,
		Logger:    logger,
		Db:        db,
		TestDB:    testDb,
	}
}

// GetPlans godoc
// @Summary List Aggregator Subscription Plans
// @Description Retrieves active plans available for aggregator subscriptions
// @Tags Aggregator Subscription
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} SubscriptionPlansResponseDto "Plans fetched successfully"
// @Failure 401 {object} entities.Response "Unauthorized"
// @Failure 500 {object} entities.Response "Internal server error"
// @Router /aggregator/subscription/plans [get]
func (h *AggregatorHandler) GetPlans(c *fiber.Ctx) error {
	if _, err := middleware.GetUserDetails(c); err != nil {
		return apperror.New(fiber.StatusUnauthorized, "error", "Unauthorized", err, nil)
	}

	db, err := middleware.GetDatabase(c)
	if err != nil {
		return apperror.New(fiber.StatusInternalServerError, "error", err.Error(), err, nil)
	}

	plans, err := h.svc.ListActivePlans(db)
	if err != nil {
		return apperror.New(http.StatusInternalServerError, "error", "failed to fetch plans", err, nil)
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
// @Param data body AggregatorSubscribeRequestDto true "Subscribe request payload"
// @Success 200 {object} AggregatorSubscribeResponseDto "Subscription initialized successfully"
// @Failure 400 {object} entities.Response "Bad request"
// @Failure 401 {object} entities.Response "Unauthorized"
// @Failure 422 {object} entities.Response "Unprocessable entity"
// @Failure 502 {object} entities.Response "Bad gateway"
// @Router /aggregator/subscription/subscribe [post]
func (h *AggregatorHandler) Subscribe(c *fiber.Ctx) error {
	userDetails, err := middleware.GetUserDetails(c)
	if err != nil {
		return apperror.New(fiber.StatusUnauthorized, "error", "Unauthorized", err, nil)
	}

	var req AggregatorSubscribeRequestDto
	if err := c.BodyParser(&req); err != nil {
		return apperror.New(fiber.StatusBadRequest, "error", "Failed to parse request body", err, nil)
	}

	if err := h.Validator.Struct(&req); err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusUnprocessableEntity, "error", "Validation failed", utility.ValidationResponse(err, h.Validator), nil)
		return c.Status(fiber.StatusUnprocessableEntity).JSON(rd)
	}

	db, err := middleware.GetDatabase(c)
	if err != nil {
		return apperror.New(fiber.StatusInternalServerError, "error", err.Error(), err, nil)
	}

	respData, code, err := h.svc.SubscribeBusinessToPlan(req, userDetails.ID, userDetails.IsSandbox, db)
	if err != nil {
		return apperror.New(code, "error", err.Error(), err, nil)
	}

	rd := utility.BuildSuccessResponse(http.StatusOK, "subscription initialized successfully", respData)
	return c.Status(code).JSON(rd)
}
