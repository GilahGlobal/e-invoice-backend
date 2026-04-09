package plugin

import (
	"einvoice-access-point/external/firs_models"
	"einvoice-access-point/external/paystack"
	"einvoice-access-point/internal/dtos"
	"einvoice-access-point/internal/services/invoice"
	pluginService "einvoice-access-point/internal/services/plugin"
	"einvoice-access-point/pkg/database"
	"einvoice-access-point/pkg/middleware"
	"einvoice-access-point/pkg/models"
	"einvoice-access-point/pkg/utility"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type Controller struct {
	Db        *database.Database
	TestDB    *database.Database
	Validator *validator.Validate
	Logger    *utility.Logger
}

// CheckBusiness godoc
// @Summary Check Business Subscription
// @Description Checks if a business exists and returns active subscription details when available
// @Tags Plugin
// @Accept json
// @Produce json
// @Param email query string true "Business email"
// @Param aggregator_id query string true "Aggregator ID"
// @Param is_sandbox query string true "Use sandbox database (true/false)"
// @Success 200 {object} dtos.PluginBusinessCheckResponseDto "Business check completed successfully"
// @Failure 400 {object} models.Response "Bad request"
// @Failure 422 {object} models.Response "Unprocessable entity"
// @Failure 500 {object} models.Response "Internal server error"
// @Router /plugin/business [get]
func (base *Controller) CheckBusiness(c *fiber.Ctx) error {
	var req dtos.PluginBusinessCheckQueryDto
	if err := c.QueryParser(&req); err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusBadRequest, "error", "Failed to parse query params", err.Error(), nil)
		return c.Status(fiber.StatusBadRequest).JSON(rd)
	}

	if err := base.Validator.Struct(&req); err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusUnprocessableEntity, "error", "Validation failed", utility.ValidationResponse(err, base.Validator), nil)
		return c.Status(fiber.StatusUnprocessableEntity).JSON(rd)
	}

	isSandbox, err := strconv.ParseBool(req.IsSandbox)
	if err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusBadRequest, "error", "is_sandbox must be true or false", err.Error(), nil)
		return c.Status(fiber.StatusBadRequest).JSON(rd)
	}

	db := middleware.GetDatabaseInstance(isSandbox, base.Db, base.TestDB)

	respData, code, err := pluginService.CheckBusinessWithSubscription(req.Email, req.AggregatorID, db)
	if err != nil {
		rd := utility.BuildErrorResponse(code, "error", err.Error(), err, nil)
		return c.Status(code).JSON(rd)
	}

	rd := utility.BuildSuccessResponse(http.StatusOK, "business check completed successfully", respData)
	return c.Status(code).JSON(rd)
}

// GetPlans godoc
// @Summary List Plugin Plans
// @Description Retrieves all available plans for plugin clients
// @Tags Plugin
// @Accept json
// @Produce json
// @Param is_sandbox query string true "Use sandbox database (true/false)"
// @Success 200 {object} dtos.PluginPlansResponseDto "Plans fetched successfully"
// @Failure 400 {object} models.Response "Bad request"
// @Failure 422 {object} models.Response "Unprocessable entity"
// @Failure 500 {object} models.Response "Internal server error"
// @Router /plugin/plans [get]
func (base *Controller) GetPlans(c *fiber.Ctx) error {
	var query dtos.PluginPlansQueryDto
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
	db := middleware.GetDatabaseInstance(isSandbox, base.Db, base.TestDB)

	plans, err := pluginService.GetAvailablePlans(db)
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusInternalServerError, "error", "failed to fetch plans", err.Error(), nil)
		return c.Status(http.StatusInternalServerError).JSON(rd)
	}

	rd := utility.BuildSuccessResponse(http.StatusOK, "plans fetched successfully", plans)
	return c.Status(http.StatusOK).JSON(rd)
}

// Subscribe godoc
// @Summary Subscribe Business To Plan
// @Description Initializes a Paystack transaction for a business subscription plan
// @Tags Plugin
// @Accept json
// @Produce json
// @Param data body dtos.PluginSubscribeRequestDto true "Subscribe request payload"
// @Success 200 {object} dtos.PluginSubscribeResponseDto "Subscription initialized successfully"
// @Failure 400 {object} models.Response "Bad request"
// @Failure 422 {object} models.Response "Unprocessable entity"
// @Failure 502 {object} models.Response "Bad gateway"
// @Router /plugin/subscribe [post]
func (base *Controller) Subscribe(c *fiber.Ctx) error {
	var req dtos.PluginSubscribeRequestDto
	if err := c.BodyParser(&req); err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusBadRequest, "error", "Failed to parse request body", err.Error(), nil)
		return c.Status(fiber.StatusBadRequest).JSON(rd)
	}

	if err := base.Validator.Struct(&req); err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusUnprocessableEntity, "error", "Validation failed", utility.ValidationResponse(err, base.Validator), nil)
		return c.Status(fiber.StatusUnprocessableEntity).JSON(rd)
	}

	db := middleware.GetDatabaseInstance(req.IsSandbox, base.Db, base.TestDB)

	respData, code, err := pluginService.SubscribeBusinessToPlan(req, db)
	if err != nil {
		rd := utility.BuildErrorResponse(code, "error", err.Error(), err, nil)
		return c.Status(code).JSON(rd)
	}

	rd := utility.BuildSuccessResponse(http.StatusOK, "subscription initialized successfully", respData)
	return c.Status(code).JSON(rd)
}

// PaystackWebhook godoc
// @Summary Handle Paystack Webhook
// @Description Verifies Paystack signature, acknowledges immediately, then processes transaction/subscription updates asynchronously
// @Tags Plugin
// @Accept json
// @Produce json
// @Param x-paystack-signature header string true "Paystack signature"
// @Param payload body object true "Webhook payload"
// @Success 200 {object} dtos.PluginWebhookResponseDto "Webhook accepted for processing"
// @Failure 400 {object} models.Response "Bad request"
// @Failure 401 {object} models.Response "Unauthorized"
// @Failure 500 {object} models.Response "Internal server error"
// @Router /plugin/paystack/webhook [post]
func (base *Controller) PaystackWebhook(c *fiber.Ctx) error {
	signature := c.Get("x-paystack-signature")
	if signature == "" {
		rd := utility.BuildErrorResponse(fiber.StatusUnauthorized, "error", "missing paystack signature", nil, nil)
		return c.Status(fiber.StatusUnauthorized).JSON(rd)
	}
	rawBody := append([]byte(nil), c.Body()...)

	if err := pluginService.ValidatePaystackSignature(rawBody, signature); err != nil {
		statusCode := fiber.StatusInternalServerError
		if errors.Is(err, pluginService.ErrInvalidPaystackSignature) {
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

	go func(body []byte, signature string, selectedDB *database.Database, environment string, reference string) {
		defer func() {
			if recovered := recover(); recovered != nil {
				base.Logger.Error("paystack webhook async panic (env=%s, ref=%s): %v", environment, reference, recovered)
			}
		}()

		_, code, err := pluginService.HandlePaystackWebhook(body, signature, selectedDB.Postgresql.DB())
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

// Register godoc
// @Summary Plugin Register
// @Description Creates a business account in sandbox first, then in production, with default subscription rows
// @Tags Plugin
// @Accept json
// @Produce json
// @Param data body dtos.SmeRegistrationDto true "Register request payload"
// @Success 201 {object} dtos.PluginRegisterResponseDto "Business created successfully"
// @Failure 400 {object} models.Response "Bad request"
// @Failure 422 {object} models.Response "Unprocessable entity"
// @Failure 500 {object} models.Response "Internal server error"
// @Router /plugin/register [post]
func (base *Controller) Register(c *fiber.Ctx) error {
	var req dtos.SmeRegistrationDto

	err := c.BodyParser(&req)
	if err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusBadRequest, "error", "Failed to parse request body", err.Error(), nil)
		return c.Status(fiber.StatusBadRequest).JSON(rd)
	}

	err = base.Validator.Struct(&req)
	if err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusUnprocessableEntity, "error", "Validation failed", utility.ValidationResponse(err, base.Validator), nil)
		return c.Status(fiber.StatusUnprocessableEntity).JSON(rd)
	}

	reqData, err := pluginService.ValidateCreateUserRequest(req, base.TestDB.Postgresql.DB())
	if err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusBadRequest, "error", err.Error(), err, nil)
		return c.Status(fiber.StatusBadRequest).JSON(rd)
	}

	sandboxRespData, code, err := pluginService.RegisterUserWithSubscription(reqData, base.TestDB.Postgresql.DB(), true)
	if err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusBadRequest, "error", err.Error(), err, nil)
		return c.Status(fiber.StatusBadRequest).JSON(rd)
	}

	// prodRespData, code, err := pluginService.RegisterUserWithSubscription(reqData, base.Db.Postgresql.DB(), false)
	// if err != nil {
	// 	rd := utility.BuildErrorResponse(code, "error", "sandbox account was created but failed to create production account", err, fiber.Map{
	// 		"sandbox": sandboxRespData,
	// 	})
	// 	return c.Status(code).JSON(rd)
	// }

	base.Logger.Info("plugin user created successfully")
	rd := utility.BuildSuccessResponse(fiber.StatusCreated, "plugin user created successfully in sandbox and production", fiber.Map{
		"sandbox": sandboxRespData,
		// "production": prodRespData,
	})
	return c.Status(code).JSON(rd)
}

// UploadInvoice godoc
// @Summary Initializes invoice creation in one go
// @Description Receives invoice data as a json
// @Tags Internal Invoice
// @Accept json
// @Produce json
// @Security
// @Param   payload  body  dtos.UploadInvoiceRequestDto  true  "Invoice Payload"
// @Success 200 {object} dtos.UploadInvoiceResponseDto "Invoice created successfully"
// @Failure 400 {object} models.Response "Bad request"
// @Failure 403 {object} models.Response "Subscription is inactive or invoice quota exhausted"
// @Router /plugin/invoice-upload [post]
func (base *Controller) UploadInvoice(c *fiber.Ctx) error {

	userDetails, err := middleware.GetUserDetails(c)
	if err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusBadRequest, "error", "unable to get user claims", nil, nil)
		return c.Status(fiber.StatusBadRequest).JSON(rd)
	}

	db := middleware.GetDatabaseInstance(userDetails.IsSandbox, base.Db, base.TestDB)

	var req dtos.UploadInvoiceRequestDto

	err = c.BodyParser(&req)
	if err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusBadRequest, "error", "Failed to parse request body", err, nil)
		return c.Status(fiber.StatusBadRequest).JSON(rd)
	}

	err = base.Validator.Struct(&req)
	if err != nil {
		rd := utility.BuildErrorResponse(
			fiber.StatusUnprocessableEntity,
			"error", "Validation failed",
			utility.ValidationErrorsToJSON(err, firs_models.InvoiceRequest{}),
			nil,
		)
		return c.Status(fiber.StatusUnprocessableEntity).JSON(rd)
	}

	if req.SmeID == nil {
		rd := utility.BuildErrorResponse(fiber.StatusBadRequest, "error", "sme_id is required", err, nil)
		return c.Status(fiber.StatusBadRequest).JSON(rd)
	}

	smeID := *req.SmeID

	_, err = uuid.Parse(smeID)
	if err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusBadRequest, "error", "sme_id is invalid", err, nil)
		return c.Status(fiber.StatusBadRequest).JSON(rd)
	}

	isPluginUser, err := invoice.ValidatePluginInvoiceEligibility(db, smeID)
	if isPluginUser && err != nil {
		switch {
		case errors.Is(err, invoice.ErrPluginSubscriptionRequired):
			rd := utility.BuildErrorResponse(fiber.StatusForbidden, "error", err.Error(), nil, nil)
			return c.Status(fiber.StatusForbidden).JSON(rd)
		case errors.Is(err, invoice.ErrPluginInvoiceQuotaExceeded):
			rd := utility.BuildErrorResponse(fiber.StatusForbidden, "error", err.Error(), nil, nil)
			return c.Status(fiber.StatusForbidden).JSON(rd)
		default:
			rd := utility.BuildErrorResponse(fiber.StatusInternalServerError, "error", err.Error(), nil, nil)
			return c.Status(fiber.StatusInternalServerError).JSON(rd)
		}
	}

	invoiceExists, err := invoice.GetInvoiceByInvoiceNumber(db, req.InvoiceNumber, userDetails.ID)
	if err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusBadRequest, "error", err.Error(), err, nil)
		return c.Status(fiber.StatusBadRequest).JSON(rd)
	}

	if invoiceExists != nil {
		blockedStatuses := map[string]bool{
			models.StatusSignedInvoice: true,
			models.StatusTransmitted:   true,
			models.StatusConfirmed:     true,
		}
		if blockedStatuses[invoiceExists.CurrentStatus] {
			rd := utility.BuildErrorResponse(fiber.StatusBadRequest, "error", "invoice with the same invoice number already exists and cannot be overwritten", nil, nil)
			return c.Status(fiber.StatusBadRequest).JSON(rd)
		}
	}

	var irnPayload dtos.InvoiceData
	if req.IRN == nil {
		IRNData, err := invoice.IRNGeneration(db, userDetails.ID, req.InvoiceNumber, userDetails.ServiceID, *userDetails.BusinessID, userDetails.IsSandbox)
		if err != nil {
			rd := *err
			return c.Status(fiber.StatusBadRequest).JSON(rd)
		}
		irnPayload = *IRNData
		req.IRN = &irnPayload.IRN
	} else {
		irnPayload = dtos.InvoiceData{
			InvoiceNumber: req.InvoiceNumber,
			IRN:           *req.IRN,
			QRCode:        invoiceExists.QrCode,
			QRCode2:       invoiceExists.EncryptedIRN,
		}
	}

	createdInvoice, _, err, isInvoiceSigned := invoice.CreateInvoice(db, req, req.InvoiceNumber, userDetails.ID, irnPayload.QRCode, irnPayload.QRCode2, invoiceExists, userDetails.IsSandbox, nil)

	response := map[string]interface{}{
		"metadata": createdInvoice.StatusHistory,
	}
	if isInvoiceSigned {
		response["data"] = irnPayload
	}

	if err != nil {
		errorArray := strings.Split(err.Error(), "-")
		rd := utility.BuildErrorResponse(fiber.StatusBadRequest, "error", errorArray[len(errorArray)-1], response, nil)
		return c.Status(fiber.StatusBadRequest).JSON(rd)
	}

	if isPluginUser && invoiceExists == nil {
		if err := invoice.ConsumePluginInvoiceQuota(db, userDetails.ID); err != nil {
			switch {
			case errors.Is(err, invoice.ErrPluginSubscriptionRequired), errors.Is(err, invoice.ErrPluginInvoiceQuotaExceeded):
				rd := utility.BuildErrorResponse(fiber.StatusForbidden, "error", err.Error(), nil, nil)
				return c.Status(fiber.StatusForbidden).JSON(rd)
			default:
				rd := utility.BuildErrorResponse(fiber.StatusInternalServerError, "error", err.Error(), nil, nil)
				return c.Status(fiber.StatusInternalServerError).JSON(rd)
			}
		}
	}

	rd := utility.BuildSuccessResponse(fiber.StatusCreated, "Invoice created successfully", response)
	return c.Status(fiber.StatusCreated).JSON(rd)
}
