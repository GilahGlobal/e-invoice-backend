package invoice

import (
	"einvoice-access-point/external/zoho"
	services "einvoice-access-point/internal/services/webhooks"
	"einvoice-access-point/pkg/middleware"
	"einvoice-access-point/pkg/utility"
	"fmt"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

func (base *Controller) HandleZohoWebhook(c *fiber.Ctx) error {
	body := c.Body()
	fmt.Printf("Webhook body is: %s\n\n", string(body))

	organisationID := c.Query("organisation_id")
	userDetails, err := middleware.GetUserDetails(c)
	if err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusBadRequest, "error", "unable to get user claims", nil, nil)
		return c.Status(fiber.StatusBadRequest).JSON(rd)
	}

	db := middleware.GetDatabaseInstance(userDetails.IsSandbox, base.Db, base.TestDB)
	if organisationID == "" {
		rd := utility.BuildErrorResponse(fiber.StatusBadRequest, "error", "No organisation ID present", nil, nil)
		return c.Status(fiber.StatusBadRequest).JSON(rd)
	}

	var payload zoho.WebhookPayload
	if err := c.BodyParser(&payload); err != nil {
		base.Logger.Error("Failed to parse request body", zap.Error(err))
		rd := utility.BuildErrorResponse(fiber.StatusBadRequest, "error", "Failed to parse request body", err.Error(), nil)
		return c.Status(fiber.StatusBadRequest).JSON(rd)
	}

	if err := base.Validator.Struct(&payload); err != nil {
		base.Logger.Error("Validation failed", zap.Error(err))
		rd := utility.BuildErrorResponse(fiber.StatusUnprocessableEntity, "error", "Validation failed", utility.ValidationResponse(err, base.Validator), nil)
		return c.Status(fiber.StatusUnprocessableEntity).JSON(rd)
	}

	// Keep raw body for signature check
	signature := c.Get("X-Zoho-Signature")

	respData, errDetails, err := services.HandleZohoWebhookService(payload, string(c.Body()), signature, db, base.Logger, base.Keys, organisationID, userDetails.IsSandbox)
	if err != nil {
		if err == services.ErrInvalidSignature {
			rd := utility.BuildErrorResponse(fiber.StatusUnauthorized, "error", "Invalid webhook signature", nil, nil)
			return c.Status(fiber.StatusUnauthorized).JSON(rd)
		}
		rd := utility.BuildErrorResponse(fiber.StatusBadRequest, "error", err.Error(), errDetails, nil)
		return c.Status(fiber.StatusBadRequest).JSON(rd)
	}

	rd := utility.BuildSuccessResponse(fiber.StatusOK, "Webhook processed successfully", respData)
	return c.Status(fiber.StatusOK).JSON(rd)
}
