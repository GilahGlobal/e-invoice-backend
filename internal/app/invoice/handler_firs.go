package invoice

import (
	"einvoice-access-point/internal/apperror"
	"einvoice-access-point/internal/middleware"
	"einvoice-access-point/internal/pkg/firs_models"
	"einvoice-access-point/internal/utility"

	"github.com/gofiber/fiber/v2"
)

// ValidateIRN godoc
// @Summary Validate IRN
// @Description Validates an Invoice Reference Number (IRN).
// @Tags Invoice
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body firs_models.IRNValidationRequest true "IRN Validation Request"
// @Success 200 {object} entities.Response "IRN validated successfully"
// @Failure 400 {object} entities.Response "Bad request"
// @Failure 422 {object} entities.Response "Validation failed"
// @Router /invoice/validate-irn [post]
func (h *Handler) ValidateIRN(c *fiber.Ctx) error {
	userDetails, err := middleware.GetUserDetails(c)
	if err != nil {
		return apperror.New(fiber.StatusBadRequest, "error", "unable to get user claims", nil, nil)
	}

	var req firs_models.IRNValidationRequest
	if err = c.BodyParser(&req); err != nil {
		return apperror.New(fiber.StatusBadRequest, "error", "Failed to parse request body", err, nil)
	}

	if err = h.Validator.Struct(&req); err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusUnprocessableEntity, "error", "Validation failed", utility.ValidationResponse(err, h.Validator), nil)
		return c.Status(fiber.StatusUnprocessableEntity).JSON(rd)
	}

	respData, errDetails, err := h.svc.ValidateIRN(req, userDetails.IsSandbox)
	if err != nil {
		return apperror.New(fiber.StatusBadRequest, "error", err.Error(), errDetails, nil)
	}

	h.Logger.Info("IRN validated successfully")
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
// @Param request body firs_models.UploadInvoiceRequestDto true "Invoice Request"
// @Success 200 {object} entities.Response "Invoice validated successfully"
// @Failure 400 {object} entities.Response "Bad request"
// @Failure 422 {object} entities.Response "Validation failed"
// @Router /invoice/validate [post]
func (h *Handler) ValidateInvoice(c *fiber.Ctx) error {
	userDetails, err := middleware.GetUserDetails(c)
	if err != nil {
		return apperror.New(fiber.StatusBadRequest, "error", "unable to get user claims", nil, nil)
	}

	var req firs_models.UploadInvoiceRequestDto
	if err = c.BodyParser(&req); err != nil {
		return apperror.New(fiber.StatusBadRequest, "error", "Failed to parse request body", err, nil)
	}

	if err = h.Validator.Struct(&req); err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusUnprocessableEntity, "error", "Validation failed", utility.ValidationResponse(err, h.Validator), nil)
		return c.Status(fiber.StatusUnprocessableEntity).JSON(rd)
	}

	respData, errDetails, err := h.svc.ValidateInvoice(req, userDetails.IsSandbox)
	if err != nil {
		return apperror.New(fiber.StatusBadRequest, "error", err.Error(), errDetails, nil)
	}

	h.Logger.Info("Invoice validated successfully")
	rd := utility.BuildSuccessResponse(fiber.StatusOK, "Invoice validated successfully", respData)
	return c.Status(fiber.StatusOK).JSON(rd)
}

// SignIRN godoc
// @Summary Sign IRN
// @Description Signs an IRN and generates a QR code.
// @Tags Invoice
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body firs_models.IRNSigningRequestData true "IRN Signing Request"
// @Success 200 {object} entities.Response "IRN signed successfully"
// @Failure 400 {object} entities.Response "Bad request"
// @Failure 422 {object} entities.Response "Validation failed"
// @Router /invoice/sign-irn [post]
func (h *Handler) SignIRN(c *fiber.Ctx) error {
	userDetails, err := middleware.GetUserDetails(c)
	if err != nil {
		return apperror.New(fiber.StatusUnauthorized, "error", "Unauthorized", err, nil)
	}

	db, err := middleware.GetDatabase(c)
	if err != nil {
		return apperror.New(fiber.StatusInternalServerError, "error", err.Error(), err, nil)
	}

	var req firs_models.IRNSigningRequestData
	if err = c.BodyParser(&req); err != nil {
		return apperror.New(fiber.StatusBadRequest, "error", "Failed to parse request body", err, nil)
	}

	if err = h.Validator.Struct(&req); err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusUnprocessableEntity, "error", "Validation failed", utility.ValidationResponse(err, h.Validator), nil)
		return c.Status(fiber.StatusUnprocessableEntity).JSON(rd)
	}

	keys, err := h.businessSvc.ResolveBusinessIRNSigningKeys(db, userDetails.ID, userDetails.IsSandbox, h.Keys)
	if err != nil {
		return apperror.New(fiber.StatusBadRequest, "error", err.Error(), nil, nil)
	}

	respData, err := h.svc.SignIRN(req.IRN, keys)
	if err != nil {
		return apperror.New(fiber.StatusBadRequest, "error", err.Error(), err, nil)
	}

	h.Logger.Info("qr code generated successfully")
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
// @Param request body firs_models.UploadInvoiceRequestDto true "Invoice Request"
// @Success 201 {object} entities.Response "Invoice signed successfully"
// @Failure 400 {object} entities.Response "Bad request"
// @Failure 422 {object} entities.Response "Validation failed"
// @Router /invoice/sign [post]
func (h *Handler) SignInvoice(c *fiber.Ctx) error {
	userDetails, err := middleware.GetUserDetails(c)
	if err != nil {
		return apperror.New(fiber.StatusUnauthorized, "error", "Unauthorized", err, nil)
	}

	var req firs_models.UploadInvoiceRequestDto
	if err = c.BodyParser(&req); err != nil {
		return apperror.New(fiber.StatusBadRequest, "error", "Failed to parse request body", err, nil)
	}

	if err = h.Validator.Struct(&req); err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusUnprocessableEntity, "error", "Validation failed", utility.ValidationResponse(err, h.Validator), nil)
		return c.Status(fiber.StatusUnprocessableEntity).JSON(rd)
	}

	respData, errDetails, err := h.svc.SignInvoice(req, userDetails.IsSandbox)
	if err != nil {
		return apperror.New(fiber.StatusBadRequest, "error", err.Error(), errDetails, nil)
	}

	h.Logger.Info("Invoice signed successfully")
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
// @Success 200 {object} entities.Response "IRN generated successfully"
// @Failure 400 {object} entities.Response "Bad request"
// @Failure 422 {object} entities.Response "Validation failed"
// @Router /invoice/generate-irn [post]
func (h *Handler) GenerateIRN(c *fiber.Ctx) error {
	var req firs_models.GenerateIRNRequestData

	userDetails, err := middleware.GetUserDetails(c)
	if err != nil {
		return apperror.New(fiber.StatusBadRequest, "error", "unable to get user claims", nil, nil)
	}

	if err = c.BodyParser(&req); err != nil {
		return apperror.New(fiber.StatusBadRequest, "error", "Failed to parse request body", err, nil)
	}

	if err = h.Validator.Struct(&req); err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusUnprocessableEntity, "error", "Validation failed", utility.ValidationResponse(err, h.Validator), nil)
		return c.Status(fiber.StatusUnprocessableEntity).JSON(rd)
	}

	respData, err := h.svc.GenerateIRN(req.InvoiceNumber, *userDetails.ServiceID)
	if err != nil {
		return apperror.New(fiber.StatusBadRequest, "error", err.Error(), err, nil)
	}

	h.Logger.Info("IRN generated successfully")
	rd := utility.BuildSuccessResponse(fiber.StatusOK, "IRN generated successfully", respData)
	return c.Status(fiber.StatusOK).JSON(rd)
}

// UpdateInvoice godoc
// @Summary Update Invoice
// @Description Updates an existing invoice using the IRN.
// @Tags Invoice
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param irn path string true "Invoice Reference Number (IRN)"
// @Param request body firs_models.UpdateInvoice true "Update Invoice Request"
// @Success 200 {object} entities.Response "Invoice updated successfully"
// @Failure 400 {object} entities.Response "Bad request"
// @Failure 422 {object} entities.Response "Validation failed"
// @Router /invoice/update/{irn} [patch]
func (h *Handler) UpdateInvoice(c *fiber.Ctx) error {
	userDetails, err := middleware.GetUserDetails(c)
	if err != nil {
		return apperror.New(fiber.StatusUnauthorized, "error", "Unauthorized", err, nil)
	}

	db, err := middleware.GetDatabase(c)
	if err != nil {
		return apperror.New(fiber.StatusInternalServerError, "error", err.Error(), err, nil)
	}

	irn := c.Params("irn")
	if irn == "" {
		return apperror.New(fiber.StatusBadRequest, "error", "irn is required", nil, nil)
	}

	var req firs_models.UpdateInvoice
	if err = c.BodyParser(&req); err != nil {
		return apperror.New(fiber.StatusBadRequest, "error", "Failed to parse request body", err, nil)
	}

	if err = h.Validator.Struct(&req); err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusUnprocessableEntity, "error", "Validation failed", utility.ValidationResponse(err, h.Validator), nil)
		return c.Status(fiber.StatusUnprocessableEntity).JSON(rd)
	}

	respData, errDetails, err := h.svc.UpdateInvoice(req, irn, userDetails.IsSandbox)
	if err != nil {
		return apperror.New(fiber.StatusBadRequest, "error", err.Error(), errDetails, nil)
	}

	if err := h.svc.UpdateStoredInvoicePaymentStatus(db, userDetails.ID, irn, req.PaymentStatus); err != nil {
		return apperror.New(
			fiber.StatusInternalServerError, "error", "invoice updated on FIRS but failed to update local invoice record", err, nil,
		)
	}

	h.Logger.Info("Invoice updated successfully")
	rd := utility.BuildSuccessResponse(fiber.StatusOK, "Invoice updated successfully", respData)
	return c.Status(fiber.StatusOK).JSON(rd)
}

// ConfirmInvoice godoc
// @Summary Confirm Invoice
// @Description Confirms an invoice with IRN.
// @Tags Invoice
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param irn path string true "Invoice Reference Number (IRN)"
// @Success 200 {object} entities.Response "Invoice confirmed successfully"
// @Failure 400 {object} entities.Response "Bad request"
// @Router /invoice/confirm/{irn} [get]
func (h *Handler) ConfirmInvoice(c *fiber.Ctx) error {
	userDetails, err := middleware.GetUserDetails(c)
	if err != nil {
		return apperror.New(fiber.StatusBadRequest, "error", "unable to get user claims", nil, nil)
	}

	irn := c.Params("irn")
	if irn == "" {
		return apperror.New(fiber.StatusBadRequest, "error", "irn is required", nil, nil)
	}

	respData, errDetails, err := h.svc.ConfirmInvoice(irn, userDetails.IsSandbox)
	if err != nil {
		return apperror.New(fiber.StatusBadRequest, "error", err.Error(), errDetails, nil)
	}

	h.Logger.Info("Invoice confirmed with irn successfully")
	rd := utility.BuildSuccessResponse(fiber.StatusOK, "Invoice confirmed with irn successfully", respData)
	return c.Status(fiber.StatusOK).JSON(rd)
}

// DownloadInvoice godoc
// @Summary Download Invoice
// @Description Downloads an invoice from FIRS using the IRN.
// @Tags Invoice
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param irn path string true "Invoice Reference Number (IRN)"
// @Success 200 {object} entities.Response "Invoice downloaded successfully"
// @Failure 400 {object} entities.Response "Bad request"
// @Router /invoice/download/{irn} [get]
func (h *Handler) DownloadInvoice(c *fiber.Ctx) error {
	userDetails, err := middleware.GetUserDetails(c)
	if err != nil {
		return apperror.New(fiber.StatusBadRequest, "error", "unable to get user claims", nil, nil)
	}

	irn := c.Params("irn")
	if irn == "" {
		return apperror.New(fiber.StatusBadRequest, "error", "irn is required", nil, nil)
	}

	respData, errDetails, err := h.svc.DownloadInvoice(irn, userDetails.IsSandbox)
	if err != nil {
		return apperror.New(fiber.StatusBadRequest, "error", err.Error(), errDetails, nil)
	}

	h.Logger.Info("Invoice downloaded with irn successfully")
	rd := utility.BuildSuccessResponse(fiber.StatusOK, "Invoice downloaded with irn successfully", respData)
	return c.Status(fiber.StatusOK).JSON(rd)
}
