package invoice

import (
	"einvoice-access-point/external/firs_models"
	"einvoice-access-point/internal/dtos"
	"einvoice-access-point/internal/services/invoice"
	"einvoice-access-point/pkg/middleware"
	"einvoice-access-point/pkg/utility"

	"github.com/gofiber/fiber/v2"
)

// SignIRN godoc
// @Summary Sign IRN
// @Description Signs an IRN and generates a QR code.
// @Tags Invoice
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body firs_models.IRNSigningRequestData true "IRN Signing Request"
// @Success 200 {object} models.Response "IRN signed successfully"
// @Failure 400 {object} models.Response "Bad request"
// @Failure 422 {object} models.Response "Validation failed"
// @Router /invoice/sign-irn [post]
func (base *Controller) SignIRN(c *fiber.Ctx) error {
	var req firs_models.IRNSigningRequestData

	err := c.BodyParser(&req)
	if err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusBadRequest, "error", "Failed to parse request body", err, nil)
		return c.Status(fiber.StatusBadRequest).JSON(rd)
	}

	err = base.Validator.Struct(&req)
	if err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusUnprocessableEntity, "error", "Validation failed", utility.ValidationResponse(err, base.Validator), nil)
		return c.Status(fiber.StatusUnprocessableEntity).JSON(rd)
	}

	respData, err := invoice.SignIRN(req.IRN, base.Keys)
	if err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusBadRequest, "error", err.Error(), err, nil)
		return c.Status(fiber.StatusBadRequest).JSON(rd)
	}

	base.Logger.Info("qr code generated successfully")
	rd := utility.BuildSuccessResponse(fiber.StatusOK, "successfully", respData)
	return c.Status(fiber.StatusOK).JSON(rd)
}

// SignInvoice godoc
// @Summary Sign Invoice
// @Description Signs an invoice and generates a digital signature.
// @Tags Invoice
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body dtos.UploadInvoiceRequestDto true "Invoice Request"
// @Success 201 {object} models.Response "Invoice signed successfully"
// @Failure 400 {object} models.Response "Bad request"
// @Failure 422 {object} models.Response "Validation failed"
// @Router /invoice/sign [post]
func (base *Controller) SignInvoice(c *fiber.Ctx) error {
	userDetails, err := middleware.GetUserDetails(c)
	if err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusUnauthorized, "error", "Unauthorized", err, nil)
		return c.Status(fiber.StatusUnauthorized).JSON(rd)
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

	respData, errDetails, err := invoice.SignInvoice(req, userDetails.IsSandbox)
	if err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusBadRequest, "error", err.Error(), errDetails, nil)
		return c.Status(fiber.StatusBadRequest).JSON(rd)
	}

	base.Logger.Info("Invoice signed successfully")
	rd := utility.BuildSuccessResponse(fiber.StatusCreated, "Invoice signed successfully", respData)
	return c.Status(fiber.StatusCreated).JSON(rd)
}

// GenerateIRN godoc
// @Summary Generate IRN
// @Description Generates a new IRN for an invoice.
// @Tags Invoice
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body firs_models.GenerateIRNRequestData true "Generate IRN Request"
// @Success 200 {object} models.Response "IRN generated successfully"
// @Failure 400 {object} models.Response "Bad request"
// @Failure 422 {object} models.Response "Validation failed"
// @Router /invoice/generate-irn [post]
func (base *Controller) GenerateIRN(c *fiber.Ctx) error {
	var req firs_models.GenerateIRNRequestData

	userDetails, err := middleware.GetUserDetails(c)
	if err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusBadRequest, "error", "unable to get user claims", nil, nil)
		return c.Status(fiber.StatusBadRequest).JSON(rd)
	}

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

	respData, err := invoice.GenerateIRN(req.InvoiceNumber, userDetails.ServiceID)
	if err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusBadRequest, "error", err.Error(), err, nil)
		return c.Status(fiber.StatusBadRequest).JSON(rd)
	}

	base.Logger.Info("IRN generated successfully")
	rd := utility.BuildSuccessResponse(fiber.StatusOK, "IRN generated successfully", respData)
	return c.Status(fiber.StatusOK).JSON(rd)
}
