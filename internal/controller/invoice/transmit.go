package invoice

import (
	"einvoice-access-point/internal/services/invoice"
	"einvoice-access-point/pkg/middleware"
	"einvoice-access-point/pkg/models"
	"einvoice-access-point/pkg/utility"

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
// @Success 200 {object} models.Response "Invoice details retrieved"
// @Failure 400 {object} models.Response "Bad request"
// @Router /invoice/transmit/lookup-irn/{irn} [get]
func (base *Controller) LookUpIRN(c *fiber.Ctx) error {
	irn := c.Params("irn")
	if irn == "" {
		rd := utility.BuildErrorResponse(fiber.StatusBadRequest, "error", "irn is required", nil, nil)
		return c.Status(fiber.StatusBadRequest).JSON(rd)
	}

	respData, errDetails, err := invoice.LookUpIRN(irn)
	if err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusBadRequest, "error", err.Error(), errDetails, nil)
		return c.Status(fiber.StatusBadRequest).JSON(rd)
	}

	base.Logger.Info("successfully")
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
// @Success 200 {object} models.Response "TIN details retrieved"
// @Failure 400 {object} models.Response "Bad request"
// @Router /invoice/transmit/lookup-tin/{tin} [get]
func (base *Controller) LookUpTIN(c *fiber.Ctx) error {
	tin := c.Params("tin")
	if tin == "" {
		rd := utility.BuildErrorResponse(fiber.StatusBadRequest, "error", "tin is required", nil, nil)
		return c.Status(fiber.StatusBadRequest).JSON(rd)
	}

	respData, errDetails, err := invoice.LookUpTIN(tin)
	if err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusBadRequest, "error", err.Error(), errDetails, nil)
		return c.Status(fiber.StatusBadRequest).JSON(rd)
	}

	base.Logger.Info("successfully")
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
// @Success 200 {object} models.Response "Party ID details retrieved"
// @Failure 400 {object} models.Response "Bad request"
// @Router /invoice/transmit/lookup-party/{party_id} [get]
func (base *Controller) LookUpPartyID(c *fiber.Ctx) error {
	partyId := c.Params("party_id")
	if partyId == "" {
		rd := utility.BuildErrorResponse(fiber.StatusBadRequest, "error", "partyID is required", nil, nil)
		return c.Status(fiber.StatusBadRequest).JSON(rd)
	}

	respData, errDetails, err := invoice.LookUpPartyID(partyId)
	if err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusBadRequest, "error", err.Error(), errDetails, nil)
		return c.Status(fiber.StatusBadRequest).JSON(rd)
	}

	base.Logger.Info("successfully")
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
// @Success 200 {object} models.Response "Invoice transmitted successfully"
// @Failure 400 {object} models.Response "Bad request"
// @Router /invoice/transmit/{irn} [post]
func (base *Controller) TransmitInvoice(c *fiber.Ctx) error {
	irn := c.Params("irn")
	userDetails, err := middleware.GetUserDetails(c)
	if irn == "" {
		rd := utility.BuildErrorResponse(fiber.StatusBadRequest, "error", "irn is required", nil, nil)
		return c.Status(fiber.StatusBadRequest).JSON(rd)
	}

	respData, errDetails, err := invoice.TransmitInvoice(irn, userDetails.IsSandbox)
	if err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusBadRequest, "error", err.Error(), errDetails, nil)
		return c.Status(fiber.StatusBadRequest).JSON(rd)
	}

	base.Logger.Info("successfully")
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
// @Success 200 {object} models.Response "Invoice confirmed successfully"
// @Failure 400 {object} models.Response "Bad request"
// @Router /invoice/transmit/confirm/{irn} [get]
func (base *Controller) TransmitConfirmInvoice(c *fiber.Ctx) error {
	irn := c.Params("irn")
	userDetails, err := middleware.GetUserDetails(c)
	if irn == "" {
		rd := utility.BuildErrorResponse(fiber.StatusBadRequest, "error", "irn is required", nil, nil)
		return c.Status(fiber.StatusBadRequest).JSON(rd)
	}

	respData, errDetails, err := invoice.TransmitConfirmInvoice(irn, userDetails.IsSandbox)
	if err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusBadRequest, "error", err.Error(), errDetails, nil)
		return c.Status(fiber.StatusBadRequest).JSON(rd)
	}

	base.Logger.Info("successfully")
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
// @Param query query models.PullDataQuery true "Query Parameters"
// @Success 200 {object} models.Response "Invoices pulled successfully"
// @Failure 400 {object} models.Response "Invalid query parameters"
// @Router /invoice/transmit/pull [get]
func (base *Controller) TransmitPull(c *fiber.Ctx) error {

	var query models.PullDataQuery
	if err := c.QueryParser(&query); err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusBadRequest, "error", "Invalid query parameters", err, nil)
		return c.Status(fiber.StatusBadRequest).JSON(rd)
	}

	respData, errDetails, err := invoice.TransmitPull(query)
	if err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusBadRequest, "error", err.Error(), errDetails, nil)
		return c.Status(fiber.StatusBadRequest).JSON(rd)
	}

	base.Logger.Info("gotten successfully")
	//fmt.Println(respData)
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
// @Success 200 {object} models.Response "Health check successful"
// @Failure 400 {object} models.Response "Bad request"
// @Router /invoice/transmit/health-check [get]
func (base *Controller) DebugHealthCheck(c *fiber.Ctx) error {

	respData, errDetails, err := invoice.DebugHealthCheck()
	if err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusBadRequest, "error", err.Error(), errDetails, nil)
		return c.Status(fiber.StatusBadRequest).JSON(rd)
	}

	base.Logger.Info("successfully")
	rd := utility.BuildSuccessResponse(fiber.StatusOK, "successfully", respData)
	return c.Status(fiber.StatusOK).JSON(rd)
}
