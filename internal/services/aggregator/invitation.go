package aggregator

import (
	"einvoice-access-point/internal/dtos"
	aggregatorRepo "einvoice-access-point/internal/repository/aggregator"
	businessRepo "einvoice-access-point/internal/repository/business"
	"einvoice-access-point/pkg/config"
	inst "einvoice-access-point/pkg/dbinit"
	"einvoice-access-point/pkg/models"
	"einvoice-access-point/pkg/resend_email"
	"einvoice-access-point/pkg/utility"
	"fmt"
	"net/http"
	"time"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func SendInvitation(businessID, aggregatorID string, db *gorm.DB) (int, error) {
	// Check business exists
	var business models.Business
	if err := db.Where("id = ?", businessID).First(&business).Error; err != nil {
		return http.StatusNotFound, fmt.Errorf("business not found")
	}

	if business.BusinessID == nil {
		return fiber.StatusForbidden, fmt.Errorf("Business ID has not been updated")
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
		// If invitation already exists, we resend the email to be idempotent
		resend_email.SendAggregatorInvitationEmail(aggregator.Email, business.CompanyName)
		return http.StatusOK, nil
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
	resend_email.SendAggregatorInvitationEmail(aggregator.Email, business.CompanyName)

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
			// Details:      fmt.Sprintf("Accepted invitation from %s", invitation.Business.CompanyName),
		}, db)

		// Notify business
		// resend_email.SendInvitationAcceptedEmail(invitation.Business.Email, invitation.Aggregator.CompanyName)
	} else {
		invitation.Status = models.InvitationStatusRejected
		invitation.RejectedAt = &now

		// Log activity
		aggregatorRepo.CreateActivityLog(&models.AggregatorActivityLog{
			ID:           utility.GenerateUUID(),
			AggregatorID: aggregatorID,
			BusinessID:   invitation.BusinessID,
			Action:       models.ActivityInvitationRejected,
			// Details:      fmt.Sprintf("Rejected invitation from %s", invitation.Business.CompanyName),
		}, db)

		// Notify business
		// resend_email.SendInvitationRejectedEmail(invitation.Business.Email, invitation.Aggregator.CompanyName)
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
	pdb := inst.InitDB(db, false)
	invitations, err := aggregatorRepo.ListPendingInvitationsByAggregator(db, aggregatorID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch invitations: %w", err)
	}

	result := make([]dtos.AggregatorInvitationDto, 0, len(invitations))
	for _, inv := range invitations {
		business, _ := businessRepo.FindUserByID(pdb, inv.BusinessID)
		result = append(result, dtos.AggregatorInvitationDto{
			ID:            inv.ID,
			BusinessID:    inv.BusinessID,
			BusinessName:  business.CompanyName,
			BusinessEmail: business.Email,
			Status:        inv.Status,
			CreatedAt:     inv.CreatedAt.Format(time.RFC3339),
		})
	}

	return result, nil
}

func ListBusinessInvitations(businessID string, db *gorm.DB) ([]dtos.BusinessInvitationDto, error) {
	pdb := inst.InitDB(db, false)
	invitations, err := aggregatorRepo.ListInvitationsByBusiness(db, businessID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch invitations: %w", err)
	}

	result := make([]dtos.BusinessInvitationDto, 0, len(invitations))
	for _, inv := range invitations {
		aggregator, _ := businessRepo.FindUserByID(pdb, inv.AggregatorID)
		result = append(result, dtos.BusinessInvitationDto{
			ID:              inv.ID,
			AggregatorID:    inv.AggregatorID,
			AggregatorName:  aggregator.CompanyName,
			AggregatorEmail: aggregator.Email,
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

func SendInvitationByEmail(businessID, email string, db *gorm.DB) (int, error) {
	// Check business exists
	var business models.Business
	if err := db.Where("id = ?", businessID).First(&business).Error; err != nil {
		return http.StatusNotFound, fmt.Errorf("business not found")
	}

	if business.BusinessID == nil {
		return fiber.StatusForbidden, fmt.Errorf("Business ID has not been updated")
	}

	// Check if business already has an aggregator
	if business.AggregatorID != nil && *business.AggregatorID != "" {
		return http.StatusBadRequest, fmt.Errorf("business already has an aggregator assigned")
	}

	// Look up aggregator by email
	var existingAggregator models.Business
	err := db.Where("email = ? AND is_aggregator = ?", email, true).First(&existingAggregator).Error

	if err == nil {
		// Aggregator exists!
		// Check if they are a pending aggregator we created earlier
		if existingAggregator.Name == "Pending Aggregator" || !existingAggregator.EmailVerified {
			// Find existing invitation
			existingInvite, _ := aggregatorRepo.CheckExistingActiveInvitation(db, businessID, existingAggregator.ID)
			if existingInvite != nil {
				// Resend signup email
				appUrl := config.GetConfig().App.Url
				signupLink := fmt.Sprintf("%s/aggregator/signup?invite=%s&email=%s", appUrl, existingInvite.InviteToken, email)
				resend_email.SendNewAggregatorInvitationEmail(email, business.CompanyName, signupLink)
				return http.StatusOK, nil
			}
		}
		// Otherwise fallback to normal SendInvitation which is now idempotent too
		return SendInvitation(businessID, existingAggregator.ID, db)
	} else if err != gorm.ErrRecordNotFound {
		return http.StatusInternalServerError, fmt.Errorf("database error checking for aggregator: %w", err)
	}

	// Aggregator does not exist. We create a pending aggregator profile.
	newAggregatorID := utility.GenerateUUID()
	hashPassword, _ := utility.HashPassword(utility.GenerateUUID() + utility.GenerateUUID())

	newAggregator := models.Business{
		ID:            newAggregatorID,
		Name:          "Pending Aggregator",
		CompanyName:   "Pending Aggregator",
		Email:         email,
		Password:      hashPassword,
		IsAggregator:  true,
		EmailVerified: false,
		AccStatus:     0,
		ServiceID:     business.ServiceID, // Default to same service ID
	}

	if err := db.Create(&newAggregator).Error; err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to create pending aggregator profile: %w", err)
	}

	// Create invitation
	inviteToken := utility.GenerateUUID()
	invitation := &models.AggregatorInvitation{
		ID:           utility.GenerateUUID(),
		BusinessID:   businessID,
		AggregatorID: newAggregatorID,
		Status:       models.InvitationStatusPending,
		InviteToken:  inviteToken,
	}

	if err := aggregatorRepo.CreateInvitation(invitation, db); err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to create invitation: %w", err)
	}

	// Send email notification to new aggregator
	appUrl := config.GetConfig().App.Url
	signupLink := fmt.Sprintf("%s/aggregator/signup?invite=%s&email=%s", appUrl, inviteToken, email)

	resend_email.SendNewAggregatorInvitationEmail(email, business.CompanyName, signupLink)

	return http.StatusCreated, nil
}
