package aggregator

import (
	"context"
	"einvoice-access-point/internal/dtos"
	aggregatorSvc "einvoice-access-point/internal/services/aggregator"
	invoiceSvc "einvoice-access-point/internal/services/invoice"
	"einvoice-access-point/pkg/middleware"
	"einvoice-access-point/pkg/models"
	"einvoice-access-point/pkg/s3"
	"einvoice-access-point/pkg/utility"
	"einvoice-access-point/pkg/workers"
	"einvoice-access-point/pkg/workers/producer"
	"log"

	"github.com/gofiber/fiber/v2"
)

// ListInvitations fetches all pending invitations for the aggregator
func (base *Controller) ListInvitations(c *fiber.Ctx) error {
	userDetails, err := middleware.GetUserDetails(c)
	if err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusUnauthorized, "error", "Unauthorized", err, nil)
		return c.Status(fiber.StatusUnauthorized).JSON(rd)
	}

	db := middleware.GetDatabaseInstance(userDetails.IsSandbox, base.Db, base.TestDB)

	invitations, err := aggregatorSvc.ListAggregatorInvitations(userDetails.ID, db)
	if err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusBadRequest, "error", err.Error(), err, nil)
		return c.Status(fiber.StatusBadRequest).JSON(rd)
	}

	rd := utility.BuildSuccessResponse(fiber.StatusOK, "Invitations fetched successfully", dtos.AggregatorInvitationListResponseDto{Data: invitations})
	return c.Status(fiber.StatusOK).JSON(rd)
}

// RespondToInvitation handles accepting or rejecting an invitation
func (base *Controller) RespondToInvitation(c *fiber.Ctx) error {
	userDetails, err := middleware.GetUserDetails(c)
	if err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusUnauthorized, "error", "Unauthorized", err, nil)
		return c.Status(fiber.StatusUnauthorized).JSON(rd)
	}

	var req dtos.RespondToInvitationDto
	if err := c.BodyParser(&req); err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusBadRequest, "error", "Failed to parse request body", err, nil)
		return c.Status(fiber.StatusBadRequest).JSON(rd)
	}

	if err := base.Validator.Struct(&req); err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusUnprocessableEntity, "error", "Validation failed", utility.ValidationErrorsToJSON(err, dtos.RespondToInvitationDto{}), nil)
		return c.Status(fiber.StatusUnprocessableEntity).JSON(rd)
	}

	db := middleware.GetDatabaseInstance(userDetails.IsSandbox, base.Db, base.TestDB)

	status, err := aggregatorSvc.RespondToInvitation(req.InvitationID, userDetails.ID, req.Accept, db)
	if err != nil {
		rd := utility.BuildErrorResponse(status, "error", err.Error(), err, nil)
		return c.Status(status).JSON(rd)
	}

	rd := utility.BuildSuccessResponse(status, "Responded to invitation successfully", nil)
	return c.Status(status).JSON(rd)
}

// Dashboard fetches high level stats
func (base *Controller) Dashboard(c *fiber.Ctx) error {
	userDetails, err := middleware.GetUserDetails(c)
	if err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusUnauthorized, "error", "Unauthorized", err, nil)
		return c.Status(fiber.StatusUnauthorized).JSON(rd)
	}

	db := middleware.GetDatabaseInstance(userDetails.IsSandbox, base.Db, base.TestDB)

	stats, err := aggregatorSvc.GetDashboard(userDetails.ID, db)
	if err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusInternalServerError, "error", err.Error(), err, nil)
		return c.Status(fiber.StatusInternalServerError).JSON(rd)
	}

	rd := utility.BuildSuccessResponse(fiber.StatusOK, "Dashboard stats fetched successfully", stats)
	return c.Status(fiber.StatusOK).JSON(rd)
}

// ListBusinesses lists accepted businesses for an aggregator
func (base *Controller) ListBusinesses(c *fiber.Ctx) error {
	userDetails, err := middleware.GetUserDetails(c)
	if err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusUnauthorized, "error", "Unauthorized", err, nil)
		return c.Status(fiber.StatusUnauthorized).JSON(rd)
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
	search := c.Query("search", "")

	db := middleware.GetDatabaseInstance(userDetails.IsSandbox, base.Db, base.TestDB)

	businesses, pagination, err := aggregatorSvc.ListBusinesses(userDetails.ID, query.Page, query.Size, search, db)
	if err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusInternalServerError, "error", err.Error(), err, nil)
		return c.Status(fiber.StatusInternalServerError).JSON(rd)
	}

	rd := utility.BuildSuccessResponse(fiber.StatusOK, "Businesses fetched successfully", businesses, pagination)
	return c.Status(fiber.StatusOK).JSON(rd)
}

// GetBusinessDetail gets details for a single accepted business
func (base *Controller) GetBusinessDetail(c *fiber.Ctx) error {
	userDetails, err := middleware.GetUserDetails(c)
	if err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusUnauthorized, "error", "Unauthorized", err, nil)
		return c.Status(fiber.StatusUnauthorized).JSON(rd)
	}

	businessID := c.Params("id")
	if businessID == "" {
		rd := utility.BuildErrorResponse(fiber.StatusBadRequest, "error", "business id is required", nil, nil)
		return c.Status(fiber.StatusBadRequest).JSON(rd)
	}

	db := middleware.GetDatabaseInstance(userDetails.IsSandbox, base.Db, base.TestDB)

	business, status, err := aggregatorSvc.GetBusinessDetail(userDetails.ID, businessID, db)
	if err != nil {
		rd := utility.BuildErrorResponse(status, "error", err.Error(), err, nil)
		return c.Status(status).JSON(rd)
	}

	rd := utility.BuildSuccessResponse(status, "Business fetched successfully", business)
	return c.Status(status).JSON(rd)
}

// RemoveBusiness removes an accepted business from the aggregator
func (base *Controller) RemoveBusiness(c *fiber.Ctx) error {
	userDetails, err := middleware.GetUserDetails(c)
	if err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusUnauthorized, "error", "Unauthorized", err, nil)
		return c.Status(fiber.StatusUnauthorized).JSON(rd)
	}

	businessID := c.Params("id")
	if businessID == "" {
		rd := utility.BuildErrorResponse(fiber.StatusBadRequest, "error", "business id is required", nil, nil)
		return c.Status(fiber.StatusBadRequest).JSON(rd)
	}

	db := middleware.GetDatabaseInstance(userDetails.IsSandbox, base.Db, base.TestDB)

	status, err := aggregatorSvc.RemoveBusiness(userDetails.ID, businessID, db)
	if err != nil {
		rd := utility.BuildErrorResponse(status, "error", err.Error(), err, nil)
		return c.Status(status).JSON(rd)
	}

	rd := utility.BuildSuccessResponse(status, "Business removed successfully", nil)
	return c.Status(status).JSON(rd)
}

// ListAllInvoices gets all invoices across all businesses uploaded by this aggregator
func (base *Controller) ListAllInvoices(c *fiber.Ctx) error {
	userDetails, err := middleware.GetUserDetails(c)
	if err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusUnauthorized, "error", "Unauthorized", err, nil)
		return c.Status(fiber.StatusUnauthorized).JSON(rd)
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

	invoices, pagination, err := aggregatorSvc.ListAllInvoices(userDetails.ID, query.Page, query.Size, db)
	if err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusInternalServerError, "error", err.Error(), err, nil)
		return c.Status(fiber.StatusInternalServerError).JSON(rd)
	}

	rd := utility.BuildSuccessResponse(fiber.StatusOK, "Invoices fetched successfully", invoices, pagination)
	return c.Status(fiber.StatusOK).JSON(rd)
}

// ListBusinessInvoices gets invoices uploaded by aggregator for a specific business
func (base *Controller) ListBusinessInvoices(c *fiber.Ctx) error {
	userDetails, err := middleware.GetUserDetails(c)
	if err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusUnauthorized, "error", "Unauthorized", err, nil)
		return c.Status(fiber.StatusUnauthorized).JSON(rd)
	}

	businessID := c.Params("id")
	if businessID == "" {
		rd := utility.BuildErrorResponse(fiber.StatusBadRequest, "error", "business id is required", nil, nil)
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

	invoices, pagination, err := aggregatorSvc.ListInvoicesByBusiness(userDetails.ID, businessID, query.Page, query.Size, db)
	if err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusInternalServerError, "error", err.Error(), err, nil)
		return c.Status(fiber.StatusInternalServerError).JSON(rd)
	}

	rd := utility.BuildSuccessResponse(fiber.StatusOK, "Invoices fetched successfully", invoices, pagination)
	return c.Status(fiber.StatusOK).JSON(rd)
}

// ListAllBulkUploads gets all bulk uploads across all businesses uploaded by this aggregator
func (base *Controller) ListAllBulkUploads(c *fiber.Ctx) error {
	userDetails, err := middleware.GetUserDetails(c)
	if err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusUnauthorized, "error", "Unauthorized", err, nil)
		return c.Status(fiber.StatusUnauthorized).JSON(rd)
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

	uploads, pagination, err := aggregatorSvc.ListAllBulkUploads(userDetails.ID, query.Page, query.Size, db)
	if err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusInternalServerError, "error", err.Error(), err, nil)
		return c.Status(fiber.StatusInternalServerError).JSON(rd)
	}

	rd := utility.BuildSuccessResponse(fiber.StatusOK, "Bulk uploads fetched successfully", uploads, pagination)
	return c.Status(fiber.StatusOK).JSON(rd)
}

// ListBulkUploadLogs gets bulk uploads uploaded by aggregator for a specific business
func (base *Controller) ListBulkUploadLogs(c *fiber.Ctx) error {
	userDetails, err := middleware.GetUserDetails(c)
	if err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusUnauthorized, "error", "Unauthorized", err, nil)
		return c.Status(fiber.StatusUnauthorized).JSON(rd)
	}

	businessID := c.Params("id")
	if businessID == "" {
		rd := utility.BuildErrorResponse(fiber.StatusBadRequest, "error", "business id is required", nil, nil)
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

	uploads, pagination, err := aggregatorSvc.ListBulkUploadsByBusiness(userDetails.ID, businessID, query.Page, query.Size, db)
	if err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusInternalServerError, "error", err.Error(), err, nil)
		return c.Status(fiber.StatusInternalServerError).JSON(rd)
	}

	rd := utility.BuildSuccessResponse(fiber.StatusOK, "Bulk uploads fetched successfully", uploads, pagination)
	return c.Status(fiber.StatusOK).JSON(rd)
}

// ActivityLog fetches the activity logs sequence for the aggregator
func (base *Controller) ActivityLog(c *fiber.Ctx) error {
	userDetails, err := middleware.GetUserDetails(c)
	if err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusUnauthorized, "error", "Unauthorized", err, nil)
		return c.Status(fiber.StatusUnauthorized).JSON(rd)
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

	logs, pagination, err := aggregatorSvc.GetActivityLog(userDetails.ID, query.Page, query.Size, db)
	if err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusInternalServerError, "error", err.Error(), err, nil)
		return c.Status(fiber.StatusInternalServerError).JSON(rd)
	}

	rd := utility.BuildSuccessResponse(fiber.StatusOK, "Activity logs fetched successfully", logs, pagination)
	return c.Status(fiber.StatusOK).JSON(rd)
}

// UploadInvoice single invoice for a managed business
func (base *Controller) UploadInvoice(c *fiber.Ctx) error {
	userDetails, err := middleware.GetUserDetails(c)
	if err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusUnauthorized, "error", "Unauthorized", err, nil)
		return c.Status(fiber.StatusUnauthorized).JSON(rd)
	}

	businessID := c.Params("id")
	if businessID == "" {
		rd := utility.BuildErrorResponse(fiber.StatusBadRequest, "error", "business id is required", nil, nil)
		return c.Status(fiber.StatusBadRequest).JSON(rd)
	}

	var req dtos.UploadInvoiceRequestDto
	if err := c.BodyParser(&req); err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusBadRequest, "error", "Failed to parse request body", err, nil)
		return c.Status(fiber.StatusBadRequest).JSON(rd)
	}

	if err := base.Validator.Struct(&req); err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusUnprocessableEntity, "error", "Validation failed", utility.ValidationErrorsToJSON(err, dtos.UploadInvoiceRequestDto{}), nil)
		return c.Status(fiber.StatusUnprocessableEntity).JSON(rd)
	}

	db := middleware.GetDatabaseInstance(userDetails.IsSandbox, base.Db, base.TestDB)

	// Verify management
	business, status, err := aggregatorSvc.GetBusinessDetail(userDetails.ID, businessID, db)
	if err != nil {
		rd := utility.BuildErrorResponse(status, "error", err.Error(), err, nil)
		return c.Status(status).JSON(rd)
	}

	invoiceExists, _ := invoiceSvc.GetInvoiceByInvoiceNumber(db, req.InvoiceNumber, businessID)
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
		IRNData, irnErr := invoiceSvc.IRNGeneration(db, businessID, req.InvoiceNumber, business.ServiceID, req.BusinessID, userDetails.IsSandbox)
		if irnErr != nil {
			return c.Status(fiber.StatusBadRequest).JSON(*irnErr)
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

	createdInvoice, _, err, isInvoiceSigned := invoiceSvc.CreateInvoice(db, req, req.InvoiceNumber, businessID, irnPayload.QRCode, irnPayload.QRCode2, invoiceExists, userDetails.IsSandbox, &userDetails.ID)

	response := map[string]interface{}{
		"metadata": createdInvoice.StatusHistory,
		"irn":      irnPayload,
	}

	if isInvoiceSigned {
		rd := utility.BuildSuccessResponse(fiber.StatusCreated, "Invoice generated successfully", response)
		return c.Status(fiber.StatusCreated).JSON(rd)
	}

	rd := utility.BuildErrorResponse(fiber.StatusCreated, "error", "failed to complete irn and invoice signing", response, nil)
	return c.Status(fiber.StatusCreated).JSON(rd)
}

// BulkUpload bulk invoices for a managed business
func (base *Controller) BulkUpload(c *fiber.Ctx) error {
	userDetails, err := middleware.GetUserDetails(c)
	if err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusUnauthorized, "error", "Unauthorized", err, nil)
		return c.Status(fiber.StatusUnauthorized).JSON(rd)
	}

	businessID := c.Params("id")
	if businessID == "" {
		rd := utility.BuildErrorResponse(fiber.StatusBadRequest, "error", "business id is required", nil, nil)
		return c.Status(fiber.StatusBadRequest).JSON(rd)
	}

	db := middleware.GetDatabaseInstance(userDetails.IsSandbox, base.Db, base.TestDB)

	business, status, err := aggregatorSvc.GetBusinessDetail(userDetails.ID, businessID, db)
	if err != nil {
		rd := utility.BuildErrorResponse(status, "error", err.Error(), err, nil)
		return c.Status(status).JSON(rd)
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

	bulkID, err := invoiceSvc.AddBulkUploadLog(db, fileURL, fileKey, businessID, &userDetails.ID)
	if err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusInternalServerError, "error", "failed to log bulk upload", nil, nil)
		return c.Status(fiber.StatusInternalServerError).JSON(rd)
	}

	err = producer.NewProducer().EnqueueTask(workers.BulkUploadTask, workers.BulkUploadInput{
		BulkID:       bulkID,
		ID:           businessID, // Owner ID of the business for signing etc
		FileKey:      fileKey,
		ServiceID:    business.ServiceID,
		BusinessID:   businessID,
		IsSandbox:    userDetails.IsSandbox,
		AggregatorID: &userDetails.ID,
	})
	if err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusInternalServerError, "error", "failed to enqueue bulk upload task", nil, nil)
		return c.Status(fiber.StatusInternalServerError).JSON(rd)
	}

	rd := utility.BuildSuccessResponse(fiber.StatusCreated, "Invoice uploaded successfully", fileURL)
	return c.Status(fiber.StatusCreated).JSON(rd)
}
