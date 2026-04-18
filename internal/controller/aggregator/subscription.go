package aggregator

import (
	"encoding/json"
	"errors"
	"net/http"

	"einvoice-access-point/external/paystack"
	"einvoice-access-point/internal/dtos"
	"einvoice-access-point/internal/services/subscription"
	"einvoice-access-point/pkg/database"
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

// PaystackWebhook godoc
// @Summary Handle Aggregator Subscription Paystack Webhook
// @Description Verifies Paystack signature, acknowledges immediately, then processes transaction and subscription updates asynchronously
// @Tags Aggregator Subscription
// @Accept json
// @Produce json
// @Param x-paystack-signature header string true "Paystack signature"
// @Param payload body object true "Webhook payload"
// @Success 200 {object} models.Response "Webhook accepted for processing"
// @Failure 400 {object} models.Response "Bad request"
// @Failure 401 {object} models.Response "Unauthorized"
// @Failure 500 {object} models.Response "Internal server error"
// @Router /aggregator/subscription/paystack/webhook [post]
func (base *Controller) PaystackWebhook(c *fiber.Ctx) error {
	signature := c.Get("x-paystack-signature")
	if signature == "" {
		rd := utility.BuildErrorResponse(fiber.StatusUnauthorized, "error", "missing paystack signature", nil, nil)
		return c.Status(fiber.StatusUnauthorized).JSON(rd)
	}
	rawBody := append([]byte(nil), c.Body()...)

	if err := subscription.ValidatePaystackSignature(rawBody, signature); err != nil {
		statusCode := fiber.StatusInternalServerError
		if errors.Is(err, subscription.ErrInvalidPaystackSignature) {
			statusCode = fiber.StatusUnauthorized
		}

		rd := utility.BuildErrorResponse(statusCode, "error", err.Error(), nil, nil)
		return c.Status(statusCode).JSON(rd)
	}

	var payload paystack.WebhookPayload
	if err := json.Unmarshal(rawBody, &payload); err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusBadRequest, "error", "invalid webhook payload", err.Error(), nil)
		return c.Status(fiber.StatusBadRequest).JSON(rd)
	}

	metadataSandbox, hasMetadataSandbox := payload.MetadataIsSandbox()
	if !hasMetadataSandbox {
		rd := utility.BuildErrorResponse(fiber.StatusBadRequest, "error", "metadata.is_sandbox is required in webhook payload", nil, nil)
		return c.Status(fiber.StatusBadRequest).JSON(rd)
	}

	targetDb := base.Db
	databaseName := "production"
	if metadataSandbox {
		targetDb = base.TestDB
		databaseName = "sandbox"
	}

	go func(body []byte, receivedSignature string, selectedDB *database.Database, environment string, reference string) {
		defer func() {
			if recovered := recover(); recovered != nil {
				base.Logger.Error("paystack webhook async panic (env=%s, ref=%s): %v", environment, reference, recovered)
			}
		}()

		_, code, err := subscription.HandlePaystackWebhook(body, receivedSignature, selectedDB.Postgresql.DB())
		if err != nil {
			base.Logger.Error("paystack webhook async processing failed (env=%s, ref=%s, code=%d): %v", environment, reference, code, err)
			return
		}

		base.Logger.Info("paystack webhook async processing completed (env=%s, ref=%s)", environment, reference)
	}(rawBody, signature, targetDb, databaseName, payload.Data.Reference)

	rd := utility.BuildSuccessResponse(http.StatusOK, "webhook accepted for processing", fiber.Map{
		"event":              payload.Event,
		"reference":          payload.Data.Reference,
		"transaction_status": "queued",
		"database":           databaseName,
	})
	return c.Status(http.StatusOK).JSON(rd)
}
