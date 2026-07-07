package aggregator

import (
	"einvoice-access-point/internal/config"
	"einvoice-access-point/internal/data/dbinit"
	"einvoice-access-point/internal/data/entities"
	"einvoice-access-point/internal/data/repositories"
	"einvoice-access-point/internal/pkg/resend_email"
	"einvoice-access-point/internal/utility"
	"fmt"
	"net/http"
	"time"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func (s *Service) SendInvitation(businessID, aggregatorID string, db *gorm.DB) (int, error) {
	// Check business exists
	var business entities.Business
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
	aggregator, err := s.repo.GetAggregatorByID(db, aggregatorID)
	if err != nil {
		return http.StatusNotFound, fmt.Errorf("aggregator not found")
	}

	// Check for existing active invitation
	existing, err := s.repo.CheckExistingActiveInvitation(db, businessID, aggregatorID)
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
	invitation := &entities.AggregatorInvitation{
		ID:           utility.GenerateUUID(),
		BusinessID:   businessID,
		AggregatorID: aggregatorID,
		Status:       entities.InvitationStatusPending,
		InviteToken:  inviteToken,
	}

	if err := s.repo.CreateInvitation(invitation, db); err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to create invitation: %w", err)
	}

	// Send email notification to aggregator
	resend_email.SendAggregatorInvitationEmail(aggregator.Email, business.CompanyName)

	return http.StatusCreated, nil
}

func (s *Service) RespondToInvitation(invitationID, aggregatorID string, accept bool, db *gorm.DB) (int, error) {
	invitation, err := s.repo.GetInvitationByID(db, invitationID)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to fetch invitation: %w", err)
	}
	if invitation == nil {
		return http.StatusNotFound, fmt.Errorf("invitation not found")
	}

	if invitation.AggregatorID != aggregatorID {
		return http.StatusForbidden, fmt.Errorf("this invitation does not belong to you")
	}

	if invitation.Status != entities.InvitationStatusPending {
		return http.StatusBadRequest, fmt.Errorf("invitation has already been %s", invitation.Status)
	}

	now := time.Now()

	if accept {
		// Check if business already got a different aggregator in the meantime
		var business entities.Business
		if err := db.Where("id = ?", invitation.BusinessID).First(&business).Error; err != nil {
			return http.StatusInternalServerError, fmt.Errorf("failed to fetch business: %w", err)
		}
		if business.AggregatorID != nil && *business.AggregatorID != "" {
			return http.StatusBadRequest, fmt.Errorf("business already has an aggregator assigned")
		}

		invitation.Status = entities.InvitationStatusAccepted
		invitation.AcceptedAt = &now

		// Link business to aggregator
		if err := db.Model(&entities.Business{}).Where("id = ?", invitation.BusinessID).
			Update("aggregator_id", aggregatorID).Error; err != nil {
			return http.StatusInternalServerError, fmt.Errorf("failed to link business to aggregator: %w", err)
		}

		// Log activity
		s.repo.CreateActivityLog(&entities.AggregatorActivityLog{
			ID:           utility.GenerateUUID(),
			AggregatorID: aggregatorID,
			BusinessID:   invitation.BusinessID,
			Action:       entities.ActivityInvitationAccepted,
			// Details:      fmt.Sprintf("Accepted invitation from %s", invitation.Business.CompanyName),
		}, db)

		// Notify business
		// resend_email.SendInvitationAcceptedEmail(invitation.Business.Email, invitation.Aggregator.CompanyName)
	} else {
		invitation.Status = entities.InvitationStatusRejected
		invitation.RejectedAt = &now

		// Log activity
		s.repo.CreateActivityLog(&entities.AggregatorActivityLog{
			ID:           utility.GenerateUUID(),
			AggregatorID: aggregatorID,
			BusinessID:   invitation.BusinessID,
			Action:       entities.ActivityInvitationRejected,
			// Details:      fmt.Sprintf("Rejected invitation from %s", invitation.Business.CompanyName),
		}, db)

		// Notify business
		// resend_email.SendInvitationRejectedEmail(invitation.Business.Email, invitation.Aggregator.CompanyName)
	}

	if err := s.repo.UpdateInvitation(invitation, db); err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to update invitation: %w", err)
	}

	return http.StatusOK, nil
}

func (s *Service) RevokeInvitation(invitationID, businessID string, db *gorm.DB) (int, error) {
	invitation, err := s.repo.GetInvitationByID(db, invitationID)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to fetch invitation: %w", err)
	}
	if invitation == nil {
		return http.StatusNotFound, fmt.Errorf("invitation not found")
	}
	if invitation.BusinessID != businessID {
		return http.StatusForbidden, fmt.Errorf("this invitation does not belong to your business")
	}
	if invitation.Status == entities.InvitationStatusRevoked {
		return http.StatusBadRequest, fmt.Errorf("invitation is already revoked")
	}

	// If invitation was accepted, unlink the aggregator from the business
	if invitation.Status == entities.InvitationStatusAccepted {
		if err := db.Model(&entities.Business{}).Where("id = ?", businessID).
			Update("aggregator_id", nil).Error; err != nil {
			return http.StatusInternalServerError, fmt.Errorf("failed to unlink aggregator: %w", err)
		}
	}

	invitation.Status = entities.InvitationStatusRevoked
	if err := s.repo.UpdateInvitation(invitation, db); err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to revoke invitation: %w", err)
	}

	return http.StatusOK, nil
}

func (s *Service) ListAggregatorInvitations(aggregatorID string, db *gorm.DB) ([]AggregatorInvitationDto, error) {
	pdb := dbinit.InitDB(db, false)
	invitations, err := s.repo.ListPendingInvitationsByAggregator(db, aggregatorID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch invitations: %w", err)
	}

	result := make([]AggregatorInvitationDto, 0, len(invitations))
	for _, inv := range invitations {
		bRepo := repositories.NewBusinessRepository(pdb, pdb)
		business, _ := bRepo.FindUserByID(pdb, inv.BusinessID)

		bizName := "Unknown Business"
		bizEmail := "unknown"
		if business != nil {
			bizName = business.CompanyName
			bizEmail = business.Email
		}

		result = append(result, AggregatorInvitationDto{
			ID:            inv.ID,
			BusinessID:    inv.BusinessID,
			BusinessName:  bizName,
			BusinessEmail: bizEmail,
			Status:        inv.Status,
			CreatedAt:     inv.CreatedAt.Format(time.RFC3339),
		})
	}

	return result, nil
}

func (s *Service) ListBusinessInvitations(businessID string, db *gorm.DB) ([]BusinessInvitationDto, error) {
	pdb := dbinit.InitDB(db, false)
	invitations, err := s.repo.ListInvitationsByBusiness(db, businessID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch invitations: %w", err)
	}

	result := make([]BusinessInvitationDto, 0, len(invitations))
	for _, inv := range invitations {
		bRepo := repositories.NewBusinessRepository(pdb, pdb)
		aggregator, _ := bRepo.FindUserByID(pdb, inv.AggregatorID)

		aggName := "Pending Aggregator"
		aggEmail := "Pending"
		if aggregator != nil {
			aggName = aggregator.CompanyName
			aggEmail = aggregator.Email
		} else {
			// Fallback: try finding the aggregator without considering AccStatus
			var fallbackAgg entities.Business
			if err := db.Unscoped().Where("id = ?", inv.AggregatorID).First(&fallbackAgg).Error; err == nil {
				aggName = fallbackAgg.CompanyName
				aggEmail = fallbackAgg.Email
			}
		}

		result = append(result, BusinessInvitationDto{
			ID:              inv.ID,
			AggregatorID:    inv.AggregatorID,
			AggregatorName:  aggName,
			AggregatorEmail: aggEmail,
			Status:          inv.Status,
			CreatedAt:       inv.CreatedAt.Format(time.RFC3339),
		})
	}

	return result, nil
}

func (s *Service) ListAvailableAggregators(search string, page, size int, db *gorm.DB) ([]AvailableAggregatorDto, int64, error) {
	aggregators, total, err := s.repo.ListAllAggregators(db, search, page, size)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to fetch aggregators: %w", err)
	}

	result := make([]AvailableAggregatorDto, 0, len(aggregators))
	for _, agg := range aggregators {
		result = append(result, AvailableAggregatorDto{
			ID:          agg.ID,
			Name:        agg.Name,
			Email:       agg.Email,
			CompanyName: agg.CompanyName,
			PhoneNumber: agg.PhoneNumber,
		})
	}

	return result, total, nil
}

func (s *Service) SendInvitationByEmail(businessID, email string, db *gorm.DB) (int, error) {
	// Check business exists
	var business entities.Business
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
	var existingAggregator entities.Business
	err := db.Where("email = ? AND is_aggregator = ?", email, true).First(&existingAggregator).Error

	if err == nil {
		// Aggregator exists!
		// Check if they are a pending aggregator we created earlier
		if existingAggregator.Name == "Pending Aggregator" || !existingAggregator.EmailVerified {
			// Find existing invitation
			existingInvite, _ := s.repo.CheckExistingActiveInvitation(db, businessID, existingAggregator.ID)
			if existingInvite != nil {
				// Resend signup email
				appUrl := config.GetConfig().App.Url
				signupLink := fmt.Sprintf("%s/aggregator/signup?invite=%s&email=%s", appUrl, existingInvite.InviteToken, email)
				resend_email.SendNewAggregatorInvitationEmail(email, business.CompanyName, signupLink)
				return http.StatusOK, nil
			}
		}
		// Otherwise fallback to normal SendInvitation which is now idempotent too
		return s.SendInvitation(businessID, existingAggregator.ID, db)
	} else if err != gorm.ErrRecordNotFound {
		return http.StatusInternalServerError, fmt.Errorf("database error checking for aggregator: %w", err)
	}

	// Aggregator does not exist. We create a pending aggregator profile.
	newAggregatorID := utility.GenerateUUID()
	hashPassword, _ := utility.HashPassword(utility.GenerateUUID() + utility.GenerateUUID())

	newAggregator := entities.Business{
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
	invitation := &entities.AggregatorInvitation{
		ID:           utility.GenerateUUID(),
		BusinessID:   businessID,
		AggregatorID: newAggregatorID,
		Status:       entities.InvitationStatusPending,
		InviteToken:  inviteToken,
	}

	if err := s.repo.CreateInvitation(invitation, db); err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to create invitation: %w", err)
	}

	// Send email notification to new aggregator
	appUrl := config.GetConfig().App.Url
	signupLink := fmt.Sprintf("%s/aggregator/signup?invite=%s&email=%s", appUrl, inviteToken, email)

	resend_email.SendNewAggregatorInvitationEmail(email, business.CompanyName, signupLink)

	return http.StatusCreated, nil
}
