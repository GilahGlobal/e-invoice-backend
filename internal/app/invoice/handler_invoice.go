package invoice

import (
	"einvoice-access-point/internal/apperror"
	"einvoice-access-point/internal/data/entities"
	"einvoice-access-point/internal/middleware"
	"einvoice-access-point/internal/pkg/cloudinary"
	"einvoice-access-point/internal/pkg/firs_models"
	"einvoice-access-point/internal/utility"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
)

// GetAllInvoices godoc
// @Summary Get all invoices
// @Description Returns a list of invoices with minimal details for a business
// @Tags Internal Invoice
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param query query entities.PaginationQuery false "Pagination (page, size)"
// @Success 200 {object} GetAllInvoicesResponseDto "invoices fetched successfully"
// @Failure 400 {object} entities.Response
// @Router /invoice [get]
func (h *Handler) GetAllInvoices(c *fiber.Ctx) error {
	userDetails, err := middleware.GetUserDetails(c)
	if err != nil {
		return apperror.New(fiber.StatusUnauthorized, "error", "Unauthorized", err, nil)
	}

	db, err := middleware.GetDatabase(c)
	if err != nil {
		return apperror.New(fiber.StatusInternalServerError, "error", err.Error(), err, nil)
	}

	var query entities.PaginationQuery
	if err := c.QueryParser(&query); err != nil {
		return apperror.New(fiber.StatusBadRequest, "error", "Invalid query parameters", err, nil)
	}
	if query.Size <= 0 {
		query.Size = 20
	}
	if query.Page <= 0 {
		query.Page = 1
	}

	invoices, pagination, err := h.svc.GetAllInvoicesByBusinessID(db, userDetails.ID, query.Page, query.Size)
	if err != nil {
		return apperror.New(fiber.StatusBadRequest, "error", err.Error(), err, nil)
	}

	rd := utility.BuildSuccessResponse(fiber.StatusOK, "Invoices fetched successfully", invoices, pagination)
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
// @Success 200 {object} GetInvoiceDetailsResponseDto "invoice details fetched successfully"
// @Failure 400 {object} entities.Response
// @Router /invoice/{invoice_id} [get]
func (h *Handler) GetInvoiceDetails(c *fiber.Ctx) error {
	invoiceID := c.Params("invoice_id")

	userDetails, err := middleware.GetUserDetails(c)
	if err != nil {
		return apperror.New(fiber.StatusUnauthorized, "error", "Unauthorized", err, nil)
	}

	db, err := middleware.GetDatabase(c)
	if err != nil {
		return apperror.New(fiber.StatusInternalServerError, "error", err.Error(), err, nil)
	}

	if invoiceID == "" {
		return apperror.New(fiber.StatusBadRequest, "error", "invoice_id is required", nil, nil)
	}

	invoice, err := h.svc.GetInvoiceDetails(db, userDetails.ID, invoiceID)
	if err != nil {
		return apperror.New(fiber.StatusBadRequest, "error", err.Error(), err, nil)
	}

	rd := utility.BuildSuccessResponse(fiber.StatusOK, "Invoice details fetched successfully", invoice)
	return c.Status(fiber.StatusOK).JSON(rd)
}

// DeleteInvoice godoc
// @Summary Delete Invoice
// @Description Deletes an invoice invoice_id
// @Tags Internal Invoice
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param invoice_id path string true "Invoice ID" format(uuid)
// @Success 200 {object} entities.Response "Invoice deleted successfully"
// @Failure 400 {object} entities.Response
// @Router /invoice/{invoice_id} [delete]
func (h *Handler) DeleteInvoice(c *fiber.Ctx) error {
	invoiceID := c.Params("invoice_id")

	userDetails, err := middleware.GetUserDetails(c)
	if err != nil {
		return apperror.New(fiber.StatusUnauthorized, "error", "Unauthorized", err, nil)
	}

	db, err := middleware.GetDatabase(c)
	if err != nil {
		return apperror.New(fiber.StatusInternalServerError, "error", err.Error(), err, nil)
	}

	if invoiceID == "" {
		return apperror.New(fiber.StatusBadRequest, "error", "invoice_id is required", nil, nil)
	}

	if err := h.svc.DeleteInvoice(db, userDetails.ID, invoiceID); err != nil {
		return apperror.New(fiber.StatusBadRequest, "error", err.Error(), err, nil)
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
// @Param   payload  body  firs_models.UploadInvoiceRequestDto  true  "Invoice Payload"
// @Success 201 {object} UploadInvoiceResponseDto "Invoice created successfully"
// @Failure 400 {object} entities.Response "Bad request"
// @Router /invoice/upload [post]
func (h *Handler) UploadInvoice(c *fiber.Ctx) error {
	client := c.Get("client")
	userDetails, err := middleware.GetUserDetails(c)
	if err != nil {
		return apperror.New(fiber.StatusBadRequest, "error", "unable to get user claims", nil, nil)
	}

	db, err := middleware.GetDatabase(c)
	if err != nil {
		return apperror.New(fiber.StatusInternalServerError, "error", err.Error(), err, nil)
	}
	setup, err := h.businessSvc.ValidateInvoiceUploadSetup(db, userDetails.ID)
	if err != nil {
		return apperror.New(fiber.StatusBadRequest, "error", err.Error(), nil, nil)
	}

	var req firs_models.UploadInvoiceRequestDto
	err = c.BodyParser(&req)
	if err != nil {
		return apperror.New(fiber.StatusBadRequest, "error", "Failed to parse request body", err, nil)
	}

	err = h.Validator.Struct(&req)
	if err != nil {
		rd := utility.BuildErrorResponse(
			fiber.StatusUnprocessableEntity,
			"error", "Validation failed",
			utility.ValidationErrorsToJSON(err, firs_models.UploadInvoiceRequestDto{}),
			nil,
		)
		return c.Status(fiber.StatusUnprocessableEntity).JSON(rd)
	}

	invoiceExists, err := h.svc.GetInvoiceByInvoiceNumber(db, req.InvoiceNumber, userDetails.ID)
	if err != nil {
		return apperror.New(fiber.StatusBadRequest, "error", err.Error(), err, nil)
	}

	if invoiceExists != nil {
		blockedStatuses := map[string]bool{
			entities.StatusSignedInvoice: true,
			entities.StatusTransmitted:   true,
			entities.StatusConfirmed:     true,
		}
		if blockedStatuses[invoiceExists.CurrentStatus] {
			response := map[string]interface{}{
				"metadata": invoiceExists.StatusHistory,
			}

			dataMap := map[string]interface{}{
				"id":             invoiceExists.ID,
				"invoice_number": invoiceExists.InvoiceNumber,
				"irn":            invoiceExists.IRN,
				"qr_code":        invoiceExists.QrCode,
				"qr_code_2":      invoiceExists.EncryptedIRN,
			}
			if invoiceExists.QrCodeBmpUrl != "" {
				dataMap["qr_code_bmp_url"] = invoiceExists.QrCodeBmpUrl
			}
			response["data"] = dataMap

			rd := utility.BuildSuccessResponse(fiber.StatusOK, "Invoice previously uploaded successfully", response)
			return c.Status(fiber.StatusOK).JSON(rd)
		}
	}

	var irnPayload InvoiceData
	if req.IRN == nil || *req.IRN == "" {
		irnData, err := h.svc.IRNGeneration(db, userDetails.ID, req.InvoiceNumber, setup.ServiceID, req.BusinessID, userDetails.IsSandbox)
		if err != nil {
			rd := *err
			return c.Status(fiber.StatusBadRequest).JSON(rd)
		}
		irnPayload = *irnData
		req.IRN = &irnPayload.IRN
	} else {
		if invoiceExists == nil {
			irnPayload = InvoiceData{
				InvoiceNumber: req.InvoiceNumber,
				IRN:           *req.IRN,
				QRCode:        "",
				QRCode2:       "",
				QRCodeBMP:     "",
			}
		} else {
			// qrBmp, _ := utility.Base64PNGToBMP(invoiceExists.QrCode)
			irnPayload = InvoiceData{
				InvoiceNumber: req.InvoiceNumber,
				IRN:           *req.IRN,
				QRCode:        invoiceExists.QrCode,
				QRCode2:       invoiceExists.EncryptedIRN,
				QRCodeBMP:     "",
			}
		}
	}

	qrCodeBMPURL := ""
	if setup.BmpUploadSelected && irnPayload.QRCodeBMP != "" {
		qrCodeBMPURL, err = cloudinary.UploadBMPBase64(irnPayload.QRCodeBMP, utility.GenerateUUID())
		if err != nil {
			qrCodeBMPURL = ""
		}
	}

	createdInvoice, _, err, isInvoiceSigned := h.svc.CreateInvoice(db, req, req.InvoiceNumber, userDetails.ID, irnPayload.QRCode, qrCodeBMPURL, irnPayload.QRCode2, invoiceExists, userDetails.IsSandbox, nil, client)

	response := map[string]interface{}{
		"metadata": createdInvoice.StatusHistory,
	}
	if isInvoiceSigned {
		dataMap := map[string]interface{}{
			"id":             createdInvoice.ID,
			"invoice_number": irnPayload.InvoiceNumber,
			"irn":            irnPayload.IRN,
			"qr_code":        irnPayload.QRCode,
			"qr_code_2":      irnPayload.QRCode2,
		}
		if qrCodeBMPURL != "" {
			dataMap["qr_code_bmp_url"] = qrCodeBMPURL
		}
		response["data"] = dataMap
	}

	if err != nil {
		if isInvoiceSigned {
			return apperror.New(fiber.StatusCreated, "partial_success", err.Error(), response, nil)
		}
		return apperror.New(fiber.StatusBadRequest, "error", err.Error(), response, nil)
	}

	rd := utility.BuildSuccessResponse(fiber.StatusCreated, "Invoice created successfully", response)
	return c.Status(fiber.StatusCreated).JSON(rd)
}

// ModifyInvoice godoc
// @Summary Modify an existing invoice
// @Description Re-uploads an invoice with the same invoice_number. Deprecates the old invoice on NRS (REJECTED), generates a fresh IRN.
// @Tags Internal Invoice
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param   payload  body  firs_models.UploadInvoiceRequestDto  true  "Invoice Payload"
// @Success 200 {object} UploadInvoiceResponseDto "Invoice modified successfully"
// @Failure 400 {object} entities.Response "Bad request"
// @Failure 422 {object} entities.Response "Validation failed"
// @Router /invoice/upload [patch]
func (h *Handler) ModifyInvoice(c *fiber.Ctx) error {
	client := c.Get("client")
	userDetails, err := middleware.GetUserDetails(c)
	if err != nil {
		return apperror.New(fiber.StatusBadRequest, "error", "unable to get user claims", nil, nil)
	}

	db, err := middleware.GetDatabase(c)
	if err != nil {
		return apperror.New(fiber.StatusInternalServerError, "error", err.Error(), err, nil)
	}
	setup, err := h.businessSvc.ValidateInvoiceUploadSetup(db, userDetails.ID)
	if err != nil {
		return apperror.New(fiber.StatusBadRequest, "error", err.Error(), nil, nil)
	}

	var req firs_models.UploadInvoiceRequestDto
	if err := c.BodyParser(&req); err != nil {
		return apperror.New(fiber.StatusBadRequest, "error", "Failed to parse request body", err, nil)
	}

	if err := h.Validator.Struct(&req); err != nil {
		rd := utility.BuildErrorResponse(
			fiber.StatusUnprocessableEntity,
			"error", "Validation failed",
			utility.ValidationErrorsToJSON(err, firs_models.UploadInvoiceRequestDto{}),
			nil,
		)
		return c.Status(fiber.StatusUnprocessableEntity).JSON(rd)
	}

	existingInvoice, err := h.svc.GetInvoiceByInvoiceNumber(db, req.InvoiceNumber, userDetails.ID)
	if err != nil {
		return apperror.New(fiber.StatusBadRequest, "error", err.Error(), err, nil)
	}
	if existingInvoice == nil {
		return apperror.New(fiber.StatusNotFound, "error", "invoice not found with the given invoice number", nil, nil)
	}

	oldIRN := existingInvoice.IRN

	blockedStatuses := map[string]bool{
		entities.StatusSignedInvoice: true,
		entities.StatusTransmitted:   true,
		entities.StatusConfirmed:     true,
	}

	if blockedStatuses[existingInvoice.CurrentStatus] {
		now := time.Now().UTC()
		created := existingInvoice.CreatedAt.UTC()
		if now.Year() == created.Year() && now.YearDay() == created.YearDay() {
			return apperror.New(fiber.StatusFailedDependency, "error", "invoice can only be modified on a different day from its creation", nil, nil)
		}

		if err := h.svc.DeprecateInvoiceOnNRS(oldIRN, userDetails.IsSandbox); err != nil {
			return apperror.New(fiber.StatusBadRequest, "error", "failed to deprecate old invoice on NRS: "+err.Error(), nil, nil)
		}
	}

	irnData, irnErr := h.svc.IRNGeneration(db, userDetails.ID, req.InvoiceNumber, setup.ServiceID, req.BusinessID, userDetails.IsSandbox)
	if irnErr != nil {
		rd := *irnErr
		return c.Status(fiber.StatusBadRequest).JSON(rd)
	}
	req.IRN = &irnData.IRN

	qrCodeBMPURL := ""
	if setup.BmpUploadSelected && irnData.QRCodeBMP != "" {
		qrCodeBMPURL, err = cloudinary.UploadBMPBase64(irnData.QRCodeBMP, utility.GenerateUUID())
		if err != nil {
			qrCodeBMPURL = ""
		}
	}

	replacedInvoice, err := h.svc.ReplaceInvoiceRecord(db, existingInvoice, req, *req.IRN, irnData.QRCode, qrCodeBMPURL, irnData.QRCode2, client)
	if err != nil {
		return apperror.New(fiber.StatusInternalServerError, "error", err.Error(), nil, nil)
	}

	firsErr, isInvoiceSigned := h.svc.FirsAllInOneProcess(req, replacedInvoice, db, userDetails.IsSandbox)

	response := map[string]interface{}{
		"metadata": replacedInvoice.StatusHistory,
	}
	if isInvoiceSigned {
		dataMap := map[string]interface{}{
			"id":             replacedInvoice.ID,
			"invoice_number": irnData.InvoiceNumber,
			"irn":            irnData.IRN,
			"qr_code":        irnData.QRCode2,
		}
		if qrCodeBMPURL != "" {
			dataMap["qr_code_bmp_url"] = qrCodeBMPURL
		}
		response["data"] = dataMap
	}

	if firsErr != nil {
		errorArray := strings.Split(firsErr.Error(), "-")
		if isInvoiceSigned {
			return apperror.New(fiber.StatusCreated, "partial_success", errorArray[len(errorArray)-1], response, nil)
		}
		return apperror.New(fiber.StatusBadRequest, "error", errorArray[len(errorArray)-1], response, nil)
	}

	rd := utility.BuildSuccessResponse(fiber.StatusOK, "Invoice modified successfully", response)
	return c.Status(fiber.StatusOK).JSON(rd)
}

// GetInvoiceStats godoc
// @Summary Get invoice statistics
// @Description Returns statistics for invoices including total, partial, successful, and failed.
// @Tags Invoice
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} GetInvoiceStatsResponseDto "Invoice statistics fetched successfully"
// @Failure 401 {object} entities.Response "Unauthorized"
// @Router /invoice/stats [get]
func (h *Handler) GetInvoiceStats(c *fiber.Ctx) error {
	userDetails, err := middleware.GetUserDetails(c)
	if err != nil {
		return apperror.New(fiber.StatusUnauthorized, "error", "Unauthorized", err, nil)
	}

	db, err := middleware.GetDatabase(c)
	if err != nil {
		return apperror.New(fiber.StatusInternalServerError, "error", err.Error(), err, nil)
	}

	var businessID, aggregatorID *string
	if userDetails.IsAggregator {
		aggregatorID = &userDetails.ID
	} else {
		businessID = &userDetails.ID
	}

	stats, err := h.svc.GetInvoiceStats(db, businessID, aggregatorID)
	if err != nil {
		return apperror.New(fiber.StatusInternalServerError, "error", "failed to retrieve invoice stats", err, nil)
	}

	rd := utility.BuildSuccessResponse(fiber.StatusOK, "Invoice statistics fetched successfully", stats)
	return c.Status(fiber.StatusOK).JSON(rd)
}
