package invoice

import (
	"einvoice-access-point/internal/apperror"
	"einvoice-access-point/internal/middleware"
	"einvoice-access-point/internal/pkg/firs_models"
	"einvoice-access-point/internal/pkg/zoho"
	"einvoice-access-point/internal/utility"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

func (h *Handler) HandleZohoWebhook(c *fiber.Ctx) error {
	organisationID := c.Query("organisation_id")

	db, err := middleware.GetDatabase(c)
	if err != nil {
		return apperror.New(fiber.StatusInternalServerError, "error", err.Error(), err, nil)
	}
	if organisationID == "" {
		return apperror.New(fiber.StatusBadRequest, "error", "No organisation ID present", nil, nil)
	}

	var payload zoho.WebhookPayload
	if err := c.BodyParser(&payload); err != nil {
		h.Logger.Error("Failed to parse request body", zap.Error(err))
		return apperror.New(fiber.StatusBadRequest, "error", "Failed to parse request body", err, nil)
	}

	if err := h.Validator.Struct(&payload); err != nil {
		h.Logger.Error("Validation failed", zap.Error(err))
		rd := utility.BuildErrorResponse(fiber.StatusUnprocessableEntity, "error", "Validation failed", utility.ValidationResponse(err, h.Validator), nil)
		return c.Status(fiber.StatusUnprocessableEntity).JSON(rd)
	}

	signature := c.Get("X-Zoho-Signature")
	respData, errDetails, err := h.svc.HandleZohoWebhookService(payload, string(c.Body()), signature, db, h.Logger, h.Keys, organisationID, true)
	if err != nil {
		if err.Error() == ErrInvalidSignature.Error() {
			return apperror.New(fiber.StatusUnauthorized, "error", "Invalid webhook signature", nil, nil)
		}
		return apperror.New(fiber.StatusBadRequest, "error", err.Error(), errDetails, nil)
	}

	rd := utility.BuildSuccessResponse(fiber.StatusOK, "Webhook processed successfully", respData)
	return c.Status(fiber.StatusOK).JSON(rd)
}

func (h *Handler) FirsWebhook(c *fiber.Ctx) error {
	var req firs_models.FirsWebhookPayload

	if err := c.BodyParser(&req); err != nil {
		return apperror.New(fiber.StatusBadRequest, "error", "Failed to parse request body", err, nil)
	}

	if err := h.Validator.Struct(&req); err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusUnprocessableEntity, "error", "Validation failed", utility.ValidationResponse(err, h.Validator), nil)
		return c.Status(fiber.StatusUnprocessableEntity).JSON(rd)
	}

	if err := h.svc.ProcessFirsWebhook(req); err != nil {
		return apperror.New(fiber.StatusBadRequest, "error", err.Error(), err, nil)
	}

	h.Logger.Info("Webhook successfully reached for irn: %s", req.IRN)
	rd := utility.BuildSuccessResponse(fiber.StatusOK, "successful", nil)
	return c.Status(fiber.StatusOK).JSON(rd)
}
