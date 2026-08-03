package bulk_upload

import (
	"context"
	"einvoice-access-point/internal/app/business"
	"einvoice-access-point/internal/apperror"
	"einvoice-access-point/internal/data/database"
	"einvoice-access-point/internal/data/dbinit"
	"einvoice-access-point/internal/data/entities"
	"einvoice-access-point/internal/middleware"
	"einvoice-access-point/internal/pkg/cloudinary"
	"einvoice-access-point/internal/utility"
	"einvoice-access-point/internal/workers"
	"einvoice-access-point/internal/workers/producer"
	"log"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
)

// Handler handles HTTP requests for bulk upload operations.
type Handler struct {
	svc         *Service
	businessSvc *business.Service
	Validator   *validator.Validate
	Logger      *utility.Logger
	Db          *database.Database
	TestDB      *database.Database
}

// NewHandler creates a new bulk upload Handler.
func NewHandler(validator *validator.Validate, logger *utility.Logger, db, testDB *database.Database) *Handler {
	prodDB := dbinit.InitDB(db.Postgresql.DB(), false)
	testDBConn := dbinit.InitDB(testDB.Postgresql.DB(), false)
	svc := NewServiceWithDB(prodDB, testDBConn)
	businessSvc := business.NewServiceWithDB(prodDB, testDBConn)
	return &Handler{
		svc:         svc,
		businessSvc: businessSvc,
		Validator:   validator,
		Logger:      logger,
		Db:          db,
		TestDB:      testDB,
	}
}

// @Summary Get bulk upload logs
// @Description Retrieve paginated bulk upload logs for a business
// @Tags Invoice
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param query query entities.PaginationQuery false "Pagination (page, size)"
// @Success 200 {object} GetBulkUploadLogsResponseDto "Bulk upload logs fetched successfully"
// @Failure 400 {object} entities.Response "Bad request"
// @Failure 401 {object} entities.Response "Unauthorized"
// @Router /invoice/bulk-upload [get]
func (h *Handler) GetBulkUploadLogs(c *fiber.Ctx) error {
	userDetails, err := middleware.GetUserDetails(c)
	if err != nil {
		return apperror.New(fiber.StatusUnauthorized, "error", "Unauthorized", err, nil)
	}

	if userDetails.BusinessID == nil || *userDetails.BusinessID == "" {
		return apperror.New(fiber.StatusBadRequest, "error", "business_id is required", nil, nil)
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

	db, err := middleware.GetDatabase(c)
	if err != nil {
		return apperror.New(fiber.StatusInternalServerError, "error", err.Error(), err, nil)
	}
	logs, pagination, err := h.svc.GetBulkUploadLogsByBusinessID(db, *userDetails.BusinessID, query.Page, query.Size)
	if err != nil {
		return apperror.New(fiber.StatusBadRequest, "error", err.Error(), err, nil)
	}

	rd := utility.BuildSuccessResponse(fiber.StatusOK, "Bulk upload logs fetched successfully", logs, pagination)
	return c.Status(fiber.StatusOK).JSON(rd)
}

// @Summary Get failed invoices from a bulk upload
// @Description Retrieve the failed invoices recorded for a specific bulk upload belonging to the authenticated business
// @Tags Invoice
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param bulk_id path string true "Bulk upload ID"
// @Success 200 {object} GetBulkUploadFailedInvoicesResponseDto "Bulk upload failed invoices fetched successfully"
// @Failure 400 {object} entities.Response "Bad request"
// @Failure 401 {object} entities.Response "Unauthorized"
// @Failure 404 {object} entities.Response "Bulk upload not found"
// @Failure 500 {object} entities.Response "Internal server error"
// @Router /invoice/bulk-upload/{bulk_id}/failed [get]
func (h *Handler) GetBulkUploadFailedInvoices(c *fiber.Ctx) error {
	userDetails, err := middleware.GetUserDetails(c)
	if err != nil {
		return apperror.New(fiber.StatusUnauthorized, "error", "Unauthorized", err, nil)
	}

	if userDetails.BusinessID == nil || *userDetails.BusinessID == "" {
		return apperror.New(fiber.StatusBadRequest, "error", "business_id is required", nil, nil)
	}

	bulkUploadID := c.Params("bulk_id")
	if bulkUploadID == "" {
		return apperror.New(fiber.StatusBadRequest, "error", "bulk upload id is required", nil, nil)
	}

	db, err := middleware.GetDatabase(c)
	if err != nil {
		return apperror.New(fiber.StatusInternalServerError, "error", err.Error(), err, nil)
	}
	failedInvoices, err := h.svc.GetBulkUploadFailedInvoices(db, bulkUploadID, *userDetails.BusinessID)
	if err != nil {
		status := fiber.StatusInternalServerError
		if strings.Contains(strings.ToLower(err.Error()), "not found") {
			status = fiber.StatusNotFound
		}

		return apperror.New(status, "error", err.Error(), err, nil)
	}

	rd := utility.BuildSuccessResponse(fiber.StatusOK, "Bulk upload failed invoices fetched successfully", failedInvoices)
	return c.Status(fiber.StatusOK).JSON(rd)
}

// @Summary Download failed invoices from a bulk upload
// @Description Download the failed invoices recorded for a specific bulk upload belonging to the authenticated business as csv or excel
// @Tags Invoice
// @Produce text/csv,application/vnd.openxmlformats-officedocument.spreadsheetml.sheet
// @Security BearerAuth
// @Param bulk_id path string true "Bulk upload ID"
// @Param format query string false "Export format" Enums(csv,excel,xlsx)
// @Success 200 {file} file "Bulk upload failed invoices file"
// @Failure 400 {object} entities.Response "Bad request"
// @Failure 401 {object} entities.Response "Unauthorized"
// @Failure 404 {object} entities.Response "Bulk upload not found"
// @Failure 500 {object} entities.Response "Internal server error"
// @Router /invoice/bulk-upload/{bulk_id}/failed/download [get]
func (h *Handler) DownloadBulkUploadFailedInvoices(c *fiber.Ctx) error {
	userDetails, err := middleware.GetUserDetails(c)
	if err != nil {
		return apperror.New(fiber.StatusUnauthorized, "error", "Unauthorized", err, nil)
	}

	if userDetails.BusinessID == nil || *userDetails.BusinessID == "" {
		return apperror.New(fiber.StatusBadRequest, "error", "business_id is required", nil, nil)
	}

	bulkUploadID := c.Params("bulk_id")
	if bulkUploadID == "" {
		return apperror.New(fiber.StatusBadRequest, "error", "bulk upload id is required", nil, nil)
	}

	db, err := middleware.GetDatabase(c)
	if err != nil {
		return apperror.New(fiber.StatusInternalServerError, "error", err.Error(), err, nil)
	}
	failedInvoices, err := h.svc.GetBulkUploadFailedInvoices(db, bulkUploadID, *userDetails.BusinessID)
	if err != nil {
		status := fiber.StatusInternalServerError
		if strings.Contains(strings.ToLower(err.Error()), "not found") {
			status = fiber.StatusNotFound
		}

		return apperror.New(status, "error", err.Error(), err, nil)
	}

	fileData, contentType, extension, err := h.svc.ExportBulkUploadFailedInvoices(failedInvoices, c.Query("format", "csv"))
	if err != nil {
		return apperror.New(fiber.StatusBadRequest, "error", err.Error(), err, nil)
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
// @Success 200 {object} entities.Response "Invoice created successfully"
// @Failure 400 {object} entities.Response "Bad request"
// @Router /invoice/create [post]
func (h *Handler) CreateInvoice(c *fiber.Ctx) error {

	userDetails, err := middleware.GetUserDetails(c)
	if err != nil {
		return apperror.New(fiber.StatusUnauthorized, "error", "Unauthorized", err, nil)
	}

	db, err := middleware.GetDatabase(c)
	if err != nil {
		return apperror.New(fiber.StatusInternalServerError, "error", err.Error(), err, nil)
	}

	setup, err := h.businessSvc.ValidateInvoiceUploadSetup(db, userDetails.ID)
	if err != nil {
		return apperror.New(fiber.StatusBadRequest, "error", err.Error(), nil, nil)
	}

	file, err := c.FormFile("file")
	if err != nil {
		return apperror.New(fiber.StatusBadRequest, "error", "invoice JSON file is required", nil, nil)
	}

	fileContent, err := file.Open()
	if err != nil {
		return apperror.New(fiber.StatusBadRequest, "error", "failed to read file", nil, nil)
	}
	defer fileContent.Close()

	ctx := context.Background()
	fileURL, fileKey, err := cloudinary.UploadRawFile(ctx, fileContent, file.Filename)
	if err != nil {
		log.Println("Cloudinary upload failed:", err)
		return c.Status(500).JSON(fiber.Map{"error": "upload failed"})
	}

	bulkID, err := h.svc.AddBulkUploadLog(db, fileURL, fileKey, setup.BusinessID, nil)
	if err != nil {
		return apperror.New(fiber.StatusInternalServerError, "error", "failed to log bulk upload", nil, nil)
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
		return apperror.New(fiber.StatusInternalServerError, "error", "failed to enqueue bulk upload task", nil, nil)
	}

	rd := utility.BuildSuccessResponse(fiber.StatusCreated, "Invoice uploaded successfully", fileURL)
	return c.Status(fiber.StatusCreated).JSON(rd)
}
