package invoice

import (
	"einvoice-access-point/external/firs_models"
	"einvoice-access-point/internal/dtos"
	"einvoice-access-point/internal/services/invoice"
	"einvoice-access-point/pkg/database"
	"einvoice-access-point/pkg/middleware"
	"einvoice-access-point/pkg/utility"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
)

type Controller struct {
	Db        *database.Database
	TestDB    *database.Database
	Validator *validator.Validate
	Logger    *utility.Logger
	Keys      *utility.CryptoKeys
}

// ValidateIRN godoc
// @Summary Validate IRN
// @Description Validates an Invoice Reference Number (IRN).
// @Tags Invoice
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body firs_models.IRNValidationRequest true "IRN Validation Request"
// @Success 200 {object} models.Response "IRN validated successfully"
// @Failure 400 {object} models.Response "Bad request"
// @Failure 422 {object} models.Response "Validation failed"
// @Router /invoice/validate-irn [post]
func (base *Controller) ValidateIRN(c *fiber.Ctx) error {
	userDetails, err := middleware.GetUserDetails(c)
	if err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusBadRequest, "error", "unable to get user claims", nil, nil)
		return c.Status(fiber.StatusBadRequest).JSON(rd)
	}

	var req firs_models.IRNValidationRequest

	err = c.BodyParser(&req)
	if err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusBadRequest, "error", "Failed to parse request body", err, nil)
		return c.Status(fiber.StatusBadRequest).JSON(rd)
	}

	err = base.Validator.Struct(&req)
	if err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusUnprocessableEntity, "error", "Validation failed", utility.ValidationResponse(err, base.Validator), nil)
		return c.Status(fiber.StatusUnprocessableEntity).JSON(rd)
	}

	respData, errDetails, err := invoice.ValidateIRN(req, userDetails.IsSandbox)
	if err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusBadRequest, "error", err.Error(), errDetails, nil)
		return c.Status(fiber.StatusBadRequest).JSON(rd)
	}

	base.Logger.Info("IRN validated successfully")
	rd := utility.BuildSuccessResponse(fiber.StatusOK, "IRN validated successfully", respData)
	return c.Status(fiber.StatusOK).JSON(rd)
}

// ValidateInvoice godoc
// @Summary Validate Invoice
// @Description Validates an invoice payload.
// @Tags Invoice
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body dtos.UploadInvoiceRequestDto true "Invoice Request"
// @Success 200 {object} models.Response "Invoice validated successfully"
// @Failure 400 {object} models.Response "Bad request"
// @Failure 422 {object} models.Response "Validation failed"
// @Router /invoice/validate [post]
func (base *Controller) ValidateInvoice(c *fiber.Ctx) error {
	userDetails, err := middleware.GetUserDetails(c)
	if err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusBadRequest, "error", "unable to get user claims", nil, nil)
		return c.Status(fiber.StatusBadRequest).JSON(rd)
	}
	var req dtos.UploadInvoiceRequestDto

	err = c.BodyParser(&req)
	if err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusBadRequest, "error", "Failed to parse request body", err, nil)
		return c.Status(fiber.StatusBadRequest).JSON(rd)
	}

	err = base.Validator.Struct(&req)
	if err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusUnprocessableEntity, "error", "Validation failed", utility.ValidationResponse(err, base.Validator), nil)
		return c.Status(fiber.StatusUnprocessableEntity).JSON(rd)
	}

	respData, errDetails, err := invoice.ValidateInvoice(req, userDetails.IsSandbox)
	if err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusBadRequest, "error", err.Error(), errDetails, nil)
		return c.Status(fiber.StatusBadRequest).JSON(rd)
	}

	base.Logger.Info("Invoice validated successfully")
	rd := utility.BuildSuccessResponse(fiber.StatusOK, "Invoice validated successfully", respData)
	return c.Status(fiber.StatusOK).JSON(rd)
}
