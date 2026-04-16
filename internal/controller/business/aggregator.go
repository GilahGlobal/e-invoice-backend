package business

import (
	"einvoice-access-point/internal/dtos"
	aggregatorSvc "einvoice-access-point/internal/services/aggregator"
	"einvoice-access-point/pkg/middleware"
	"einvoice-access-point/pkg/models"
	"einvoice-access-point/pkg/utility"

	"github.com/gofiber/fiber/v2"
)

// @Summary List Available Aggregators
// @Description Fetch all available aggregators
// @Tags Business Aggregator Portal
// @Produce json
// @Security BearerAuth
// @Success 200 {object} dtos.AggregatorInvitationListResponseDto "Aggregators fetched successfully"
// @Failure 401 {object} models.Response "Unauthorized"
// @Failure 500 {object} models.Response "Internal server error"
// @Router /business/aggregators [get]
func (base *Controller) ListAvailableAggregators(c *fiber.Ctx) error {
	userDetails, err := middleware.GetUserDetails(c)
	if err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusUnauthorized, "error", "Unauthorized", err, nil)
		return c.Status(fiber.StatusUnauthorized).JSON(rd)
	}

	if userDetails.IsAggregator {
		rd := utility.BuildErrorResponse(fiber.StatusForbidden, "error", "Aggregator account cannot view other aggregators", nil, nil)
		return c.Status(fiber.StatusForbidden).JSON(rd)
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

	db := middleware.GetDatabaseInstance(userDetails.IsSandbox, base.Db, base.TestDb)

	aggregators, total, err := aggregatorSvc.ListAvailableAggregators(search, query.Page, query.Size, db)
	if err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusInternalServerError, "error", err.Error(), err, nil)
		return c.Status(fiber.StatusInternalServerError).JSON(rd)
	}

	// build dummy pagination struct since available method uses database.PaginationResponse
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
// @Success 200 {object} dtos.BaseResponseDto "Invitation sent successfully"
// @Failure 401 {object} models.Response "Unauthorized"
// @Failure 500 {object} models.Response "Internal server error"
// @Router /business/aggregators/invite [post]
func (base *Controller) SendAggregatorInvitation(c *fiber.Ctx) error {
	userDetails, err := middleware.GetUserDetails(c)

	if err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusUnauthorized, "error", "Unauthorized", err, nil)
		return c.Status(fiber.StatusUnauthorized).JSON(rd)
	}

	if userDetails.IsAggregator {
		rd := utility.BuildErrorResponse(fiber.StatusForbidden, "error", "Aggregator account cannot send invitations", nil, nil)
		return c.Status(fiber.StatusForbidden).JSON(rd)
	}

	if userDetails.BusinessID == nil {
		rd := utility.BuildErrorResponse(fiber.StatusForbidden, "error", "Business ID missing", nil, nil)
		return c.Status(fiber.StatusForbidden).JSON(rd)
	}

	var req dtos.SendAggregatorInvitationDto
	if err := c.BodyParser(&req); err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusBadRequest, "error", "Failed to parse request body", err, nil)
		return c.Status(fiber.StatusBadRequest).JSON(rd)
	}

	if err := base.Validator.Struct(&req); err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusUnprocessableEntity, "error", "Validation failed", utility.ValidationErrorsToJSON(err, dtos.SendAggregatorInvitationDto{}), nil)
		return c.Status(fiber.StatusUnprocessableEntity).JSON(rd)
	}

	db := middleware.GetDatabaseInstance(userDetails.IsSandbox, base.Db, base.TestDb)

	status, err := aggregatorSvc.SendInvitation(userDetails.ID, req.AggregatorID, db)
	if err != nil {
		rd := utility.BuildErrorResponse(status, "error", err.Error(), err, nil)
		return c.Status(status).JSON(rd)
	}

	rd := utility.BuildSuccessResponse(status, "Invitation sent successfully", nil)
	return c.Status(status).JSON(rd)
}

// @Summary List Sent Invitations
// @Description Fetch all sent invitations
// @Tags Business Aggregator Portal
// @Produce json
// @Security BearerAuth
// @Success 200 {object} dtos.BusinessInvitationListResponseDto "Invitations fetched successfully"
// @Failure 401 {object} models.Response "Unauthorized"
// @Failure 500 {object} models.Response "Internal server error"
// @Router /business/aggregators/invitations [get]
func (base *Controller) ListSentInvitations(c *fiber.Ctx) error {
	userDetails, err := middleware.GetUserDetails(c)
	if err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusUnauthorized, "error", "Unauthorized", err, nil)
		return c.Status(fiber.StatusUnauthorized).JSON(rd)
	}

	if userDetails.IsAggregator {
		rd := utility.BuildErrorResponse(fiber.StatusForbidden, "error", "Aggregator account cannot list sent invitations", nil, nil)
		return c.Status(fiber.StatusForbidden).JSON(rd)
	}

	if userDetails.BusinessID == nil {
		rd := utility.BuildErrorResponse(fiber.StatusForbidden, "error", "Business ID missing", nil, nil)
		return c.Status(fiber.StatusForbidden).JSON(rd)
	}

	db := middleware.GetDatabaseInstance(userDetails.IsSandbox, base.Db, base.TestDb)

	invitations, err := aggregatorSvc.ListBusinessInvitations(userDetails.ID, db)
	if err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusInternalServerError, "error", err.Error(), err, nil)
		return c.Status(fiber.StatusInternalServerError).JSON(rd)
	}

	rd := utility.BuildSuccessResponse(fiber.StatusOK, "Invitations fetched successfully", invitations)
	return c.Status(fiber.StatusOK).JSON(rd)
}

// @Summary Revoke Invitation
// @Description Revoke an invitation
// @Tags Business Aggregator Portal
// @Produce json
// @Security BearerAuth
// @Success 200 {object} dtos.BaseResponseDto "Invitation revoked successfully"
// @Failure 401 {object} models.Response "Unauthorized"
// @Failure 500 {object} models.Response "Internal server error"
// @Router /business/aggregators/invitations/{id} [delete]
func (base *Controller) RevokeAggregatorInvitation(c *fiber.Ctx) error {
	userDetails, err := middleware.GetUserDetails(c)
	if err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusUnauthorized, "error", "Unauthorized", err, nil)
		return c.Status(fiber.StatusUnauthorized).JSON(rd)
	}

	if userDetails.IsAggregator {
		rd := utility.BuildErrorResponse(fiber.StatusForbidden, "error", "Aggregator account cannot revoke invitations", nil, nil)
		return c.Status(fiber.StatusForbidden).JSON(rd)
	}

	if userDetails.BusinessID == nil {
		rd := utility.BuildErrorResponse(fiber.StatusForbidden, "error", "Business ID missing", nil, nil)
		return c.Status(fiber.StatusForbidden).JSON(rd)
	}

	invitationID := c.Params("id")
	if invitationID == "" {
		rd := utility.BuildErrorResponse(fiber.StatusBadRequest, "error", "invitation id is required", nil, nil)
		return c.Status(fiber.StatusBadRequest).JSON(rd)
	}

	db := middleware.GetDatabaseInstance(userDetails.IsSandbox, base.Db, base.TestDb)

	status, err := aggregatorSvc.RevokeInvitation(invitationID, userDetails.ID, db)
	if err != nil {
		rd := utility.BuildErrorResponse(status, "error", err.Error(), err, nil)
		return c.Status(status).JSON(rd)
	}

	rd := utility.BuildSuccessResponse(status, "Invitation revoked successfully", nil)
	return c.Status(status).JSON(rd)
}

func mapToGeneric(invitations []dtos.BusinessInvitationDto) []dtos.AggregatorInvitationDto {
	// Simple map for uniform response struct
	// This is a little hacky but returns a consistent structure for the frontend
	result := make([]dtos.AggregatorInvitationDto, len(invitations))
	for i, v := range invitations {
		result[i] = dtos.AggregatorInvitationDto{
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
