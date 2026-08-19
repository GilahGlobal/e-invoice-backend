package invoice

import (
	"einvoice-access-point/internal/app/business"
	"einvoice-access-point/internal/app/token"
	"einvoice-access-point/internal/apperror"
	"einvoice-access-point/internal/data/database"
	"einvoice-access-point/internal/data/dbinit"
	"einvoice-access-point/internal/data/entities"
	"einvoice-access-point/internal/middleware"
	"einvoice-access-point/internal/pkg/cloudinary"
	"einvoice-access-point/internal/pkg/firs_models"
	"einvoice-access-point/internal/utility"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
)

type Handler struct {
	svc         *Service
	businessSvc *business.Service
	Validator   *validator.Validate
	Logger      *utility.Logger
	Db          *database.Database
	TestDB      *database.Database
	Keys        *utility.CryptoKeys
}

func NewHandler(validator *validator.Validate, logger *utility.Logger, db, testDB *database.Database, keys *utility.CryptoKeys) *Handler {
	prodDB := dbinit.InitDB(db.Postgresql.DB(), false)
	testDBConn := dbinit.InitDB(testDB.Postgresql.DB(), false)
	tokenSvc := token.NewServiceWithDB(prodDB, testDBConn)
	businessSvc := business.NewServiceWithDB(prodDB, testDBConn)
	svc := NewServiceWithDB(prodDB, testDBConn, tokenSvc, businessSvc)
	return &Handler{
		svc:         svc,
		businessSvc: businessSvc,
		Validator:   validator,
		Logger:      logger,
		Db:          db,
		TestDB:      testDB,
		Keys:        keys,
	}
}

// @Summary Get All Invoices
// @Description Fetch all invoices for the authenticated user/business
// @Tags Invoice
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number"
// @Param size query int false "Page size"
// @Success 200 {object} GetAllInvoicesResponseDto
// @Failure 400 {object} entities.Response
// @Failure 401 {object} entities.Response
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

// @Summary Get Invoice Details
// @Description Fetch details of a specific invoice by ID
// @Tags Invoice
// @Produce json
// @Security BearerAuth
// @Param invoice_id path string true "Invoice ID"
// @Success 200 {object} GetInvoiceDetailsResponseDto
// @Failure 400 {object} entities.Response
// @Failure 401 {object} entities.Response
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

// @Summary Delete Invoice
// @Description Delete a specific invoice by ID
// @Tags Invoice
// @Produce json
// @Security BearerAuth
// @Param invoice_id path string true "Invoice ID"
// @Success 200 {object} entities.Response
// @Failure 400 {object} entities.Response
// @Failure 401 {object} entities.Response
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

// @Summary Upload Invoice
// @Description Upload a new invoice
// @Tags Invoice
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body firs_models.UploadInvoiceRequestDto true "Invoice Upload Request"
// @Success 201 {object} UploadInvoiceResponseDto
// @Failure 400 {object} entities.Response
// @Failure 401 {object} entities.Response
// @Failure 422 {object} entities.Response
// @Router /invoice [post]
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
		
		isBlocked := blockedStatuses[invoiceExists.CurrentStatus]
		if invoiceExists.CurrentStatus == entities.StatusSignedInvoice && invoiceExists.HasFailedStatus() {
			isBlocked = false
		}
		
		if isBlocked {
			qrCode := invoiceExists.QrCode
			if qrCode == "" {
				qrCode = invoiceExists.EncryptedIRN
			}

			response := map[string]interface{}{
				"metadata": invoiceExists.StatusHistory,
			}

			dataMap := map[string]interface{}{
				"id":             invoiceExists.ID,
				"invoice_number": invoiceExists.InvoiceNumber,
				"irn":            invoiceExists.IRN,
				"qr_code":        qrCode,
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
			keys, err := h.businessSvc.ResolveBusinessIRNSigningKeys(db, userDetails.ID, userDetails.IsSandbox, h.Keys)
			if err != nil {
				return apperror.New(fiber.StatusBadRequest, "error", err.Error(), err, nil)
			}

			signedIRN, err := h.svc.SignIRN(*req.IRN, keys)
			if err != nil {
				return apperror.New(fiber.StatusBadRequest, "error", err.Error(), err, nil)
			}

			irnPayload = InvoiceData{
				InvoiceNumber: req.InvoiceNumber,
				IRN:           *req.IRN,
				QRCode:        signedIRN.QrCodeImage,
				QRCode2:       signedIRN.EncryptedIRN,
				QRCodeBMP:     signedIRN.QrCodeImageBMP,
			}
		} else {
			qrCode := invoiceExists.QrCode
			if qrCode == "" {
				qrCode = invoiceExists.EncryptedIRN
			}
			irnPayload = InvoiceData{
				InvoiceNumber: req.InvoiceNumber,
				IRN:           *req.IRN,
				QRCode:        qrCode,
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

	response := map[string]interface{}{}
	if createdInvoice != nil {
		response["metadata"] = createdInvoice.StatusHistory
	}
	if isInvoiceSigned && createdInvoice != nil {
		dataMap := map[string]interface{}{
			"id":             createdInvoice.ID,
			"invoice_number": irnPayload.InvoiceNumber,
			"irn":            irnPayload.IRN,
			"qr_code":        createdInvoice.QrCode,
			"qr_code_2":      createdInvoice.EncryptedIRN,
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

// @Summary Modify Invoice
// @Description Modify an existing invoice
// @Tags Invoice
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param invoice_id path string true "Invoice ID"
// @Param request body firs_models.UploadInvoiceRequestDto true "Invoice Modify Request"
// @Success 200 {object} entities.Response
// @Failure 400 {object} entities.Response
// @Failure 401 {object} entities.Response
// @Failure 422 {object} entities.Response
// @Router /invoice/{invoice_id} [put]
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

	isBlocked := blockedStatuses[existingInvoice.CurrentStatus]
	if existingInvoice.CurrentStatus == entities.StatusSignedInvoice && existingInvoice.HasFailedStatus() {
		isBlocked = false
	}

	if isBlocked {
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
			"qr_code":        replacedInvoice.QrCode,
			"qr_code_2":      replacedInvoice.EncryptedIRN,
		}
		if qrCodeBMPURL != "" {
			dataMap["qr_code_bmp_url"] = qrCodeBMPURL
		}
		response["data"] = dataMap
	}

	if firsErr != nil {
		normalizedErr := utility.ExtractRelevantErrorMessage(firsErr)
		if isInvoiceSigned {
			return apperror.New(fiber.StatusCreated, "partial_success", normalizedErr, response, nil)
		}
		return apperror.New(fiber.StatusBadRequest, "error", normalizedErr, response, nil)
	}

	rd := utility.BuildSuccessResponse(fiber.StatusOK, "Invoice modified successfully", response)
	return c.Status(fiber.StatusOK).JSON(rd)
}

// @Summary Get Invoice Stats
// @Description Get statistics for invoices
// @Tags Invoice
// @Produce json
// @Security BearerAuth
// @Success 200 {object} GetInvoiceStatsResponseDto
// @Failure 400 {object} entities.Response
// @Failure 401 {object} entities.Response
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

// @Summary Validate IRN
// @Description Validates an Invoice Reference Number (IRN)
// @Tags Invoice
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body firs_models.IRNValidationRequest true "IRN Validation Request"
// @Success 200 {object} entities.Response
// @Failure 400 {object} entities.Response
// @Failure 422 {object} entities.Response
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

	db, err := middleware.GetDatabase(c)
	if err != nil {
		return apperror.New(fiber.StatusInternalServerError, "error", err.Error(), err, nil)
	}

	if _, err := h.svc.GetInvoiceByIRN(db, req.IRN, userDetails.ID); err != nil {
		return apperror.New(fiber.StatusNotFound, "error", "invoice not found or does not belong to you", err, nil)
	}

	respData, errDetails, err := h.svc.ValidateIRN(req, userDetails.IsSandbox)
	if err != nil {
		return apperror.New(fiber.StatusBadRequest, "error", err.Error(), errDetails, nil)
	}

	h.Logger.Info("IRN validated successfully")
	rd := utility.BuildSuccessResponse(fiber.StatusOK, "IRN validated successfully", respData)
	return c.Status(fiber.StatusOK).JSON(rd)
}

// @Summary Validate Invoice
// @Description Validates an invoice payload
// @Tags Invoice
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body firs_models.UploadInvoiceRequestDto true "Invoice Request"
// @Success 200 {object} entities.Response
// @Failure 400 {object} entities.Response
// @Failure 422 {object} entities.Response
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

// @Summary Sign IRN
// @Description Sign an IRN
// @Tags Invoice
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body firs_models.IRNSigningRequestData true "Sign IRN Request"
// @Success 200 {object} entities.Response
// @Failure 400 {object} entities.Response
// @Failure 422 {object} entities.Response
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

	if _, err := h.svc.GetInvoiceByIRN(db, req.IRN, userDetails.ID); err != nil {
		return apperror.New(fiber.StatusNotFound, "error", "invoice not found or does not belong to you", err, nil)
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

// @Summary Sign Invoice
// @Description Sign an Invoice
// @Tags Invoice
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body firs_models.UploadInvoiceRequestDto true "Sign Invoice Request"
// @Success 200 {object} entities.Response
// @Failure 400 {object} entities.Response
// @Failure 422 {object} entities.Response
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

// @Summary Generate IRN
// @Description Generate an IRN for an invoice
// @Tags Invoice
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body firs_models.GenerateIRNRequestData true "Generate IRN Request"
// @Success 200 {object} entities.Response
// @Failure 400 {object} entities.Response
// @Failure 422 {object} entities.Response
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

// @Summary Update Invoice by IRN
// @Description Update an invoice by its IRN
// @Tags Invoice
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param irn path string true "IRN"
// @Param request body firs_models.UpdateInvoice true "Update Invoice Request"
// @Success 200 {object} entities.Response
// @Failure 400 {object} entities.Response
// @Failure 422 {object} entities.Response
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

	if _, err := h.svc.GetInvoiceByIRN(db, irn, userDetails.ID); err != nil {
		return apperror.New(fiber.StatusNotFound, "error", "invoice not found or does not belong to you", err, nil)
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

// @Summary Confirm Invoice
// @Description Confirm an invoice
// @Tags Invoice
// @Produce json
// @Security BearerAuth
// @Param irn path string true "IRN"
// @Success 200 {object} entities.Response
// @Failure 400 {object} entities.Response
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

	db, err := middleware.GetDatabase(c)
	if err != nil {
		return apperror.New(fiber.StatusInternalServerError, "error", err.Error(), err, nil)
	}

	if _, err := h.svc.GetInvoiceByIRN(db, irn, userDetails.ID); err != nil {
		return apperror.New(fiber.StatusNotFound, "error", "invoice not found or does not belong to you", err, nil)
	}

	respData, errDetails, err := h.svc.ConfirmInvoice(irn, userDetails.IsSandbox)
	if err != nil {
		return apperror.New(fiber.StatusBadRequest, "error", err.Error(), errDetails, nil)
	}

	h.Logger.Info("Invoice confirmed with irn successfully")
	rd := utility.BuildSuccessResponse(fiber.StatusOK, "Invoice confirmed with irn successfully", respData)
	return c.Status(fiber.StatusOK).JSON(rd)
}

// @Summary Download Invoice
// @Description Download an invoice
// @Tags Invoice
// @Produce json
// @Security BearerAuth
// @Param irn path string true "IRN"
// @Success 200 {object} entities.Response
// @Failure 400 {object} entities.Response
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

	db, err := middleware.GetDatabase(c)
	if err != nil {
		return apperror.New(fiber.StatusInternalServerError, "error", err.Error(), err, nil)
	}

	if _, err := h.svc.GetInvoiceByIRN(db, irn, userDetails.ID); err != nil {
		return apperror.New(fiber.StatusNotFound, "error", "invoice not found or does not belong to you", err, nil)
	}

	respData, errDetails, err := h.svc.DownloadInvoice(irn, userDetails.IsSandbox)
	if err != nil {
		return apperror.New(fiber.StatusBadRequest, "error", err.Error(), errDetails, nil)
	}

	h.Logger.Info("Invoice downloaded with irn successfully")
	rd := utility.BuildSuccessResponse(fiber.StatusOK, "Invoice downloaded with irn successfully", respData)
	return c.Status(fiber.StatusOK).JSON(rd)
}

// @Summary Bulk Update Invoices
// @Description Bulk update multiple invoices
// @Tags Invoice
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body firs_models.BulkUpdateInvoiceRequest true "Bulk Update Request"
// @Success 200 {object} BulkUpdateInvoiceResponseDto
// @Failure 400 {object} entities.Response
// @Failure 422 {object} entities.Response
// @Router /invoice/update [patch]
func (h *Handler) BulkUpdateInvoice(c *fiber.Ctx) error {
	userDetails, err := middleware.GetUserDetails(c)
	if err != nil {
		return apperror.New(fiber.StatusUnauthorized, "error", "Unauthorized", err, nil)
	}

	db, err := middleware.GetDatabase(c)
	if err != nil {
		return apperror.New(fiber.StatusInternalServerError, "error", err.Error(), err, nil)
	}

	var req firs_models.BulkUpdateInvoiceRequest
	if err = c.BodyParser(&req); err != nil {
		return apperror.New(fiber.StatusBadRequest, "error", "Failed to parse request body", err, nil)
	}

	if err = h.Validator.Struct(&req); err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusUnprocessableEntity, "error", "Validation failed", utility.ValidationResponse(err, h.Validator), nil)
		return c.Status(fiber.StatusUnprocessableEntity).JSON(rd)
	}

	for _, invoiceReq := range req.Invoices {
		if _, err := h.svc.GetInvoiceByIRN(db, invoiceReq.IRN, userDetails.ID); err != nil {
			return apperror.New(fiber.StatusNotFound, "error", "invoice not found or does not belong to you: "+invoiceReq.IRN, err, nil)
		}
	}

	respData, err := h.svc.BulkUpdateInvoice(db, userDetails.ID, req, userDetails.IsSandbox)
	if err != nil {
		return apperror.New(fiber.StatusBadRequest, "error", err.Error(), err, nil)
	}

	h.Logger.Info("Bulk update completed")
	rd := utility.BuildSuccessResponse(fiber.StatusOK, "Bulk update completed", respData)
	return c.Status(fiber.StatusOK).JSON(rd)
}

// @Summary Look Up IRN
// @Description Look up an invoice by IRN
// @Tags Invoice
// @Produce json
// @Security BearerAuth
// @Param irn path string true "IRN"
// @Success 200 {object} entities.Response
// @Failure 400 {object} entities.Response
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

// @Summary Look Up TIN
// @Description Look up by TIN
// @Tags Invoice
// @Produce json
// @Security BearerAuth
// @Param tin path string true "TIN"
// @Success 200 {object} entities.Response
// @Failure 400 {object} entities.Response
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

// @Summary Look Up Party ID
// @Description Look up by Party ID
// @Tags Invoice
// @Produce json
// @Security BearerAuth
// @Param party_id path string true "Party ID"
// @Success 200 {object} entities.Response
// @Failure 400 {object} entities.Response
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

// @Summary Transmit Invoice
// @Description Transmit an invoice by IRN
// @Tags Invoice
// @Produce json
// @Security BearerAuth
// @Param irn path string true "IRN"
// @Success 200 {object} entities.Response
// @Failure 400 {object} entities.Response
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

// @Summary Confirm Transmitted Invoice
// @Description Confirm a transmitted invoice by IRN
// @Tags Invoice
// @Produce json
// @Security BearerAuth
// @Param irn path string true "IRN"
// @Success 200 {object} entities.Response
// @Failure 400 {object} entities.Response
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

// @Summary Pull Transmitted Invoices
// @Description Pull transmitted invoices
// @Tags Invoice
// @Produce json
// @Security BearerAuth
// @Param start_date query string false "Start Date"
// @Param end_date query string false "End Date"
// @Success 200 {object} entities.Response
// @Failure 400 {object} entities.Response
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

// @Summary Transmit Health Check
// @Description Health check for FIRS transmission
// @Tags Invoice
// @Produce json
// @Security BearerAuth
// @Success 200 {object} entities.Response
// @Failure 400 {object} entities.Response
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
