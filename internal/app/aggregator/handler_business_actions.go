package aggregator

import (
	"einvoice-access-point/internal/apperror"
	"einvoice-access-point/internal/data/entities"
	"einvoice-access-point/internal/middleware"
	"einvoice-access-point/internal/utility"
	"log"

	"github.com/gofiber/fiber/v2"
)

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

	// build dummy pagination struct since available method uses datah.PaginationResponse
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
	// Simple map for uniform response struct
	// This is a little hacky but returns a consistent structure for the frontend
	result := make([]AggregatorInvitationDto, len(invitations))
	for i, v := range invitations {
		result[i] = AggregatorInvitationDto{
			ID:            v.ID,
			BusinessID:    v.AggregatorID,    // putting aggregator id here for frontend convenience list
			BusinessName:  v.AggregatorName,  // mapping name
			BusinessEmail: v.AggregatorEmail, // mapping email
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
