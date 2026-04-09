package aggregator

import (
	"einvoice-access-point/internal/dtos"
	aggregatorRepo "einvoice-access-point/internal/repository/aggregator"
	"einvoice-access-point/pkg/models"
	"einvoice-access-point/pkg/ses"
	"einvoice-access-point/pkg/utility"
	"fmt"
	"net/http"
	"time"

	"gorm.io/gorm"
)

func SendInvitation(businessID, aggregatorID string, db *gorm.DB) (int, error) {
	// Check business exists
	var business models.Business
	if err := db.Where("id = ?", businessID).First(&business).Error; err != nil {
		return http.StatusNotFound, fmt.Errorf("business not found")
	}

	// Check if business already has an aggregator
	if business.AggregatorID != nil && *business.AggregatorID != "" {
		return http.StatusBadRequest, fmt.Errorf("business already has an aggregator assigned")
	}

	// Check aggregator exists
	aggregator, err := aggregatorRepo.GetAggregatorByID(db, aggregatorID)
	if err != nil {
		return http.StatusNotFound, fmt.Errorf("aggregator not found")
	}

	// Check for existing active invitation
	existing, err := aggregatorRepo.CheckExistingActiveInvitation(db, businessID, aggregatorID)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to check existing invitations: %w", err)
	}
	if existing != nil {
		return http.StatusBadRequest, fmt.Errorf("an active invitation already exists for this aggregator")
	}

	// Create invitation
	inviteToken := utility.GenerateUUID()
	invitation := &models.AggregatorInvitation{
		ID:           utility.GenerateUUID(),
		BusinessID:   businessID,
		AggregatorID: aggregatorID,
		Status:       models.InvitationStatusPending,
		InviteToken:  inviteToken,
	}

	if err := aggregatorRepo.CreateInvitation(invitation, db); err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to create invitation: %w", err)
	}

	// Send email notification to aggregator
	ses.SendAggregatorInvitationEmail(aggregator.Email, business.CompanyName)

	return http.StatusCreated, nil
}

func RespondToInvitation(invitationID, aggregatorID string, accept bool, db *gorm.DB) (int, error) {
	invitation, err := aggregatorRepo.GetInvitationByID(db, invitationID)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to fetch invitation: %w", err)
	}
	if invitation == nil {
		return http.StatusNotFound, fmt.Errorf("invitation not found")
	}

	if invitation.AggregatorID != aggregatorID {
		return http.StatusForbidden, fmt.Errorf("this invitation does not belong to you")
	}

	if invitation.Status != models.InvitationStatusPending {
		return http.StatusBadRequest, fmt.Errorf("invitation has already been %s", invitation.Status)
	}

	now := time.Now()

	if accept {
		// Check if business already got a different aggregator in the meantime
		var business models.Business
		if err := db.Where("id = ?", invitation.BusinessID).First(&business).Error; err != nil {
			return http.StatusInternalServerError, fmt.Errorf("failed to fetch business: %w", err)
		}
		if business.AggregatorID != nil && *business.AggregatorID != "" {
			return http.StatusBadRequest, fmt.Errorf("business already has an aggregator assigned")
		}

		invitation.Status = models.InvitationStatusAccepted
		invitation.AcceptedAt = &now

		// Link business to aggregator
		if err := db.Model(&models.Business{}).Where("id = ?", invitation.BusinessID).
			Update("aggregator_id", aggregatorID).Error; err != nil {
			return http.StatusInternalServerError, fmt.Errorf("failed to link business to aggregator: %w", err)
		}

		// Log activity
		aggregatorRepo.CreateActivityLog(&models.AggregatorActivityLog{
			ID:           utility.GenerateUUID(),
			AggregatorID: aggregatorID,
			BusinessID:   invitation.BusinessID,
			Action:       models.ActivityInvitationAccepted,
			Details:      fmt.Sprintf("Accepted invitation from %s", invitation.Business.CompanyName),
		}, db)

		// Notify business
		ses.SendInvitationAcceptedEmail(invitation.Business.Email, invitation.Aggregator.CompanyName)
	} else {
		invitation.Status = models.InvitationStatusRejected
		invitation.RejectedAt = &now

		// Log activity
		aggregatorRepo.CreateActivityLog(&models.AggregatorActivityLog{
			ID:           utility.GenerateUUID(),
			AggregatorID: aggregatorID,
			BusinessID:   invitation.BusinessID,
			Action:       models.ActivityInvitationRejected,
			Details:      fmt.Sprintf("Rejected invitation from %s", invitation.Business.CompanyName),
		}, db)

		// Notify business
		ses.SendInvitationRejectedEmail(invitation.Business.Email, invitation.Aggregator.CompanyName)
	}

	if err := aggregatorRepo.UpdateInvitation(invitation, db); err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to update invitation: %w", err)
	}

	return http.StatusOK, nil
}

func RevokeInvitation(invitationID, businessID string, db *gorm.DB) (int, error) {
	invitation, err := aggregatorRepo.GetInvitationByID(db, invitationID)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to fetch invitation: %w", err)
	}
	if invitation == nil {
		return http.StatusNotFound, fmt.Errorf("invitation not found")
	}
	if invitation.BusinessID != businessID {
		return http.StatusForbidden, fmt.Errorf("this invitation does not belong to your business")
	}
	if invitation.Status == models.InvitationStatusRevoked {
		return http.StatusBadRequest, fmt.Errorf("invitation is already revoked")
	}

	// If invitation was accepted, unlink the aggregator from the business
	if invitation.Status == models.InvitationStatusAccepted {
		if err := db.Model(&models.Business{}).Where("id = ?", businessID).
			Update("aggregator_id", nil).Error; err != nil {
			return http.StatusInternalServerError, fmt.Errorf("failed to unlink aggregator: %w", err)
		}
	}

	invitation.Status = models.InvitationStatusRevoked
	if err := aggregatorRepo.UpdateInvitation(invitation, db); err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to revoke invitation: %w", err)
	}

	return http.StatusOK, nil
}

func ListAggregatorInvitations(aggregatorID string, db *gorm.DB) ([]dtos.AggregatorInvitationDto, error) {
	invitations, err := aggregatorRepo.ListPendingInvitationsByAggregator(db, aggregatorID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch invitations: %w", err)
	}

	result := make([]dtos.AggregatorInvitationDto, 0, len(invitations))
	for _, inv := range invitations {
		result = append(result, dtos.AggregatorInvitationDto{
			ID:            inv.ID,
			BusinessID:    inv.BusinessID,
			BusinessName:  inv.Business.CompanyName,
			BusinessEmail: inv.Business.Email,
			Status:        inv.Status,
			CreatedAt:     inv.CreatedAt.Format(time.RFC3339),
		})
	}

	return result, nil
}

func ListBusinessInvitations(businessID string, db *gorm.DB) ([]dtos.BusinessInvitationDto, error) {
	invitations, err := aggregatorRepo.ListInvitationsByBusiness(db, businessID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch invitations: %w", err)
	}

	result := make([]dtos.BusinessInvitationDto, 0, len(invitations))
	for _, inv := range invitations {
		result = append(result, dtos.BusinessInvitationDto{
			ID:              inv.ID,
			AggregatorID:    inv.AggregatorID,
			AggregatorName:  inv.Aggregator.CompanyName,
			AggregatorEmail: inv.Aggregator.Email,
			Status:          inv.Status,
			CreatedAt:       inv.CreatedAt.Format(time.RFC3339),
		})
	}

	return result, nil
}

func ListAvailableAggregators(search string, page, size int, db *gorm.DB) ([]dtos.AvailableAggregatorDto, int64, error) {
	aggregators, total, err := aggregatorRepo.ListAllAggregators(db, search, page, size)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to fetch aggregators: %w", err)
	}

	result := make([]dtos.AvailableAggregatorDto, 0, len(aggregators))
	for _, agg := range aggregators {
		result = append(result, dtos.AvailableAggregatorDto{
			ID:          agg.ID,
			Name:        agg.Name,
			Email:       agg.Email,
			CompanyName: agg.CompanyName,
			PhoneNumber: agg.PhoneNumber,
		})
	}

	return result, total, nil
}
