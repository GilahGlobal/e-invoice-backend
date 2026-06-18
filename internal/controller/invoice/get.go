package invoice

import (
	"context"
	"einvoice-access-point/external/firs_models"
	"einvoice-access-point/internal/dtos"
	businessservice "einvoice-access-point/internal/services/business"
	"einvoice-access-point/internal/services/invoice"
	"einvoice-access-point/pkg/middleware"
	"einvoice-access-point/pkg/models"
	"einvoice-access-point/pkg/s3"
	"einvoice-access-point/pkg/utility"
	"einvoice-access-point/pkg/workers"
	"einvoice-access-point/pkg/workers/producer"
	"log"
	"strings"
	"time"

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
// @Param query query models.PaginationQuery false "Pagination (page, size)"
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

	var query models.PaginationQuery
	if err := c.QueryParser(&query); err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusBadRequest, "error", "Invalid query parameters", err, nil)
		return c.Status(fiber.StatusBadRequest).JSON(rd)
	}
	if query.Size <= 0 {
		query.Size = 20
	}
	if query.Page <= 0 {
		query.Page = 1
	}

	invoices, pagination, err := invoice.GetAllInvoicesByBusinessID(db, userDetails.ID, query.Page, query.Size)
	if err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusBadRequest, "error", err.Error(), err, nil)
		return c.Status(fiber.StatusBadRequest).JSON(rd)
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

// GetBulkUploadLogs godoc
// @Summary Get bulk upload logs
// @Description Retrieve paginated bulk upload logs for a business
// @Tags Invoice
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param query query models.PaginationQuery false "Pagination (page, size)"
// @Success 200 {object} dtos.GetBulkUploadLogsResponseDto "Bulk upload logs fetched successfully"
// @Failure 400 {object} models.Response "Bad request"
// @Failure 401 {object} models.Response "Unauthorized"
// @Router /invoice/bulk-upload [get]
func (base *Controller) GetBulkUploadLogs(c *fiber.Ctx) error {
	userDetails, err := middleware.GetUserDetails(c)
	if err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusUnauthorized, "error", "Unauthorized", err, nil)
		return c.Status(fiber.StatusUnauthorized).JSON(rd)
	}

	if userDetails.BusinessID == nil || *userDetails.BusinessID == "" {
		rd := utility.BuildErrorResponse(fiber.StatusBadRequest, "error", "business_id is required", nil, nil)
		return c.Status(fiber.StatusBadRequest).JSON(rd)
	}

	var query models.PaginationQuery
	if err := c.QueryParser(&query); err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusBadRequest, "error", "Invalid query parameters", err, nil)
		return c.Status(fiber.StatusBadRequest).JSON(rd)
	}
	if query.Size <= 0 {
		query.Size = 20
	}
	if query.Page <= 0 {
		query.Page = 1
	}

	db := middleware.GetDatabaseInstance(userDetails.IsSandbox, base.Db, base.TestDB)
	logs, pagination, err := invoice.GetBulkUploadLogsByBusinessID(db, *userDetails.BusinessID, query.Page, query.Size)
	if err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusBadRequest, "error", err.Error(), err, nil)
		return c.Status(fiber.StatusBadRequest).JSON(rd)
	}

	rd := utility.BuildSuccessResponse(fiber.StatusOK, "Bulk upload logs fetched successfully", logs, pagination)
	return c.Status(fiber.StatusOK).JSON(rd)
}

// GetBulkUploadFailedInvoices godoc
// @Summary Get failed invoices from a bulk upload
// @Description Retrieve the failed invoices recorded for a specific bulk upload belonging to the authenticated business
// @Tags Invoice
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param bulk_id path string true "Bulk upload ID"
// @Success 200 {object} dtos.GetBulkUploadFailedInvoicesResponseDto "Bulk upload failed invoices fetched successfully"
// @Failure 400 {object} models.Response "Bad request"
// @Failure 401 {object} models.Response "Unauthorized"
// @Failure 404 {object} models.Response "Bulk upload not found"
// @Failure 500 {object} models.Response "Internal server error"
// @Router /invoice/bulk-upload/{bulk_id}/failed [get]
func (base *Controller) GetBulkUploadFailedInvoices(c *fiber.Ctx) error {
	userDetails, err := middleware.GetUserDetails(c)
	if err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusUnauthorized, "error", "Unauthorized", err, nil)
		return c.Status(fiber.StatusUnauthorized).JSON(rd)
	}

	if userDetails.BusinessID == nil || *userDetails.BusinessID == "" {
		rd := utility.BuildErrorResponse(fiber.StatusBadRequest, "error", "business_id is required", nil, nil)
		return c.Status(fiber.StatusBadRequest).JSON(rd)
	}

	bulkUploadID := c.Params("bulk_id")
	if bulkUploadID == "" {
		rd := utility.BuildErrorResponse(fiber.StatusBadRequest, "error", "bulk upload id is required", nil, nil)
		return c.Status(fiber.StatusBadRequest).JSON(rd)
	}

	db := middleware.GetDatabaseInstance(userDetails.IsSandbox, base.Db, base.TestDB)
	failedInvoices, err := invoice.GetBulkUploadFailedInvoices(db, bulkUploadID, *userDetails.BusinessID)
	if err != nil {
		status := fiber.StatusInternalServerError
		if strings.Contains(strings.ToLower(err.Error()), "not found") {
			status = fiber.StatusNotFound
		}

		rd := utility.BuildErrorResponse(status, "error", err.Error(), err, nil)
		return c.Status(status).JSON(rd)
	}

	rd := utility.BuildSuccessResponse(fiber.StatusOK, "Bulk upload failed invoices fetched successfully", failedInvoices)
	return c.Status(fiber.StatusOK).JSON(rd)
}

// DownloadBulkUploadFailedInvoices godoc
// @Summary Download failed invoices from a bulk upload
// @Description Download the failed invoices recorded for a specific bulk upload belonging to the authenticated business as csv or excel
// @Tags Invoice
// @Produce text/csv,application/vnd.openxmlformats-officedocument.spreadsheetml.sheet
// @Security BearerAuth
// @Param bulk_id path string true "Bulk upload ID"
// @Param format query string false "Export format" Enums(csv,excel,xlsx)
// @Success 200 {file} file "Bulk upload failed invoices file"
// @Failure 400 {object} models.Response "Bad request"
// @Failure 401 {object} models.Response "Unauthorized"
// @Failure 404 {object} models.Response "Bulk upload not found"
// @Failure 500 {object} models.Response "Internal server error"
// @Router /invoice/bulk-upload/{bulk_id}/failed/download [get]
func (base *Controller) DownloadBulkUploadFailedInvoices(c *fiber.Ctx) error {
	userDetails, err := middleware.GetUserDetails(c)
	if err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusUnauthorized, "error", "Unauthorized", err, nil)
		return c.Status(fiber.StatusUnauthorized).JSON(rd)
	}

	if userDetails.BusinessID == nil || *userDetails.BusinessID == "" {
		rd := utility.BuildErrorResponse(fiber.StatusBadRequest, "error", "business_id is required", nil, nil)
		return c.Status(fiber.StatusBadRequest).JSON(rd)
	}

	bulkUploadID := c.Params("bulk_id")
	if bulkUploadID == "" {
		rd := utility.BuildErrorResponse(fiber.StatusBadRequest, "error", "bulk upload id is required", nil, nil)
		return c.Status(fiber.StatusBadRequest).JSON(rd)
	}

	db := middleware.GetDatabaseInstance(userDetails.IsSandbox, base.Db, base.TestDB)
	failedInvoices, err := invoice.GetBulkUploadFailedInvoices(db, bulkUploadID, *userDetails.BusinessID)
	if err != nil {
		status := fiber.StatusInternalServerError
		if strings.Contains(strings.ToLower(err.Error()), "not found") {
			status = fiber.StatusNotFound
		}

		rd := utility.BuildErrorResponse(status, "error", err.Error(), err, nil)
		return c.Status(status).JSON(rd)
	}

	fileData, contentType, extension, err := invoice.ExportBulkUploadFailedInvoices(failedInvoices, c.Query("format", "csv"))
	if err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusBadRequest, "error", err.Error(), err, nil)
		return c.Status(fiber.StatusBadRequest).JSON(rd)
	}

	c.Set("Content-Type", contentType)
	c.Set("Content-Disposition", "attachment; filename=\"bulk_upload_failed_invoices_"+bulkUploadID+"."+extension+"\"")
	return c.Status(fiber.StatusOK).Send(fileData)
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

	setup, err := businessservice.ValidateInvoiceUploadSetup(db, userDetails.ID)
	if err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusBadRequest, "error", err.Error(), nil, nil)
		return c.Status(fiber.StatusBadRequest).JSON(rd)
	}

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

	bulkID, err := invoice.AddBulkUploadLog(db, fileURL, fileKey, setup.BusinessID, nil)
	if err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusInternalServerError, "error", "failed to log bulk upload", nil, nil)
		return c.Status(fiber.StatusInternalServerError).JSON(rd)
	}

	err = producer.NewProducer().EnqueueTask(workers.BulkUploadTask, workers.BulkUploadInput{
		BulkID:     bulkID,
		ID:         userDetails.ID,
		FileKey:    fileKey,
		ServiceID:  setup.ServiceID,
		BusinessID: setup.BusinessID,
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
// @Router /invoice/upload [post]
func (base *Controller) UploadInvoice(c *fiber.Ctx) error {

	client := c.Get("client")
	userDetails, err := middleware.GetUserDetails(c)
	if err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusBadRequest, "error", "unable to get user claims", nil, nil)
		return c.Status(fiber.StatusBadRequest).JSON(rd)
	}

	db := middleware.GetDatabaseInstance(userDetails.IsSandbox, base.Db, base.TestDB)
	setup, err := businessservice.ValidateInvoiceUploadSetup(db, userDetails.ID)
	if err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusBadRequest, "error", err.Error(), nil, nil)
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
		IRNData, err := invoice.IRNGeneration(db, userDetails.ID, req.InvoiceNumber, setup.ServiceID, req.BusinessID, userDetails.IsSandbox)
		if err != nil {
			rd := *err
			return c.Status(fiber.StatusBadRequest).JSON(rd)
		}
		irnPayload = *IRNData
		req.IRN = &irnPayload.IRN
	} else {
		if invoiceExists == nil {
			irnPayload = dtos.InvoiceData{
				InvoiceNumber: req.InvoiceNumber,
				IRN:           *req.IRN,
				QRCode:        "",
				QRCode2:       "",
			}
		} else {
			irnPayload = dtos.InvoiceData{
				InvoiceNumber: req.InvoiceNumber,
				IRN:           *req.IRN,
				QRCode:        invoiceExists.QrCode,
				QRCode2:       invoiceExists.EncryptedIRN,
			}
		}
	}
	log.Println("nkfknfdjn")
	createdInvoice, _, err, isInvoiceSigned := invoice.CreateInvoice(db, req, req.InvoiceNumber, userDetails.ID, irnPayload.QRCode, irnPayload.QRCode2, invoiceExists, userDetails.IsSandbox, nil, client)

	response := map[string]interface{}{
		"metadata": createdInvoice.StatusHistory,
	}
	if isInvoiceSigned {
		response["data"] = map[string]interface{}{
			"id":             createdInvoice.ID,
			"invoice_number": irnPayload.InvoiceNumber,
			"irn":            irnPayload.IRN,
			"qr_code":        irnPayload.QRCode,
			"qr_code_2":      irnPayload.QRCode2,
		}
	}

	if err != nil {
		errorArray := strings.Split(err.Error(), "-")
		rd := utility.BuildErrorResponse(fiber.StatusBadRequest, "error", errorArray[len(errorArray)-1], response, nil)
		return c.Status(fiber.StatusBadRequest).JSON(rd)
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
// @Param   payload  body  dtos.UploadInvoiceRequestDto  true  "Invoice Payload"
// @Success 200 {object} dtos.UploadInvoiceResponseDto "Invoice modified successfully"
// @Failure 400 {object} models.Response "Bad request"
// @Failure 422 {object} models.Response "Validation failed"
// @Router /invoice/upload [patch]
func (base *Controller) ModifyInvoice(c *fiber.Ctx) error {

	client := c.Get("client")
	userDetails, err := middleware.GetUserDetails(c)
	if err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusBadRequest, "error", "unable to get user claims", nil, nil)
		return c.Status(fiber.StatusBadRequest).JSON(rd)
	}

	db := middleware.GetDatabaseInstance(userDetails.IsSandbox, base.Db, base.TestDB)
	setup, err := businessservice.ValidateInvoiceUploadSetup(db, userDetails.ID)
	if err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusBadRequest, "error", err.Error(), nil, nil)
		return c.Status(fiber.StatusBadRequest).JSON(rd)
	}

	var req dtos.UploadInvoiceRequestDto
	if err := c.BodyParser(&req); err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusBadRequest, "error", "Failed to parse request body", err, nil)
		return c.Status(fiber.StatusBadRequest).JSON(rd)
	}

	if err := base.Validator.Struct(&req); err != nil {
		rd := utility.BuildErrorResponse(
			fiber.StatusUnprocessableEntity,
			"error", "Validation failed",
			utility.ValidationErrorsToJSON(err, firs_models.InvoiceRequest{}),
			nil,
		)
		return c.Status(fiber.StatusUnprocessableEntity).JSON(rd)
	}

	// Look up existing invoice
	existingInvoice, err := invoice.GetInvoiceByInvoiceNumber(db, req.InvoiceNumber, userDetails.ID)
	if err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusBadRequest, "error", err.Error(), err, nil)
		return c.Status(fiber.StatusBadRequest).JSON(rd)
	}
	if existingInvoice == nil {
		rd := utility.BuildErrorResponse(fiber.StatusNotFound, "error", "invoice not found with the given invoice number", nil, nil)
		return c.Status(fiber.StatusNotFound).JSON(rd)
	}

	// Deprecate old invoice on NRS — abort if this fails
	oldIRN := existingInvoice.IRN

	blockedStatuses := map[string]bool{
		models.StatusSignedInvoice: true,
		models.StatusTransmitted:   true,
		models.StatusConfirmed:     true,
	}

	if blockedStatuses[existingInvoice.CurrentStatus] {
		now := time.Now().UTC()
		created := existingInvoice.CreatedAt.UTC()
		if now.Year() == created.Year() && now.YearDay() == created.YearDay() {
			rd := utility.BuildErrorResponse(fiber.StatusFailedDependency, "error", "invoice can only be modified on a different day from its creation", nil, nil)
			return c.Status(fiber.StatusNotFound).JSON(rd)
		}

		if err := invoice.DeprecateInvoiceOnNRS(oldIRN, userDetails.IsSandbox); err != nil {
			rd := utility.BuildErrorResponse(fiber.StatusBadRequest, "error", "failed to deprecate old invoice on NRS: "+err.Error(), nil, nil)
			return c.Status(fiber.StatusBadRequest).JSON(rd)
		}
	}

	// Generate fresh IRN (always new, ignore any IRN in the request)
	irnData, irnErr := invoice.IRNGeneration(db, userDetails.ID, req.InvoiceNumber, setup.ServiceID, req.BusinessID, userDetails.IsSandbox)
	if irnErr != nil {
		rd := *irnErr
		return c.Status(fiber.StatusBadRequest).JSON(rd)
	}
	req.IRN = &irnData.IRN

	// Hard-replace the old record in-place
	replacedInvoice, err := invoice.ReplaceInvoiceRecord(db, existingInvoice, req, *req.IRN, irnData.QRCode, irnData.QRCode2, client)
	if err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusInternalServerError, "error", err.Error(), nil, nil)
		return c.Status(fiber.StatusInternalServerError).JSON(rd)
	}

	// Run full FIRS pipeline on the new invoice
	firsErr, isInvoiceSigned := invoice.FirsAllInOneProcess(req, replacedInvoice, db, userDetails.IsSandbox)

	response := map[string]interface{}{
		"metadata": replacedInvoice.StatusHistory,
	}
	if isInvoiceSigned {
		response["data"] = map[string]interface{}{
			"id":             replacedInvoice.ID,
			"invoice_number": irnData.InvoiceNumber,
			"irn":            irnData.IRN,
			"qr_code":        irnData.QRCode,
			"qr_code_2":      irnData.QRCode2,
		}
	}

	if firsErr != nil {
		errorArray := strings.Split(firsErr.Error(), "-")
		rd := utility.BuildErrorResponse(fiber.StatusBadRequest, "error", errorArray[len(errorArray)-1], response, nil)
		return c.Status(fiber.StatusBadRequest).JSON(rd)
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
// @Success 200 {object} dtos.GetInvoiceStatsResponseDto "Invoice statistics fetched successfully"
// @Failure 401 {object} models.Response "Unauthorized"
// @Router /invoice/stats [get]
func (base *Controller) GetInvoiceStats(c *fiber.Ctx) error {
	userDetails, err := middleware.GetUserDetails(c)
	if err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusUnauthorized, "error", "Unauthorized", err, nil)
		return c.Status(fiber.StatusUnauthorized).JSON(rd)
	}

	db := middleware.GetDatabaseInstance(userDetails.IsSandbox, base.Db, base.TestDB)

	var businessID, aggregatorID *string
	if userDetails.IsAggregator {
		aggregatorID = &userDetails.ID
	} else {
		businessID = &userDetails.ID
	}

	stats, err := invoice.GetInvoiceStats(db, businessID, aggregatorID)
	if err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusInternalServerError, "error", "failed to retrieve invoice stats", err, nil)
		return c.Status(fiber.StatusInternalServerError).JSON(rd)
	}

	rd := utility.BuildSuccessResponse(fiber.StatusOK, "Invoice statistics fetched successfully", stats)
	return c.Status(fiber.StatusOK).JSON(rd)
}
