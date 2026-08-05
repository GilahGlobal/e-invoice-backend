package invoice

import (
	"einvoice-access-point/internal/apperror"
	"einvoice-access-point/internal/data/entities"
	"einvoice-access-point/internal/middleware"
	"einvoice-access-point/internal/utility"

	"github.com/gofiber/fiber/v2"
)

// LookUpIRN godoc
// @Summary Look Up IRN
// @Description Retrieves invoice details using the IRN.
// @Tags Invoice
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param irn path string true "Invoice Reference Number (IRN)"
// @Success 200 {object} entities.Response "Invoice details retrieved"
// @Failure 400 {object} entities.Response "Bad request"
// @Router /invoice/transmit/lookup-irn/{irn} [get]
func (h *Handler) LookUpIRN(c *fiber.Ctx) error {
	irn := c.Params("irn")
	if irn == "" {
		return apperror.New(fiber.StatusBadRequest, "error", "irn is required", nil, nil)
	}

	userDetails, err := middleware.GetUserDetails(c)
	if err != nil {
		return apperror.New(fiber.StatusBadRequest, "error", "unable to get user claims", nil, nil)
	}

	db, err := middleware.GetDatabase(c)
	if err != nil {
		return apperror.New(fiber.StatusInternalServerError, "error", err.Error(), err, nil)
	}

	if _, err := h.svc.GetInvoiceByIRN(db, irn, userDetails.ID); err != nil {
		return apperror.New(fiber.StatusNotFound, "error", "invoice not found or does not belong to you", err, nil)
	}

	respData, errDetails, err := h.svc.LookUpIRN(irn)
	if err != nil {
		return apperror.New(fiber.StatusBadRequest, "error", err.Error(), errDetails, nil)
	}

	h.Logger.Info("successfully")
	rd := utility.BuildSuccessResponse(fiber.StatusOK, "successfully", respData)
	return c.Status(fiber.StatusOK).JSON(rd)
}

// LookUpTIN godoc
// @Summary Look Up TIN
// @Description Retrieves taxpayer details using TIN.
// @Tags Invoice
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param tin path string true "Tax Identification Number (TIN)"
// @Success 200 {object} entities.Response "TIN details retrieved"
// @Failure 400 {object} entities.Response "Bad request"
// @Router /invoice/transmit/lookup-tin/{tin} [get]
func (h *Handler) LookUpTIN(c *fiber.Ctx) error {
	tin := c.Params("tin")
	userDetails, _ := middleware.GetUserDetails(c)
	if tin == "" {
		return apperror.New(fiber.StatusBadRequest, "error", "tin is required", nil, nil)
	}

	respData, errDetails, err := h.svc.LookUpTIN(tin, userDetails.IsSandbox)
	if err != nil {
		return apperror.New(fiber.StatusBadRequest, "error", err.Error(), errDetails, nil)
	}

	h.Logger.Info("successfully")
	rd := utility.BuildSuccessResponse(fiber.StatusOK, "successfully", respData)
	return c.Status(fiber.StatusOK).JSON(rd)
}

// LookUpPartyID godoc
// @Summary Look Up Party ID
// @Description Retrieves details using Party ID.
// @Tags Invoice
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param party_id path string true "Party ID"
// @Success 200 {object} entities.Response "Party ID details retrieved"
// @Failure 400 {object} entities.Response "Bad request"
// @Router /invoice/transmit/lookup-party/{party_id} [get]
func (h *Handler) LookUpPartyID(c *fiber.Ctx) error {
	partyId := c.Params("party_id")
	if partyId == "" {
		return apperror.New(fiber.StatusBadRequest, "error", "partyID is required", nil, nil)
	}

	respData, errDetails, err := h.svc.LookUpPartyID(partyId)
	if err != nil {
		return apperror.New(fiber.StatusBadRequest, "error", err.Error(), errDetails, nil)
	}

	h.Logger.Info("successfully")
	rd := utility.BuildSuccessResponse(fiber.StatusOK, "successfully", respData)
	return c.Status(fiber.StatusOK).JSON(rd)
}

// TransmitInvoice godoc
// @Summary Transmit Invoice
// @Description Transmits an invoice to FIRS using the IRN.
// @Tags Invoice
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param irn path string true "Invoice Reference Number (IRN)"
// @Success 200 {object} entities.Response "Invoice transmitted successfully"
// @Failure 400 {object} entities.Response "Bad request"
// @Router /invoice/transmit/{irn} [post]
func (h *Handler) TransmitInvoice(c *fiber.Ctx) error {
	irn := c.Params("irn")
	userDetails, err := middleware.GetUserDetails(c)
	if err != nil {
		return apperror.New(fiber.StatusBadRequest, "error", "unable to get user claims", nil, nil)
	}
	if irn == "" {
		return apperror.New(fiber.StatusBadRequest, "error", "irn is required", nil, nil)
	}

	db, err := middleware.GetDatabase(c)
	if err != nil {
		return apperror.New(fiber.StatusInternalServerError, "error", err.Error(), err, nil)
	}

	if _, err := h.svc.GetInvoiceByIRN(db, irn, userDetails.ID); err != nil {
		return apperror.New(fiber.StatusNotFound, "error", "invoice not found or does not belong to you", err, nil)
	}

	respData, errDetails, err := h.svc.TransmitInvoice(irn, userDetails.IsSandbox)
	if err != nil {
		return apperror.New(fiber.StatusBadRequest, "error", err.Error(), errDetails, nil)
	}

	h.Logger.Info("successfully")
	rd := utility.BuildSuccessResponse(fiber.StatusOK, "successfully", respData)
	return c.Status(fiber.StatusOK).JSON(rd)
}

// TransmitConfirmInvoice godoc
// @Summary Confirm Transmitted Invoice
// @Description Confirms a transmitted invoice using the IRN.
// @Tags Invoice
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param irn path string true "Invoice Reference Number (IRN)"
// @Success 200 {object} entities.Response "Invoice confirmed successfully"
// @Failure 400 {object} entities.Response "Bad request"
// @Router /invoice/transmit/confirm/{irn} [get]
func (h *Handler) TransmitConfirmInvoice(c *fiber.Ctx) error {
	irn := c.Params("irn")
	userDetails, err := middleware.GetUserDetails(c)
	if err != nil {
		return apperror.New(fiber.StatusBadRequest, "error", "unable to get user claims", nil, nil)
	}
	if irn == "" {
		return apperror.New(fiber.StatusBadRequest, "error", "irn is required", nil, nil)
	}

	db, err := middleware.GetDatabase(c)
	if err != nil {
		return apperror.New(fiber.StatusInternalServerError, "error", err.Error(), err, nil)
	}

	if _, err := h.svc.GetInvoiceByIRN(db, irn, userDetails.ID); err != nil {
		return apperror.New(fiber.StatusNotFound, "error", "invoice not found or does not belong to you", err, nil)
	}

	respData, errDetails, err := h.svc.TransmitConfirmInvoice(irn, userDetails.IsSandbox)
	if err != nil {
		return apperror.New(fiber.StatusBadRequest, "error", err.Error(), errDetails, nil)
	}

	h.Logger.Info("successfully")
	rd := utility.BuildSuccessResponse(fiber.StatusOK, "successfully", respData)
	return c.Status(fiber.StatusOK).JSON(rd)
}

// TransmitPull godoc
// @Summary Pull Transmitted Invoices
// @Description Pulls invoices from FIRS using query params.
// @Tags Invoice
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param query query entities.PullDataQuery true "Query Parameters"
// @Success 200 {object} entities.Response "Invoices pulled successfully"
// @Failure 400 {object} entities.Response "Invalid query parameters"
// @Router /invoice/transmit/pull [get]
func (h *Handler) TransmitPull(c *fiber.Ctx) error {
	var query entities.PullDataQuery
	if err := c.QueryParser(&query); err != nil {
		return apperror.New(fiber.StatusBadRequest, "error", "Invalid query parameters", err, nil)
	}

	respData, errDetails, err := h.svc.TransmitPull(query)
	if err != nil {
		return apperror.New(fiber.StatusBadRequest, "error", err.Error(), errDetails, nil)
	}

	h.Logger.Info("gotten successfully")
	rd := utility.BuildSuccessResponse(fiber.StatusOK, "gotten successfully", respData)
	return c.Status(fiber.StatusOK).JSON(rd)
}

// DebugHealthCheck godoc
// @Summary Debug Health Check
// @Description Performs a debug health check on invoice transmission service.
// @Tags Invoice
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} entities.Response "Health check successful"
// @Failure 400 {object} entities.Response "Bad request"
// @Router /invoice/transmit/health-check [get]
func (h *Handler) DebugHealthCheck(c *fiber.Ctx) error {
	respData, errDetails, err := h.svc.DebugHealthCheck()
	if err != nil {
		return apperror.New(fiber.StatusBadRequest, "error", err.Error(), errDetails, nil)
	}

	h.Logger.Info("successfully")
	rd := utility.BuildSuccessResponse(fiber.StatusOK, "successfully", respData)
	return c.Status(fiber.StatusOK).JSON(rd)
}
