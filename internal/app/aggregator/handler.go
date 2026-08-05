package aggregator

import (
	"context"
	"einvoice-access-point/internal/app/bulk_upload"
	"einvoice-access-point/internal/app/business"
	"einvoice-access-point/internal/app/invoice"
	"einvoice-access-point/internal/app/subscription"
	"einvoice-access-point/internal/app/token"
	"einvoice-access-point/internal/apperror"
	"einvoice-access-point/internal/config"
	"einvoice-access-point/internal/data/database"
	"einvoice-access-point/internal/data/dbinit"
	"einvoice-access-point/internal/data/entities"
	"einvoice-access-point/internal/middleware"
	"einvoice-access-point/internal/pkg/cloudinary"
	"einvoice-access-point/internal/pkg/s3"
	"einvoice-access-point/internal/utility"
	"einvoice-access-point/internal/workers"
	"einvoice-access-point/internal/workers/producer"
	"io"
	"log"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
)

type Handler struct {
	svc           *Service
	subSvc        *subscription.Service
	businessSvc   *business.Service
	invoiceSvc    *invoice.Service
	bulkUploadSvc *bulk_upload.Service
	Validator     *validator.Validate
	Logger        *utility.Logger
	Db            *database.Database
	TestDb        *database.Database
}

func NewHandler(validator *validator.Validate, logger *utility.Logger, db, testDb *database.Database, cfg *config.Configuration) *Handler {
	prodDB := dbinit.InitDB(db.Postgresql.DB(), false)
	testDBConn := dbinit.InitDB(testDb.Postgresql.DB(), false)

	businessSvc := business.NewServiceWithDB(prodDB, testDBConn)
	bulkUploadSvc := bulk_upload.NewServiceWithDB(prodDB, testDBConn)
	tokenSvc := token.NewServiceWithDB(prodDB, testDBConn)
	invoiceSvc := invoice.NewServiceWithDB(prodDB, testDBConn, tokenSvc, businessSvc)
	subSvc := subscription.NewServiceWithDB(prodDB, testDBConn)

	svc := NewServiceWithDB(prodDB, testDBConn, subSvc, businessSvc, bulkUploadSvc, cfg)

	return &Handler{
		svc:           svc,
		subSvc:        subSvc,
		businessSvc:   businessSvc,
		invoiceSvc:    invoiceSvc,
		bulkUploadSvc: bulkUploadSvc,
		Validator:     validator,
		Logger:        logger,
		Db:            db,
		TestDb:        testDb,
	}
}

// @Summary List Available Aggregators
// @Description Fetch all available aggregators
// @Tags Business Aggregator Portal
// @Produce json
// @Security BearerAuth
// @Success 200 {object} AggregatorInvitationListResponseDto "Aggregators fetched successfully"
// @Failure 401 {object} entities.Response "Unauthorized"
// @Failure 500 {object} entities.Response "Internal server error"
// @Router /business/aggregators [get]
func (h *Handler) ListAvailableAggregators(c *fiber.Ctx) error {
	userDetails, err := middleware.GetUserDetails(c)
	if err != nil {
		return apperror.New(fiber.StatusUnauthorized, "error", "Unauthorized", err, nil)
	}

	if userDetails.IsAggregator {
		return apperror.New(fiber.StatusForbidden, "error", "Aggregator account cannot view other aggregators", nil, nil)
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
	search := c.Query("search", "")

	db, err := middleware.GetDatabase(c)
	if err != nil {
		return apperror.New(fiber.StatusInternalServerError, "error", err.Error(), err, nil)
	}

	aggregators, total, err := h.svc.ListAvailableAggregators(search, query.Page, query.Size, db)
	if err != nil {
		return apperror.New(fiber.StatusInternalServerError, "error", err.Error(), err, nil)
	}

	rd := utility.BuildSuccessResponse(fiber.StatusOK, "Aggregators fetched successfully", map[string]interface{}{
		"aggregators": aggregators,
		"total":       total,
		"page":        query.Page,
		"size":        query.Size,
	})
	return c.Status(fiber.StatusOK).JSON(rd)
}

// @Summary Send Invitation
// @Description Send an invitation to an aggregator
// @Tags Business Aggregator Portal
// @Produce json
// @Security BearerAuth
// @Param data body SendAggregatorInvitationDto true "Send aggregator invitation request payload"
// @Success 200 {object} entities.Response "Invitation sent successfully"
// @Failure 401 {object} entities.Response "Unauthorized"
// @Failure 500 {object} entities.Response "Internal server error"
// @Router /business/aggregators/invite [post]
func (h *Handler) SendAggregatorInvitation(c *fiber.Ctx) error {
	userDetails, err := middleware.GetUserDetails(c)

	if err != nil {
		return apperror.New(fiber.StatusUnauthorized, "error", "Unauthorized", err, nil)
	}

	if userDetails.IsAggregator {
		return apperror.New(fiber.StatusForbidden, "error", "Aggregator account cannot send invitations", nil, nil)
	}

	var req SendAggregatorInvitationDto
	if err := c.BodyParser(&req); err != nil {
		return apperror.New(fiber.StatusBadRequest, "error", "Failed to parse request body", err, nil)
	}

	if err := h.Validator.Struct(&req); err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusUnprocessableEntity, "error", "Validation failed", utility.ValidationErrorsToJSON(err, SendAggregatorInvitationDto{}), nil)
		return c.Status(fiber.StatusUnprocessableEntity).JSON(rd)
	}

	db, err := middleware.GetDatabase(c)
	if err != nil {
		return apperror.New(fiber.StatusInternalServerError, "error", err.Error(), err, nil)
	}

	status, err := h.svc.SendInvitation(userDetails.ID, req.AggregatorID, db)
	if err != nil {
		return apperror.New(status, "error", err.Error(), err, nil)
	}

	rd := utility.BuildSuccessResponse(status, "Invitation sent successfully", nil)
	return c.Status(status).JSON(rd)
}

// @Summary List Sent Invitations
// @Description Fetch all sent invitations
// @Tags Business Aggregator Portal
// @Produce json
// @Security BearerAuth
// @Success 200 {object} BusinessInvitationListResponseDto "Invitations fetched successfully"
// @Failure 401 {object} entities.Response "Unauthorized"
// @Failure 500 {object} entities.Response "Internal server error"
// @Router /business/aggregators/invitations [get]
func (h *Handler) ListSentInvitations(c *fiber.Ctx) error {
	userDetails, err := middleware.GetUserDetails(c)
	if err != nil {
		return apperror.New(fiber.StatusUnauthorized, "error", "Unauthorized", err, nil)
	}

	if userDetails.IsAggregator {
		return apperror.New(fiber.StatusForbidden, "error", "Aggregator account cannot list sent invitations", nil, nil)
	}

	if userDetails.BusinessID == nil {
		return apperror.New(fiber.StatusForbidden, "error", "Business ID missing", nil, nil)
	}

	db, err := middleware.GetDatabase(c)
	if err != nil {
		return apperror.New(fiber.StatusInternalServerError, "error", err.Error(), err, nil)
	}

	log.Println("got here")
	invitations, err := h.svc.ListBusinessInvitations(userDetails.ID, db)
	if err != nil {
		return apperror.New(fiber.StatusInternalServerError, "error", err.Error(), err, nil)
	}

	rd := utility.BuildSuccessResponse(fiber.StatusOK, "Invitations fetched successfully", invitations)
	return c.Status(fiber.StatusOK).JSON(rd)
}

// @Summary Revoke Invitation
// @Description Revoke an invitation
// @Tags Business Aggregator Portal
// @Produce json
// @Security BearerAuth
// @Success 200 {object} entities.Response "Invitation revoked successfully"
// @Failure 401 {object} entities.Response "Unauthorized"
// @Failure 500 {object} entities.Response "Internal server error"
// @Router /business/aggregators/invitations/{id} [delete]
func (h *Handler) RevokeAggregatorInvitation(c *fiber.Ctx) error {
	userDetails, err := middleware.GetUserDetails(c)
	if err != nil {
		return apperror.New(fiber.StatusUnauthorized, "error", "Unauthorized", err, nil)
	}

	if userDetails.IsAggregator {
		return apperror.New(fiber.StatusForbidden, "error", "Aggregator account cannot revoke invitations", nil, nil)
	}

	if userDetails.BusinessID == nil {
		return apperror.New(fiber.StatusForbidden, "error", "Business ID missing", nil, nil)
	}

	invitationID := c.Params("id")
	if invitationID == "" {
		return apperror.New(fiber.StatusBadRequest, "error", "invitation id is required", nil, nil)
	}

	db, err := middleware.GetDatabase(c)
	if err != nil {
		return apperror.New(fiber.StatusInternalServerError, "error", err.Error(), err, nil)
	}

	status, err := h.svc.RevokeInvitation(invitationID, userDetails.ID, db)
	if err != nil {
		return apperror.New(status, "error", err.Error(), err, nil)
	}

	rd := utility.BuildSuccessResponse(status, "Invitation revoked successfully", nil)
	return c.Status(status).JSON(rd)
}

func mapToGeneric(invitations []BusinessInvitationDto) []AggregatorInvitationDto {
	result := make([]AggregatorInvitationDto, len(invitations))
	for i, v := range invitations {
		result[i] = AggregatorInvitationDto{
			ID:            v.ID,
			BusinessID:    v.AggregatorID,
			BusinessName:  v.AggregatorName,
			BusinessEmail: v.AggregatorEmail,
			Status:        v.Status,
			CreatedAt:     v.CreatedAt,
		}
	}
	return result
}

// @Summary Send Invitation By Email
// @Description Send an invitation to an aggregator by their email. If they do not exist, creates a profile for them.
// @Tags Business Aggregator Portal
// @Produce json
// @Security BearerAuth
// @Param data body SendAggregatorInvitationByEmailDto true "Send aggregator invitation by email request payload"
// @Success 200 {object} entities.Response "Invitation sent successfully"
// @Failure 401 {object} entities.Response "Unauthorized"
// @Failure 500 {object} entities.Response "Internal server error"
// @Router /business/aggregators/invite-by-email [post]
func (h *Handler) SendAggregatorInvitationByEmail(c *fiber.Ctx) error {
	userDetails, err := middleware.GetUserDetails(c)

	if err != nil {
		return apperror.New(fiber.StatusUnauthorized, "error", "Unauthorized", err, nil)
	}

	if userDetails.IsAggregator {
		return apperror.New(fiber.StatusForbidden, "error", "Aggregator account cannot send invitations", nil, nil)
	}

	var req SendAggregatorInvitationByEmailDto
	if err := c.BodyParser(&req); err != nil {
		return apperror.New(fiber.StatusBadRequest, "error", "Failed to parse request body", err, nil)
	}

	if err := h.Validator.Struct(&req); err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusUnprocessableEntity, "error", "Validation failed", utility.ValidationErrorsToJSON(err, SendAggregatorInvitationByEmailDto{}), nil)
		return c.Status(fiber.StatusUnprocessableEntity).JSON(rd)
	}

	db, err := middleware.GetDatabase(c)
	if err != nil {
		return apperror.New(fiber.StatusInternalServerError, "error", err.Error(), err, nil)
	}

	status, err := h.svc.SendInvitationByEmail(userDetails.ID, req.Email, db)
	if err != nil {
		return apperror.New(status, "error", err.Error(), err, nil)
	}

	rd := utility.BuildSuccessResponse(status, "Invitation sent successfully to email", nil)
	return c.Status(status).JSON(rd)
}

// CreateBusiness godoc
// @Summary Create a business under aggregator
// @Description Creates a new business and associates it with the calling aggregator
// @Tags Aggregator Portal
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param data body CreateBusinessDto true "Business details"
// @Success 201 {object} CreateBusinessResponseDto "Business created successfully"
// @Failure 400 {object} entities.Response "Bad request"
// @Failure 401 {object} entities.Response "Unauthorized"
// @Failure 500 {object} entities.Response "Internal server error"
// @Router /aggregator/businesses [post]
func (h *Handler) CreateBusiness(c *fiber.Ctx) error {
	userDetails, err := middleware.GetUserDetails(c)
	if err != nil {
		return apperror.New(fiber.StatusUnauthorized, "error", "Unauthorized", err, nil)
	}

	var req CreateBusinessDto
	if err := c.BodyParser(&req); err != nil {
		return apperror.New(fiber.StatusBadRequest, "error", "Failed to parse request body", err, nil)
	}

	if err := h.Validator.Struct(&req); err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusUnprocessableEntity, "error", "Validation failed", utility.ValidationErrorsToJSON(err, CreateBusinessDto{}), nil)
		return c.Status(fiber.StatusUnprocessableEntity).JSON(rd)
	}

	db, err := middleware.GetDatabase(c)
	if err != nil {
		return apperror.New(fiber.StatusInternalServerError, "error", err.Error(), err, nil)
	}

	if err := h.svc.CreateBusiness(db, req, userDetails.ID); err != nil {
		return apperror.New(fiber.StatusBadRequest, "error", err.Error(), nil, nil)
	}

	rd := utility.BuildSuccessResponse(fiber.StatusCreated, "Business created successfully", nil)
	return c.Status(fiber.StatusCreated).JSON(rd)
}

// @Summary Update Business Setup
// @Description Upload crypto keys and/or set service ID and business ID for a managed business. At least one field must be provided.
// @Tags Aggregator Portal
// @Accept multipart/form-data
// @Produce json
// @Security BearerAuth
// @Param id path string true "Business ID"
// @Param file formData file false "Crypto keys document (optional)"
// @Param irn_public_key formData string false "IRN Public Key as string (optional)"
// @Param irn_certificate formData string false "IRN Certificate as string (optional)"
// @Param service_id formData string false "Service ID (optional)"
// @Param business_id formData string false "Business ID / FIRS identifier (optional)"
// @Success 200 {object} entities.Response "Business setup updated successfully"
// @Failure 400 {object} entities.Response "Bad request"
// @Failure 401 {object} entities.Response "Unauthorized"
// @Failure 404 {object} entities.Response "Business not found"
// @Failure 500 {object} entities.Response "Internal server error"
// @Router /aggregator/businesses/{id} [patch]
func (h *Handler) UpdateBusinessSetup(c *fiber.Ctx) error {
	userDetails, err := middleware.GetUserDetails(c)
	if err != nil {
		return apperror.New(fiber.StatusUnauthorized, "error", "Unauthorized", err, nil)
	}

	businessID := c.Params("id")
	if businessID == "" {
		return apperror.New(fiber.StatusBadRequest, "error", "business id is required", nil, nil)
	}

	serviceID := c.FormValue("service_id")
	fBusinessID := c.FormValue("business_id")
	irnPublicKey := c.FormValue("irn_public_key")
	irnCertificate := c.FormValue("irn_certificate")
	file, fileErr := c.FormFile("file")

	hasFile := fileErr == nil && file != nil
	hasServiceID := serviceID != ""
	hasBusinessID := fBusinessID != ""
	hasIRNPublicKey := irnPublicKey != ""
	hasIRNCertificate := irnCertificate != ""

	if !hasFile && !hasServiceID && !hasBusinessID && !hasIRNPublicKey && !hasIRNCertificate {
		return apperror.New(fiber.StatusBadRequest, "error", "at least one of file, service_id, business_id, irn_public_key, or irn_certificate must be provided", nil, nil)
	}

	db, err := middleware.GetDatabase(c)
	if err != nil {
		return apperror.New(fiber.StatusInternalServerError, "error", err.Error(), err, nil)
	}

	business, status, err := h.svc.GetBusinessDetail(userDetails.ID, businessID, db)
	if err != nil {
		return apperror.New(status, "error", err.Error(), err, nil)
	}

	activityDetails := "Updated setup for business " + business.CompanyName + ":"

	if hasFile {
		openedFile, err := file.Open()
		if err != nil {
			return apperror.New(fiber.StatusBadRequest, "error", "failed to open crypto keys file", err, nil)
		}
		defer openedFile.Close()

		fileContent, err := io.ReadAll(openedFile)
		if err != nil {
			return apperror.New(fiber.StatusBadRequest, "error", "failed to read crypto keys file", err, nil)
		}

		if err := h.businessSvc.SaveBusinessIRNSigningKeys(db, businessID, fileContent); err != nil {
			return apperror.New(fiber.StatusBadRequest, "error", err.Error(), nil, nil)
		}

		activityDetails += " crypto_keys=uploaded"
	}

	if hasServiceID || hasBusinessID || hasIRNPublicKey || hasIRNCertificate {
		var req AggregatorUpdateBusinessSetupDto
		if hasServiceID {
			req.ServiceID = &serviceID
			activityDetails += " service_id=" + serviceID
		}
		if hasBusinessID {
			req.BusinessID = &fBusinessID
			activityDetails += " business_id=" + fBusinessID
		}
		if hasIRNPublicKey {
			req.IRNPublicKey = &irnPublicKey
			activityDetails += " irn_public_key=provided"
		}
		if hasIRNCertificate {
			req.IRNCertificate = &irnCertificate
			activityDetails += " irn_certificate=provided"
		}

		if err := h.svc.UpdateBusinessSetup(db, businessID, req, userDetails.ID); err != nil {
			return apperror.New(fiber.StatusInternalServerError, "error", err.Error(), err, nil)
		}
	}

	h.svc.LogActivity(db, userDetails.ID, businessID, entities.ActivityBusinessSetupUpdate, activityDetails)

	rd := utility.BuildSuccessResponse(fiber.StatusOK, "Business setup updated successfully", nil)
	return c.Status(fiber.StatusOK).JSON(rd)
}

// @Summary List Invitations
// @Description Fetch all pending invitations for the aggregator
// @Tags Aggregator Portal
// @Produce json
// @Security BearerAuth
// @Success 200 {object} AggregatorInvitationListResponseDto "Invitations fetched successfully"
// @Failure 401 {object} entities.Response "Unauthorized"
// @Failure 500 {object} entities.Response "Internal server error"
// @Router /aggregator/invitations [get]
func (h *Handler) ListInvitations(c *fiber.Ctx) error {
	userDetails, err := middleware.GetUserDetails(c)
	if err != nil {
		return apperror.New(fiber.StatusUnauthorized, "error", "Unauthorized", err, nil)
	}

	db, err := middleware.GetDatabase(c)
	if err != nil {
		return apperror.New(fiber.StatusInternalServerError, "error", err.Error(), err, nil)
	}

	invitations, err := h.svc.ListAggregatorInvitations(userDetails.ID, db)
	if err != nil {
		return apperror.New(fiber.StatusBadRequest, "error", err.Error(), err, nil)
	}

	rd := utility.BuildSuccessResponse(fiber.StatusOK, "Invitations fetched successfully", invitations)
	return c.Status(fiber.StatusOK).JSON(rd)
}

// @Summary Respond to Invitation
// @Description Accept or reject an invitation
// @Tags Aggregator Portal
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param data body RespondToInvitationDto true "Invitation Response payload"
// @Success 200 {object} entities.Response "Responded to invitation successfully"
// @Failure 400 {object} entities.Response "Bad request, validation failed"
// @Failure 401 {object} entities.Response "Unauthorized"
// @Failure 422 {object} entities.Response "Unprocessable entity"
// @Failure 500 {object} entities.Response "Internal server error"
// @Router /aggregator/invitations/respond [post]
func (h *Handler) RespondToInvitation(c *fiber.Ctx) error {
	userDetails, err := middleware.GetUserDetails(c)
	if err != nil {
		return apperror.New(fiber.StatusUnauthorized, "error", "Unauthorized", err, nil)
	}

	var req RespondToInvitationDto
	if err := c.BodyParser(&req); err != nil {
		return apperror.New(fiber.StatusBadRequest, "error", "Failed to parse request body", err, nil)
	}

	if err := h.Validator.Struct(&req); err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusUnprocessableEntity, "error", "Validation failed", utility.ValidationErrorsToJSON(err, RespondToInvitationDto{}), nil)
		return c.Status(fiber.StatusUnprocessableEntity).JSON(rd)
	}

	db, err := middleware.GetDatabase(c)
	if err != nil {
		return apperror.New(fiber.StatusInternalServerError, "error", err.Error(), err, nil)
	}

	status, err := h.svc.RespondToInvitation(req.InvitationID, userDetails.ID, req.Accept, db)
	if err != nil {
		return apperror.New(status, "error", err.Error(), err, nil)
	}

	rd := utility.BuildSuccessResponse(status, "Responded to invitation successfully", nil)
	return c.Status(status).JSON(rd)
}

// @Summary Dashboard Stats
// @Description Fetch high level stats for aggregator dashboard
// @Tags Aggregator Portal
// @Produce json
// @Security BearerAuth
// @Success 200 {object} AggregatorDashboardResponseDto "Dashboard stats fetched successfully"
// @Failure 401 {object} entities.Response "Unauthorized"
// @Failure 500 {object} entities.Response "Internal server error"
// @Router /aggregator/dashboard [get]
func (h *Handler) Dashboard(c *fiber.Ctx) error {
	userDetails, err := middleware.GetUserDetails(c)
	if err != nil {
		return apperror.New(fiber.StatusUnauthorized, "error", "Unauthorized", err, nil)
	}

	db, err := middleware.GetDatabase(c)
	if err != nil {
		return apperror.New(fiber.StatusInternalServerError, "error", err.Error(), err, nil)
	}

	stats, err := h.svc.GetDashboard(userDetails.ID, db)
	if err != nil {
		return apperror.New(fiber.StatusInternalServerError, "error", err.Error(), err, nil)
	}

	rd := utility.BuildSuccessResponse(fiber.StatusOK, "Dashboard stats fetched successfully", stats)
	return c.Status(fiber.StatusOK).JSON(rd)
}

// @Summary List Businesses
// @Description List accepted businesses for an aggregator
// @Tags Aggregator Portal
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number"
// @Param size query int false "Page size"
// @Param search query string false "Search term"
// @Success 200 {object} AggregatorBusinessListResponseDto "Businesses fetched successfully"
// @Failure 401 {object} entities.Response "Unauthorized"
// @Failure 500 {object} entities.Response "Internal server error"
// @Router /aggregator/businesses [get]
func (h *Handler) ListBusinesses(c *fiber.Ctx) error {
	userDetails, err := middleware.GetUserDetails(c)
	if err != nil {
		return apperror.New(fiber.StatusUnauthorized, "error", "Unauthorized", err, nil)
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
	search := c.Query("search", "")

	db, err := middleware.GetDatabase(c)
	if err != nil {
		return apperror.New(fiber.StatusInternalServerError, "error", err.Error(), err, nil)
	}

	businesses, pagination, err := h.svc.ListBusinesses(userDetails.ID, query.Page, query.Size, search, db)
	if err != nil {
		return apperror.New(fiber.StatusInternalServerError, "error", err.Error(), err, nil)
	}

	rd := utility.BuildSuccessResponse(fiber.StatusOK, "Businesses fetched successfully", businesses, pagination)
	return c.Status(fiber.StatusOK).JSON(rd)
}

// @Summary Get Business Detail
// @Description Get details for a single accepted business
// @Tags Aggregator Portal
// @Produce json
// @Security BearerAuth
// @Param id path string true "Business ID"
// @Success 200 {object} AggregatorBusinessFullDetailDto "Business fetched successfully"
// @Failure 400 {object} entities.Response "Bad request"
// @Failure 401 {object} entities.Response "Unauthorized"
// @Failure 500 {object} entities.Response "Internal server error"
// @Router /aggregator/businesses/{id} [get]
func (h *Handler) GetBusinessDetail(c *fiber.Ctx) error {
	userDetails, err := middleware.GetUserDetails(c)
	if err != nil {
		return apperror.New(fiber.StatusUnauthorized, "error", "Unauthorized", err, nil)
	}

	businessID := c.Params("id")
	if businessID == "" {
		return apperror.New(fiber.StatusBadRequest, "error", "business id is required", nil, nil)
	}

	db, err := middleware.GetDatabase(c)
	if err != nil {
		return apperror.New(fiber.StatusInternalServerError, "error", err.Error(), err, nil)
	}

	business, status, err := h.svc.GetBusinessDetail(userDetails.ID, businessID, db)
	if err != nil {
		return apperror.New(status, "error", err.Error(), err, nil)
	}

	rd := utility.BuildSuccessResponse(status, "Business fetched successfully", business)
	return c.Status(status).JSON(rd)
}

// @Summary Remove Business
// @Description Remove an accepted business from the aggregator
// @Tags Aggregator Portal
// @Produce json
// @Security BearerAuth
// @Param id path string true "Business ID"
// @Success 200 {object} entities.Response "Business removed successfully"
// @Failure 400 {object} entities.Response "Bad request"
// @Failure 401 {object} entities.Response "Unauthorized"
// @Failure 500 {object} entities.Response "Internal server error"
// @Router /aggregator/businesses/{id} [delete]
func (h *Handler) RemoveBusiness(c *fiber.Ctx) error {
	userDetails, err := middleware.GetUserDetails(c)
	if err != nil {
		return apperror.New(fiber.StatusUnauthorized, "error", "Unauthorized", err, nil)
	}

	businessID := c.Params("id")
	if businessID == "" {
		return apperror.New(fiber.StatusBadRequest, "error", "business id is required", nil, nil)
	}

	db, err := middleware.GetDatabase(c)
	if err != nil {
		return apperror.New(fiber.StatusInternalServerError, "error", err.Error(), err, nil)
	}

	status, err := h.svc.RemoveBusiness(userDetails.ID, businessID, db)
	if err != nil {
		return apperror.New(status, "error", err.Error(), err, nil)
	}

	rd := utility.BuildSuccessResponse(status, "Business removed successfully", nil)
	return c.Status(status).JSON(rd)
}

// @Summary List All Invoices
// @Description Gets all invoices across all businesses uploaded by this aggregator
// @Tags Aggregator Portal
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number"
// @Param size query int false "Page size"
// @Success 200 {object} AggregatorInvoiceListResponseDto "Invoices fetched successfully"
// @Failure 401 {object} entities.Response "Unauthorized"
// @Failure 500 {object} entities.Response "Internal server error"
// @Router /aggregator/invoices [get]
func (h *Handler) ListAllInvoices(c *fiber.Ctx) error {
	userDetails, err := middleware.GetUserDetails(c)
	if err != nil {
		return apperror.New(fiber.StatusUnauthorized, "error", "Unauthorized", err, nil)
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

	invoices, pagination, err := h.svc.ListAllInvoices(userDetails.ID, query.Page, query.Size, db)
	if err != nil {
		return apperror.New(fiber.StatusInternalServerError, "error", err.Error(), err, nil)
	}

	rd := utility.BuildSuccessResponse(fiber.StatusOK, "Invoices fetched successfully", invoices, pagination)
	return c.Status(fiber.StatusOK).JSON(rd)
}

// @Summary List Business Invoices
// @Description Gets invoices uploaded by aggregator for a specific business
// @Tags Aggregator Portal
// @Produce json
// @Security BearerAuth
// @Param id path string true "Business ID"
// @Param page query int false "Page number"
// @Param size query int false "Page size"
// @Success 200 {object} AggregatorInvoiceListResponseDto "Invoices fetched successfully"
// @Failure 400 {object} entities.Response "Bad request"
// @Failure 401 {object} entities.Response "Unauthorized"
// @Failure 500 {object} entities.Response "Internal server error"
// @Router /aggregator/invoices/{id} [get]
func (h *Handler) ListBusinessInvoices(c *fiber.Ctx) error {
	userDetails, err := middleware.GetUserDetails(c)
	if err != nil {
		return apperror.New(fiber.StatusUnauthorized, "error", "Unauthorized", err, nil)
	}

	businessID := c.Params("id")
	if businessID == "" {
		return apperror.New(fiber.StatusBadRequest, "error", "business id is required", nil, nil)
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

	invoices, pagination, err := h.svc.ListInvoicesByBusiness(userDetails.ID, businessID, query.Page, query.Size, db)
	if err != nil {
		return apperror.New(fiber.StatusInternalServerError, "error", err.Error(), err, nil)
	}

	rd := utility.BuildSuccessResponse(fiber.StatusOK, "Invoices fetched successfully", invoices, pagination)
	return c.Status(fiber.StatusOK).JSON(rd)
}

// @Summary List All Bulk Uploads
// @Description Gets all bulk uploads across all businesses uploaded by this aggregator
// @Tags Aggregator Portal
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number"
// @Param size query int false "Page size"
// @Success 200 {object} AggregatorBulkUploadListResponseDto "Bulk uploads fetched successfully"
// @Failure 401 {object} entities.Response "Unauthorized"
// @Failure 500 {object} entities.Response "Internal server error"
// @Router /aggregator/bulk-uploads [get]
func (h *Handler) ListAllBulkUploads(c *fiber.Ctx) error {
	userDetails, err := middleware.GetUserDetails(c)
	if err != nil {
		return apperror.New(fiber.StatusUnauthorized, "error", "Unauthorized", err, nil)
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

	uploads, pagination, err := h.svc.ListAllBulkUploads(userDetails.ID, query.Page, query.Size, db)
	if err != nil {
		return apperror.New(fiber.StatusInternalServerError, "error", err.Error(), err, nil)
	}

	rd := utility.BuildSuccessResponse(fiber.StatusOK, "Bulk uploads fetched successfully", uploads, pagination)
	return c.Status(fiber.StatusOK).JSON(rd)
}

// @Summary List Bulk Uploads by Business
// @Description Gets bulk uploads uploaded by aggregator for a specific business
// @Tags Aggregator Portal
// @Produce json
// @Security BearerAuth
// @Param business_id path string true "Business ID"
// @Param page query int false "Page number"
// @Param size query int false "Page size"
// @Success 200 {object} GetBulkUploadLogsResponseDto "Bulk uploads fetched successfully"
// @Failure 400 {object} entities.Response "Bad request"
// @Failure 401 {object} entities.Response "Unauthorized"
// @Failure 500 {object} entities.Response "Internal server error"
// @Router /aggregator/bulk-uploads/{business_id} [get]
func (h *Handler) ListBulkUploadLogs(c *fiber.Ctx) error {
	userDetails, err := middleware.GetUserDetails(c)
	if err != nil {
		return apperror.New(fiber.StatusUnauthorized, "error", "Unauthorized", err, nil)
	}

	businessID := c.Params("business_id")
	if businessID == "" {
		return apperror.New(fiber.StatusBadRequest, "error", "business id is required", nil, nil)
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

	uploads, pagination, err := h.svc.ListBulkUploadsByBusiness(userDetails.ID, businessID, query.Page, query.Size, db)
	if err != nil {
		return apperror.New(fiber.StatusInternalServerError, "error", err.Error(), err, nil)
	}

	rd := utility.BuildSuccessResponse(fiber.StatusOK, "Bulk uploads fetched successfully", uploads, pagination)
	return c.Status(fiber.StatusOK).JSON(rd)
}

// @Summary Get failed invoices from a bulk upload
// @Description Gets the failed invoices recorded for a specific bulk upload uploaded by this aggregator
// @Tags Aggregator Portal
// @Produce json
// @Security BearerAuth
// @Param bulk_id path string true "Bulk upload ID"
// @Success 200 {object} GetBulkUploadFailedInvoicesResponseDto "Bulk upload failed invoices fetched successfully"
// @Failure 400 {object} entities.Response "Bad request"
// @Failure 401 {object} entities.Response "Unauthorized"
// @Failure 404 {object} entities.Response "Bulk upload not found"
// @Failure 500 {object} entities.Response "Internal server error"
// @Router /aggregator/bulk-uploads/{bulk_id}/failed [get]
func (h *Handler) GetBulkUploadFailedInvoices(c *fiber.Ctx) error {
	userDetails, err := middleware.GetUserDetails(c)
	if err != nil {
		return apperror.New(fiber.StatusUnauthorized, "error", "Unauthorized", err, nil)
	}

	bulkUploadID := c.Params("bulk_id")
	if bulkUploadID == "" {
		return apperror.New(fiber.StatusBadRequest, "error", "bulk upload id is required", nil, nil)
	}

	db, err := middleware.GetDatabase(c)
	if err != nil {
		return apperror.New(fiber.StatusInternalServerError, "error", err.Error(), err, nil)
	}
	failedInvoices, status, err := h.svc.GetBulkUploadFailedInvoices(userDetails.ID, bulkUploadID, db)
	if err != nil {
		return apperror.New(status, "error", err.Error(), err, nil)
	}

	rd := utility.BuildSuccessResponse(status, "Bulk upload failed invoices fetched successfully", failedInvoices)
	return c.Status(status).JSON(rd)
}

// @Summary Download failed invoices from a bulk upload
// @Description Download the failed invoices recorded for a specific bulk upload uploaded by this aggregator as csv or excel
// @Tags Aggregator Portal
// @Produce text/csv,application/vnd.openxmlformats-officedocument.spreadsheetml.sheet
// @Security BearerAuth
// @Param bulk_id path string true "Bulk upload ID"
// @Param format query string false "Export format" Enums(csv,excel,xlsx)
// @Success 200 {file} file "Bulk upload failed invoices file"
// @Failure 400 {object} entities.Response "Bad request"
// @Failure 401 {object} entities.Response "Unauthorized"
// @Failure 404 {object} entities.Response "Bulk upload not found"
// @Failure 500 {object} entities.Response "Internal server error"
// @Router /aggregator/bulk-uploads/{bulk_id}/failed/download [get]
func (h *Handler) DownloadBulkUploadFailedInvoices(c *fiber.Ctx) error {
	userDetails, err := middleware.GetUserDetails(c)
	if err != nil {
		return apperror.New(fiber.StatusUnauthorized, "error", "Unauthorized", err, nil)
	}

	bulkUploadID := c.Params("bulk_id")
	if bulkUploadID == "" {
		return apperror.New(fiber.StatusBadRequest, "error", "bulk upload id is required", nil, nil)
	}

	db, err := middleware.GetDatabase(c)
	if err != nil {
		return apperror.New(fiber.StatusInternalServerError, "error", err.Error(), err, nil)
	}
	failedInvoices, status, err := h.svc.GetBulkUploadFailedInvoices(userDetails.ID, bulkUploadID, db)
	if err != nil {
		return apperror.New(status, "error", err.Error(), err, nil)
	}

	fileData, contentType, extension, err := h.bulkUploadSvc.ExportBulkUploadFailedInvoices(failedInvoices, c.Query("format", "csv"))
	if err != nil {
		return apperror.New(fiber.StatusBadRequest, "error", err.Error(), err, nil)
	}

	c.Set("Content-Type", contentType)
	c.Set("Content-Disposition", "attachment; filename=\"bulk_upload_failed_invoices_"+bulkUploadID+"."+extension+"\"")
	return c.Status(fiber.StatusOK).Send(fileData)
}

// @Summary Activity Log
// @Description Fetch the activity logs sequence for the aggregator
// @Tags Aggregator Portal
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number"
// @Param size query int false "Page size"
// @Success 200 {object} AggregatorActivityLogListResponseDto "Activity logs fetched successfully"
// @Failure 401 {object} entities.Response "Unauthorized"
// @Failure 500 {object} entities.Response "Internal server error"
// @Router /aggregator/activity-log [get]
func (h *Handler) ActivityLog(c *fiber.Ctx) error {
	userDetails, err := middleware.GetUserDetails(c)
	if err != nil {
		return apperror.New(fiber.StatusUnauthorized, "error", "Unauthorized", err, nil)
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

	logs, pagination, err := h.svc.GetActivityLog(userDetails.ID, query.Page, query.Size, db)
	if err != nil {
		return apperror.New(fiber.StatusInternalServerError, "error", err.Error(), err, nil)
	}

	rd := utility.BuildSuccessResponse(fiber.StatusOK, "Activity logs fetched successfully", logs, pagination)
	return c.Status(fiber.StatusOK).JSON(rd)
}

// @Summary Upload Invoice
// @Description Upload a single invoice for a managed business
// @Tags Aggregator Portal
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Business ID"
// @Param data body firs_models.UploadInvoiceRequestDto true "Invoice payload"
// @Success 201 {object} AggregatorInvoiceUploadResponseDto "Invoice generated successfully"
// @Failure 400 {object} entities.Response "Bad request"
// @Failure 401 {object} entities.Response "Unauthorized"
// @Failure 422 {object} entities.Response "Unprocessable entity"
// @Failure 500 {object} entities.Response "Internal server error"
// @Router /aggregator/invoices/{id} [post]
func (h *Handler) UploadInvoice(c *fiber.Ctx) error {
	client := c.Get("client")
	userDetails, err := middleware.GetUserDetails(c)
	if err != nil {
		return apperror.New(fiber.StatusUnauthorized, "error", "Unauthorized", err, nil)
	}

	businessID := c.Params("id")
	if businessID == "" {
		return apperror.New(fiber.StatusBadRequest, "error", "business id is required", nil, nil)
	}

	var req UploadInvoiceRequestDto
	if err := c.BodyParser(&req); err != nil {
		return apperror.New(fiber.StatusBadRequest, "error", "Failed to parse request body", err, nil)
	}

	if err := h.Validator.Struct(&req); err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusUnprocessableEntity, "error", "Validation failed", utility.ValidationErrorsToJSON(err, UploadInvoiceRequestDto{}), nil)
		return c.Status(fiber.StatusUnprocessableEntity).JSON(rd)
	}

	db, err := middleware.GetDatabase(c)
	if err != nil {
		return apperror.New(fiber.StatusInternalServerError, "error", err.Error(), err, nil)
	}

	_, status, err := h.svc.GetBusinessDetail(userDetails.ID, businessID, db)
	if err != nil {
		return apperror.New(status, "error", err.Error(), err, nil)
	}

	setup, err := h.businessSvc.ValidateInvoiceUploadSetup(db, businessID)
	if err != nil {
		return apperror.New(fiber.StatusBadRequest, "error", err.Error(), nil, nil)
	}

	if _, status, err = h.subSvc.RequireAggregatorBusinessSubscription(db, userDetails.ID, businessID); err != nil {
		return apperror.New(status, "error", err.Error(), err, nil)
	}

	invoiceSvc := h.invoiceSvc
	invoiceExists, _ := invoiceSvc.GetInvoiceByInvoiceNumber(db, req.InvoiceNumber, businessID)

	if invoiceExists != nil {
		blockedStatuses := map[string]bool{
			entities.StatusSignedInvoice: true,
			entities.StatusTransmitted:   true,
			entities.StatusConfirmed:     true,
		}
		if blockedStatuses[invoiceExists.CurrentStatus] {
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

	reservedSubscriptionID := ""
	if invoiceExists == nil {
	}

	var irnPayload InvoiceData
	if req.IRN == nil {
		IRNData, irnErr := invoiceSvc.IRNGeneration(db, businessID, req.InvoiceNumber, setup.ServiceID, req.BusinessID, userDetails.IsSandbox)
		if irnErr != nil {
			if reservedSubscriptionID != "" {
				_ = h.subSvc.ReleaseReservedInvoices(db, reservedSubscriptionID, 1)
			}
			return c.Status(fiber.StatusBadRequest).JSON(*irnErr)
		}
		irnPayload = *IRNData
		req.IRN = &irnPayload.IRN
	} else {
		if invoiceExists == nil {
			keys, err := h.businessSvc.ResolveBusinessIRNSigningKeys(db, businessID, userDetails.IsSandbox, nil)
			if err != nil {
				return apperror.New(fiber.StatusBadRequest, "error", err.Error(), err, nil)
			}

			signedIRN, err := invoiceSvc.SignIRN(*req.IRN, keys)
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
			qrBmp, _ := utility.Base64PNGToBMP(invoiceExists.QrCode)
			qrCode := invoiceExists.QrCode
			if qrCode == "" {
				qrCode = invoiceExists.EncryptedIRN
			}
			irnPayload = InvoiceData{
				InvoiceNumber: req.InvoiceNumber,
				IRN:           *req.IRN,
				QRCode:        qrCode,
				QRCode2:       invoiceExists.EncryptedIRN,
				QRCodeBMP:     qrBmp,
			}
		}
	}

	qrCodeBMPURL := ""
	if irnPayload.QRCodeBMP != "" {
		qrCodeBMPURL, err = cloudinary.UploadBMPBase64(irnPayload.QRCodeBMP, utility.GenerateUUID())
		if err != nil {
			return apperror.New(fiber.StatusBadGateway, "error", "failed to upload qr code image", err, nil)
		}
	}

	createdInvoice, _, err, isInvoiceSigned := invoiceSvc.CreateInvoice(db, req, req.InvoiceNumber, businessID, irnPayload.QRCode, qrCodeBMPURL, irnPayload.QRCode2, invoiceExists, userDetails.IsSandbox, &userDetails.ID, client)
	if reservedSubscriptionID != "" && createdInvoice == nil {
		_ = h.subSvc.ReleaseReservedInvoices(db, reservedSubscriptionID, 1)
	}

	response := map[string]interface{}{}
	if createdInvoice != nil {
		response["metadata"] = createdInvoice.StatusHistory
	}

	if isInvoiceSigned {
		response["data"] = map[string]interface{}{
			"id":              createdInvoice.ID,
			"invoice_number":  irnPayload.InvoiceNumber,
			"irn":             irnPayload.IRN,
			"qr_code":         createdInvoice.QrCode,
			"qr_code_2":       createdInvoice.EncryptedIRN,
			"qr_code_bmp_url": qrCodeBMPURL,
		}
		rd := utility.BuildSuccessResponse(fiber.StatusCreated, "Invoice generated successfully", response)
		return c.Status(fiber.StatusCreated).JSON(rd)
	}

	return apperror.New(fiber.StatusCreated, "error", "failed to complete irn and invoice signing", response, nil)
}

// @Summary Bulk Upload Initializer
// @Description Bulk invoices upload for a managed business
// @Tags Aggregator Portal
// @Accept multipart/form-data
// @Produce json
// @Security BearerAuth
// @Param id path string true "Business ID"
// @Param file formData file true "Invoice JSON file"
// @Success 201 {object} entities.Response "Invoice uploaded successfully"
// @Failure 400 {object} entities.Response "Bad request"
// @Failure 401 {object} entities.Response "Unauthorized"
// @Failure 500 {object} entities.Response "Internal server error"
// @Router /aggregator/bulk-uploads/{id} [post]
func (h *Handler) BulkUpload(c *fiber.Ctx) error {
	userDetails, err := middleware.GetUserDetails(c)
	if err != nil {
		return apperror.New(fiber.StatusUnauthorized, "error", "Unauthorized", err, nil)
	}

	businessID := c.Params("id")
	if businessID == "" {
		return apperror.New(fiber.StatusBadRequest, "error", "business id is required", nil, nil)
	}

	db, err := middleware.GetDatabase(c)
	if err != nil {
		return apperror.New(fiber.StatusInternalServerError, "error", err.Error(), err, nil)
	}

	_, status, err := h.svc.GetBusinessDetail(userDetails.ID, businessID, db)
	if err != nil {
		return apperror.New(status, "error", err.Error(), err, nil)
	}

	setup, err := h.businessSvc.ValidateInvoiceUploadSetup(db, businessID)
	if err != nil {
		return apperror.New(fiber.StatusBadRequest, "error", err.Error(), nil, nil)
	}

	subscriptionRecord, status, err := h.subSvc.RequireAggregatorBusinessSubscription(db, userDetails.ID, businessID)
	if err != nil {
		return apperror.New(status, "error", err.Error(), err, nil)
	}
	if subscriptionRecord.RemainingInvoices <= 0 {
		return apperror.New(fiber.StatusForbidden, "error", "subscription invoice limit exhausted for this business", nil, nil)
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
	fileURL, fileKey, err := s3.UploadFileToS3(ctx, fileContent, file)
	if err != nil {
		log.Println("S3 upload failed:", err)
		return c.Status(500).JSON(fiber.Map{"error": "upload failed"})
	}

	bulkID, err := h.bulkUploadSvc.AddBulkUploadLog(db, fileURL, fileKey, businessID, &userDetails.ID)
	if err != nil {
		return apperror.New(fiber.StatusInternalServerError, "error", "failed to log bulk upload", nil, nil)
	}

	err = producer.NewProducer().EnqueueTask(workers.BulkUploadTask, workers.BulkUploadInput{
		BulkID:       bulkID,
		ID:           businessID,
		FileKey:      fileKey,
		ServiceID:    setup.ServiceID,
		BusinessID:   businessID,
		IsSandbox:    userDetails.IsSandbox,
		AggregatorID: &userDetails.ID,
	})
	if err != nil {
		return apperror.New(fiber.StatusInternalServerError, "error", "failed to enqueue bulk upload task", nil, nil)
	}

	rd := utility.BuildSuccessResponse(fiber.StatusCreated, "Invoice uploaded successfully", fileURL)
	return c.Status(fiber.StatusCreated).JSON(rd)
}

// @Summary List All Transactions
// @Description Gets all transaction history for the aggregator
// @Tags Aggregator Portal
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number"
// @Param size query int false "Page size"
// @Success 200 {object} AggregatorTransactionListResponseDto "Transactions fetched successfully"
// @Failure 401 {object} entities.Response "Unauthorized"
// @Failure 500 {object} entities.Response "Internal server error"
// @Router /aggregator/transactions [get]
func (h *Handler) ListAllTransactions(c *fiber.Ctx) error {
	userDetails, err := middleware.GetUserDetails(c)
	if err != nil {
		return apperror.New(fiber.StatusUnauthorized, "error", "Unauthorized", err, nil)
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

	transactions, pagination, err := h.svc.ListAllTransactions(userDetails.ID, query.Page, query.Size, db)
	if err != nil {
		return apperror.New(fiber.StatusInternalServerError, "error", err.Error(), err, nil)
	}

	rd := utility.BuildSuccessResponse(fiber.StatusOK, "Transactions fetched successfully", transactions, pagination)
	return c.Status(fiber.StatusOK).JSON(rd)
}

// GetInvoiceStats godoc
// @Summary Get invoice statistics for aggregator
// @Description Returns overall statistics for invoices including total, partial, successful, and failed for all businesses under the aggregator.
// @Tags Aggregator Portal
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} GetInvoiceStatsResponseDto "Invoice statistics fetched successfully"
// @Failure 401 {object} entities.Response "Unauthorized"
// @Router /aggregator/stats [get]
func (h *Handler) GetInvoiceStats(c *fiber.Ctx) error {
	userDetails, err := middleware.GetUserDetails(c)
	if err != nil {
		return apperror.New(fiber.StatusUnauthorized, "error", "Unauthorized", err, nil)
	}

	db, err := middleware.GetDatabase(c)
	if err != nil {
		return apperror.New(fiber.StatusInternalServerError, "error", err.Error(), err, nil)
	}

	stats, err := h.invoiceSvc.GetInvoiceStats(db, nil, &userDetails.ID)
	if err != nil {
		return apperror.New(fiber.StatusInternalServerError, "error", "failed to retrieve invoice stats", err, nil)
	}

	rd := utility.BuildSuccessResponse(fiber.StatusOK, "Invoice statistics fetched successfully", stats)
	return c.Status(fiber.StatusOK).JSON(rd)
}

// GetBusinessInvoiceStats godoc
// @Summary Get invoice statistics for a specific business under an aggregator
// @Description Returns overall statistics for invoices including total, partial, successful, and failed for a specific business managed by the aggregator.
// @Tags Aggregator Portal
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Business ID" format(uuid)
// @Success 200 {object} GetInvoiceStatsResponseDto "Invoice statistics fetched successfully"
// @Failure 400 {object} entities.Response "Bad request"
// @Failure 401 {object} entities.Response "Unauthorized"
// @Router /aggregator/businesses/{id}/stats [get]
func (h *Handler) GetBusinessInvoiceStats(c *fiber.Ctx) error {
	userDetails, err := middleware.GetUserDetails(c)
	if err != nil {
		return apperror.New(fiber.StatusUnauthorized, "error", "Unauthorized", err, nil)
	}

	businessID := c.Params("id")
	if businessID == "" {
		return apperror.New(fiber.StatusBadRequest, "error", "business id is required", nil, nil)
	}

	db, err := middleware.GetDatabase(c)
	if err != nil {
		return apperror.New(fiber.StatusInternalServerError, "error", err.Error(), err, nil)
	}

	stats, err := h.invoiceSvc.GetInvoiceStats(db, &businessID, &userDetails.ID)
	if err != nil {
		return apperror.New(fiber.StatusInternalServerError, "error", "failed to retrieve invoice stats", err, nil)
	}

	rd := utility.BuildSuccessResponse(fiber.StatusOK, "Business invoice statistics fetched successfully", stats)
	return c.Status(fiber.StatusOK).JSON(rd)
}

// GetInvoiceDetail godoc
// @Summary Get specific invoice details for aggregator
// @Description Fetches the full details of a specific invoice uploaded by the aggregator
// @Tags Aggregator Portal
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param invoice_id path string true "Invoice ID"
// @Success 200 {object} AggregatorGetInvoiceResponseDto "Invoice details fetched successfully"
// @Failure 400 {object} entities.Response "Bad request"
// @Failure 401 {object} entities.Response "Unauthorized"
// @Failure 404 {object} entities.Response "Not found"
// @Failure 500 {object} entities.Response "Internal server error"
// @Router /aggregator/invoices/single/{invoice_id} [get]
func (h *Handler) GetInvoiceDetail(c *fiber.Ctx) error {
	userDetails, err := middleware.GetUserDetails(c)
	if err != nil {
		return apperror.New(fiber.StatusUnauthorized, "error", "Unauthorized", err, nil)
	}

	invoiceID := c.Params("invoice_id")
	if invoiceID == "" {
		return apperror.New(fiber.StatusBadRequest, "error", "invoice_id is required", nil, nil)
	}

	db, err := middleware.GetDatabase(c)
	if err != nil {
		return apperror.New(fiber.StatusInternalServerError, "error", err.Error(), err, nil)
	}

	invoice, err := h.svc.GetInvoiceDetail(userDetails.ID, invoiceID, db)
	if err != nil {
		return apperror.New(fiber.StatusNotFound, "error", err.Error(), nil, nil)
	}

	rd := utility.BuildSuccessResponse(fiber.StatusOK, "Invoice details fetched successfully", invoice)
	return c.Status(fiber.StatusOK).JSON(rd)
}
