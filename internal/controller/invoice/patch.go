package invoice

import (
	"einvoice-access-point/external/firs_models"
	"einvoice-access-point/internal/services/invoice"
	"einvoice-access-point/pkg/middleware"
	"einvoice-access-point/pkg/utility"

	"github.com/gofiber/fiber/v2"
)

// UpdateInvoice godoc
// @Summary Update Invoice
// @Description Updates an existing invoice using the IRN.
// @Tags Invoice
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param irn path string true "Invoice Reference Number (IRN)"
// @Param request body firs_models.UpdateInvoice true "Update Invoice Request"
// @Success 200 {object} dtos.BaseResponseDto "Invoice updated successfully"
// @Failure 400 {object} models.Response "Bad request"
// @Failure 422 {object} models.Response "Validation failed"
// @Router /invoice/update/{irn} [patch]
func (base *Controller) UpdateInvoice(c *fiber.Ctx) error {
	userDetails, err := middleware.GetUserDetails(c)
	if err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusUnauthorized, "error", "Unauthorized", err, nil)
		return c.Status(fiber.StatusUnauthorized).JSON(rd)
	}

	db := middleware.GetDatabaseInstance(userDetails.IsSandbox, base.Db, base.TestDB)

	irn := c.Params("irn")
	if irn == "" {
		rd := utility.BuildErrorResponse(fiber.StatusBadRequest, "error", "irn is required", nil, nil)
		return c.Status(fiber.StatusBadRequest).JSON(rd)
	}

	var req firs_models.UpdateInvoice

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

	respData, errDetails, err := invoice.UpdateInvoice(req, irn, userDetails.IsSandbox)
	if err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusBadRequest, "error", err.Error(), errDetails, nil)
		return c.Status(fiber.StatusBadRequest).JSON(rd)
	}

	if err := invoice.UpdateStoredInvoicePaymentStatus(db, userDetails.ID, irn, req.PaymentStatus); err != nil {
		rd := utility.BuildErrorResponse(
			fiber.StatusInternalServerError,
			"error",
			"invoice updated on FIRS but failed to update local invoice record",
			err.Error(),
			nil,
		)
		return c.Status(fiber.StatusInternalServerError).JSON(rd)
	}

	base.Logger.Info("Invoice updated successfully")
	rd := utility.BuildSuccessResponse(fiber.StatusOK, "Invoice updated successfully", respData)
	return c.Status(fiber.StatusOK).JSON(rd)
}
