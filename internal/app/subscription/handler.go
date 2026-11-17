package subscription

import (
	"einvoice-access-point/internal/apperror"
	"einvoice-access-point/internal/data/database"
	"einvoice-access-point/internal/data/dbinit"
	"einvoice-access-point/internal/middleware"
	"einvoice-access-point/internal/pkg/paystack"
	"einvoice-access-point/internal/utility"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type Handler struct {
	svc       *Service
	Db        *database.Database
	TestDb    *database.Database
	Validator *validator.Validate
	Logger    *utility.Logger
}

func NewHandler(validator *validator.Validate, logger *utility.Logger, db, testDb *database.Database) *Handler {
	prodDB := dbinit.InitDB(db.Postgresql.DB(), false)
	testDBConn := dbinit.InitDB(testDb.Postgresql.DB(), false)
	svc := NewServiceWithDB(prodDB, testDBConn)
	return &Handler{
		svc:       svc,
		Validator: validator,
		Logger:    logger,
		Db:        db,
		TestDb:    testDb,
	}
}

// GetPlans godoc
// @Summary List Subscription Plans
// @Description Retrieves all available subscription plans
// @Tags Subscription
// @Accept json
// @Produce json
// @Param is_sandbox query string false "Use sandbox database (true/false) when unauthenticated"
// @Success 200 {object} SubscriptionPlansResponseDto "Plans fetched successfully"
// @Failure 400 {object} entities.Response "Bad request"
// @Failure 422 {object} entities.Response "Unprocessable entity"
// @Failure 500 {object} entities.Response "Internal server error"
// @Router /subscription/plans [get]
func (h *Handler) GetPlans(c *fiber.Ctx) error {
	db, err := middleware.GetDatabase(c)
	if err != nil {
		isSandboxQuery := c.Query("is_sandbox")
		if isSandboxQuery != "" {
			isSandbox, parseErr := strconv.ParseBool(isSandboxQuery)
			if parseErr != nil {
				return apperror.New(fiber.StatusBadRequest, "error", "is_sandbox must be true or false", parseErr, nil)
			}
			if isSandbox {
				db = h.TestDb.Postgresql.DB()
			} else {
				db = h.Db.Postgresql.DB()
			}
			err = nil
		}
	}
	if err != nil || db == nil {
		return apperror.New(fiber.StatusInternalServerError, "error", "failed to determine database connection", err, nil)
	}

	plans, err := h.svc.ListActivePlans(db)
	if err != nil {
		return apperror.New(http.StatusInternalServerError, "error", "failed to fetch plans", err, nil)
	}

	rd := utility.BuildSuccessResponse(fiber.StatusOK, "plans fetched successfully", plans)
	return c.Status(fiber.StatusOK).JSON(rd)
}

// Subscribe godoc
// @Summary Subscribe To A Plan
// @Description Initializes a Paystack transaction for a business or aggregator subscription plan
// @Tags Subscription
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param data body SubscribeRequestDto true "Subscribe request payload"
// @Success 200 {object} SubscribeResponseDto "Subscription initialized successfully"
// @Failure 400 {object} entities.Response "Bad request"
// @Failure 401 {object} entities.Response "Unauthorized"
// @Failure 422 {object} entities.Response "Unprocessable entity"
// @Failure 502 {object} entities.Response "Bad gateway"
// @Router /subscription/subscribe [post]
func (h *Handler) Subscribe(c *fiber.Ctx) error {
	userDetails, err := middleware.GetUserDetails(c)
	if err != nil {
		return apperror.New(fiber.StatusUnauthorized, "error", "Unauthorized", err, nil)
	}

	var req SubscribeRequestDto
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

	respData, code, err := h.svc.InitializeSubscription(req, userDetails, db)
	if err != nil {
		return apperror.New(code, "error", err.Error(), err, nil)
	}

	rd := utility.BuildSuccessResponse(http.StatusOK, "subscription initialized successfully", respData)
	return c.Status(code).JSON(rd)
}

// GetSubscriptionStatus godoc
// @Summary Get Current Subscription Status
// @Description Retrieves active subscription details and remaining quota for authenticated business
// @Tags Subscription
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} SubscriptionStatusResponseDto "Subscription status fetched successfully"
// @Failure 401 {object} entities.Response "Unauthorized"
// @Failure 500 {object} entities.Response "Internal server error"
// @Router /subscription/status [get]
func (h *Handler) GetSubscriptionStatus(c *fiber.Ctx) error {
	userDetails, err := middleware.GetUserDetails(c)
	if err != nil {
		return apperror.New(fiber.StatusUnauthorized, "error", "Unauthorized", err, nil)
	}

	db, err := middleware.GetDatabase(c)
	if err != nil {
		return apperror.New(fiber.StatusInternalServerError, "error", err.Error(), err, nil)
	}

	businessID := userDetails.ID
	if userDetails.IsAggregator {
		if queryBizID := strings.TrimSpace(c.Query("business_id")); queryBizID != "" {
			businessID = queryBizID
		}
	}

	status, code, err := h.svc.GetBusinessSubscriptionStatus(db, businessID)
	if err != nil {
		return apperror.New(code, "error", err.Error(), err, nil)
	}

	rd := utility.BuildSuccessResponse(fiber.StatusOK, "subscription status fetched successfully", status)
	return c.Status(fiber.StatusOK).JSON(rd)
}

func (h *Handler) PaystackWebhook(c *fiber.Ctx) error {
	// signature := c.Get("x-paystack-signature")
	// if signature == "" {
	// 	rd := utility.BuildErrorResponse(fiber.StatusUnauthorized, "error", "missing paystack signature", nil, nil)
	// 	return c.Status(fiber.StatusUnauthorized).JSON(rd)
	// }
	rawBody := append([]byte(nil), c.Body()...)

	// log.Println("paystack webhook: ", string(rawBody))
	// if err := h.svc.ValidatePaystackSignature(rawBody, signature); err != nil {
	// 	statusCode := fiber.StatusInternalServerError
	// 	if errors.Is(err, h.svc.ErrInvalidPaystackSignature) {
	// 		statusCode = fiber.StatusUnauthorized
	// 	}

	// 	rd := utility.BuildErrorResponse(statusCode, "error", err.Error(), nil, nil)
	// 	return c.Status(statusCode).JSON(rd)
	// }

	var payload paystack.PaystackWebhookPayload
	if err := json.Unmarshal(rawBody, &payload); err != nil {
		return apperror.New(fiber.StatusBadRequest, "error", "invalid webhook payload", err, nil)
	}

	metadataSandbox, hasMetadataSandbox := payload.MetadataIsSandbox()
	if !hasMetadataSandbox {
		return apperror.New(fiber.StatusBadRequest, "error", "metadata.is_sandbox is required in webhook payload", nil, nil)
	}

	db, err := middleware.GetDatabase(c)
	if err != nil {
		return apperror.New(fiber.StatusInternalServerError, "error", err.Error(), err, nil)
	}
	databaseName := "production"
	if metadataSandbox {
		databaseName = "sandbox"
	}

	go func(payload *paystack.PaystackWebhookPayload, db *gorm.DB, environment, reference string) {
		defer func() {
			if recovered := recover(); recovered != nil {
				h.Logger.Error("paystack webhook async panic (env=%s, ref=%s): %v", environment, reference, recovered)
			}
		}()

		_, code, err := h.svc.HandlePaystackWebhook(payload, db)
		if err != nil {
			h.Logger.Error("paystack webhook async processing failed (env=%s, ref=%s, code=%d): %v", environment, reference, code, err)
			return
		}

		h.Logger.Info("paystack webhook async processing completed (env=%s, ref=%s)", environment, reference)
	}(&payload, db, databaseName, payload.Data.Reference)

	rd := utility.BuildSuccessResponse(http.StatusOK, "webhook accepted for processing", nil)
	return c.Status(http.StatusOK).JSON(rd)
}
