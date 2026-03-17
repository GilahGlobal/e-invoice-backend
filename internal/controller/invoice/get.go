package invoice

import (
	"context"
	"einvoice-access-point/external/firs_models"
	"einvoice-access-point/internal/dtos"
	"einvoice-access-point/internal/services/invoice"
	"einvoice-access-point/pkg/middleware"
	"einvoice-access-point/pkg/models"
	"einvoice-access-point/pkg/s3"
	"einvoice-access-point/pkg/utility"
	"einvoice-access-point/pkg/workers"
	"einvoice-access-point/pkg/workers/producer"
	"log"
	"strings"

	"github.com/gofiber/fiber/v2"
)

// ConfirmInvoice godoc
// @Summary Confirm Invoice
// @Description Confirms an invoice with IRN.
// @Tags Invoice
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param irn path string true "Invoice Reference Number (IRN)"
// @Success 200 {object} models.Response "Invoice confirmed successfully"
// @Failure 400 {object} models.Response "Bad request"
// @Router /invoice/confirm/{irn} [get]
func (base *Controller) ConfirmInvoice(c *fiber.Ctx) error {
	userDetails, err := middleware.GetUserDetails(c)
	if err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusBadRequest, "error", "unable to get user claims", nil, nil)
		return c.Status(fiber.StatusBadRequest).JSON(rd)
	}

	irn := c.Params("irn")
	if irn == "" {
		rd := utility.BuildErrorResponse(fiber.StatusBadRequest, "error", "irn is required", nil, nil)
		return c.Status(fiber.StatusBadRequest).JSON(rd)
	}

	respData, errDetails, err := invoice.ConfirmInvoice(irn, userDetails.IsSandbox)
	if err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusBadRequest, "error", err.Error(), errDetails, nil)
		return c.Status(fiber.StatusBadRequest).JSON(rd)
	}

	base.Logger.Info("Invoice confirmed with irn successfully")
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
// @Success 200 {object} models.Response "Invoice downloaded successfully"
// @Failure 400 {object} models.Response "Bad request"
// @Router /invoice/download/{irn} [get]
func (base *Controller) DownloadInvoice(c *fiber.Ctx) error {
	userDetails, err := middleware.GetUserDetails(c)
	if err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusBadRequest, "error", "unable to get user claims", nil, nil)
		return c.Status(fiber.StatusBadRequest).JSON(rd)
	}

	irn := c.Params("irn")
	if irn == "" {
		rd := utility.BuildErrorResponse(fiber.StatusBadRequest, "error", "irn is required", nil, nil)
		return c.Status(fiber.StatusBadRequest).JSON(rd)
	}

	respData, errDetails, err := invoice.DownloadInvoice(irn, userDetails.IsSandbox)
	if err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusBadRequest, "error", err.Error(), errDetails, nil)
		return c.Status(fiber.StatusBadRequest).JSON(rd)
	}

	base.Logger.Info("Invoice downloaded with irn successfully")
	rd := utility.BuildSuccessResponse(fiber.StatusOK, "Invoice downloaded with irn successfully", respData)
	return c.Status(fiber.StatusOK).JSON(rd)
}

// GetAllInvoices godoc
// @Summary Get all invoices
// @Description Returns a list of invoices with minimal details for a business
// @Tags Internal Invoice
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} dtos.GetAllInvoicesResponseDto "invoices fetched successfully"
// @Failure 400 {object} models.Response
// @Router /invoice [get]
func (base *Controller) GetAllInvoices(c *fiber.Ctx) error {
	userDetails, err := middleware.GetUserDetails(c)
	if err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusUnauthorized, "error", "Unauthorized", err, nil)
		return c.Status(fiber.StatusUnauthorized).JSON(rd)
	}

	db := middleware.GetDatabaseInstance(userDetails.IsSandbox, base.Db, base.TestDB)

	invoices, err := invoice.GetAllInvoicesByBusinessID(db, userDetails.ID)
	if err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusBadRequest, "error", err.Error(), err, nil)
		return c.Status(fiber.StatusBadRequest).JSON(rd)
	}

	rd := utility.BuildSuccessResponse(fiber.StatusOK, "Invoices fetched successfully", invoices)
	return c.Status(fiber.StatusOK).JSON(rd)
}

// GetInvoiceDetails godoc
// @Summary Get one invoice details
// @Description Returns full invoice details by invoice ID
// @Tags Internal Invoice
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param invoice_id path string true "Invoice ID" format(uuid)
// @Success 200 {object} dtos.GetInvoiceDetailsResponseDto "invoice details fetched successfully"
// @Failure 400 {object} models.Response
// @Router /invoice/{invoice_id} [get]
func (base *Controller) GetInvoiceDetails(c *fiber.Ctx) error {
	invoiceID := c.Params("invoice_id")

	userDetails, err := middleware.GetUserDetails(c)
	if err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusUnauthorized, "error", "Unauthorized", err, nil)
		return c.Status(fiber.StatusUnauthorized).JSON(rd)
	}

	db := middleware.GetDatabaseInstance(userDetails.IsSandbox, base.Db, base.TestDB)

	if invoiceID == "" {
		rd := utility.BuildErrorResponse(fiber.StatusBadRequest, "error", "invoice_id is required", nil, nil)
		return c.Status(fiber.StatusBadRequest).JSON(rd)
	}

	invoice, err := invoice.GetInvoiceDetails(db, userDetails.ID, invoiceID)
	if err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusBadRequest, "error", err.Error(), err, nil)
		return c.Status(fiber.StatusBadRequest).JSON(rd)
	}

	rd := utility.BuildSuccessResponse(fiber.StatusOK, "Invoice details fetched successfully", invoice)
	return c.Status(fiber.StatusOK).JSON(rd)
}

// CreateInvoice godoc
// @Summary Create a new Invoice
// @Description Upload a JSON invoice file and store it in DB
// @Tags Internal Invoice
// @Accept multipart/form-data
// @Produce json
// @Security BearerAuth
// @Param file formData file true "Invoice JSON File"
// @Success 200 {object} models.Response "Invoice created successfully"
// @Failure 400 {object} models.Response "Bad request"
// @Router /invoice/create [post]
func (base *Controller) CreateInvoice(c *fiber.Ctx) error {

	userDetails, err := middleware.GetUserDetails(c)
	if err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusUnauthorized, "error", "Unauthorized", err, nil)
		return c.Status(fiber.StatusUnauthorized).JSON(rd)
	}

	db := middleware.GetDatabaseInstance(userDetails.IsSandbox, base.Db, base.TestDB)

	file, err := c.FormFile("file")
	if err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusBadRequest, "error", "invoice JSON file is required", nil, nil)
		return c.Status(fiber.StatusBadRequest).JSON(rd)
	}

	fileContent, err := file.Open()
	if err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusBadRequest, "error", "failed to read file", nil, nil)
		return c.Status(fiber.StatusBadRequest).JSON(rd)
	}
	defer fileContent.Close()

	ctx := context.Background()
	fileURL, fileKey, err := s3.UploadFileToS3(ctx, fileContent, file)
	if err != nil {
		log.Println("S3 upload failed:", err)
		return c.Status(500).JSON(fiber.Map{"error": "upload failed"})
	}

	err = invoice.AddBulkUploadLog(db, fileURL, fileKey)
	if err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusInternalServerError, "error", "failed to log bulk upload", nil, nil)
		return c.Status(fiber.StatusInternalServerError).JSON(rd)
	}

	err = producer.NewProducer().EnqueueTask(workers.BulkUploadTask, workers.BulkUploadInput{
		ID:         userDetails.ID,
		FileKey:    fileKey,
		ServiceID:  userDetails.ServiceID,
		BusinessID: *userDetails.BusinessID,
		IsSandbox:  userDetails.IsSandbox,
	})
	if err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusInternalServerError, "error", "failed to enqueue bulk upload task", nil, nil)
		return c.Status(fiber.StatusInternalServerError).JSON(rd)
	}

	rd := utility.BuildSuccessResponse(fiber.StatusCreated, "Invoice uploaded successfully", fileURL)
	return c.Status(fiber.StatusCreated).JSON(rd)
}

// DeleteInvoice godoc
// @Summary Delete Invoice
// @Description Deletes an invoice invoice_id
// @Tags Internal Invoice
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param invoice_id path string true "Invoice ID" format(uuid)
// @Success 200 {object} dtos.BaseResponseDto "Invoice deleted successfully"
// @Failure 400 {object} models.Response
// @Router /invoice/{invoice_id} [delete]
func (base *Controller) DeleteInvoice(c *fiber.Ctx) error {
	invoiceID := c.Params("invoice_id")

	userDetails, err := middleware.GetUserDetails(c)
	if err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusUnauthorized, "error", "Unauthorized", err, nil)
		return c.Status(fiber.StatusUnauthorized).JSON(rd)
	}

	db := middleware.GetDatabaseInstance(userDetails.IsSandbox, base.Db, base.TestDB)

	if invoiceID == "" {
		rd := utility.BuildErrorResponse(fiber.StatusBadRequest, "error", "invoice_id is required", nil, nil)
		return c.Status(fiber.StatusBadRequest).JSON(rd)
	}

	if err := invoice.DeleteInvoice(db, userDetails.ID, invoiceID); err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusBadRequest, "error", err.Error(), err, nil)
		return c.Status(fiber.StatusBadRequest).JSON(rd)
	}

	rd := utility.BuildSuccessResponse(fiber.StatusOK, "Invoice deleted successfully", nil)
	return c.Status(fiber.StatusOK).JSON(rd)

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
// @Router /invoice/upload [post]
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
		IRNData, err := invoice.IRNGeneration(req.InvoiceNumber, userDetails.ServiceID, req.BusinessID, userDetails.IsSandbox)
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

	createdInvoice, _, err, isInvoiceSigned := invoice.CreateInvoice(db, req, req.InvoiceNumber, userDetails.ID, irnPayload.QRCode, irnPayload.QRCode2, invoiceExists, userDetails.IsSandbox)

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

	rd := utility.BuildSuccessResponse(fiber.StatusCreated, "Invoice created successfully", response)
	return c.Status(fiber.StatusCreated).JSON(rd)
}
