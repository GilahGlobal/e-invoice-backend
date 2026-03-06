package invoice

import (
	userRepo "einvoice-access-point/internal/repository/business"
	subscriptionRepo "einvoice-access-point/internal/repository/subscription"
	inst "einvoice-access-point/pkg/dbinit"
	"errors"
	"fmt"

	"gorm.io/gorm"
)

var (
	ErrPluginSubscriptionRequired = errors.New("active subscription is required for plugin users")
	ErrPluginInvoiceQuotaExceeded = errors.New("invoice quota exhausted for current subscription")
)

func ValidatePluginInvoiceEligibility(db *gorm.DB, businessID string) (bool, error) {
	if db == nil {
		return false, fmt.Errorf("database connection is required")
	}

	pdb := inst.InitDB(db, false)
	isPluginUser, err := userRepo.IsPluginUserByID(pdb, businessID)
	if err != nil {
		return false, fmt.Errorf("failed to fetch business profile: %w", err)
	}
	if !isPluginUser {
		return false, nil
	}

	subscription, err := subscriptionRepo.GetLatestSubscriptionByBusinessID(pdb, businessID)
	if err != nil {
		return true, fmt.Errorf("failed to fetch subscription: %w", err)
	}
	if subscription == nil || !subscription.IsActive {
		return true, ErrPluginSubscriptionRequired
	}
	if subscription.RemainingInvoices <= 0 {
		return true, ErrPluginInvoiceQuotaExceeded
	}

	return true, nil
}

func ConsumePluginInvoiceQuota(db *gorm.DB, businessID string) error {
	if db == nil {
		return fmt.Errorf("database connection is required")
	}

	pdb := inst.InitDB(db, false)
	isPluginUser, err := userRepo.IsPluginUserByID(pdb, businessID)
	if err != nil {
		return fmt.Errorf("failed to fetch business profile: %w", err)
	}
	if !isPluginUser {
		return nil
	}

	subscription, err := subscriptionRepo.GetLatestSubscriptionByBusinessID(pdb, businessID)
	if err != nil {
		return fmt.Errorf("failed to fetch subscription: %w", err)
	}
	if subscription == nil || !subscription.IsActive {
		return ErrPluginSubscriptionRequired
	}
	if subscription.RemainingInvoices <= 0 {
		return ErrPluginInvoiceQuotaExceeded
	}

	subscription.UsedInvoices += 1
	subscription.RemainingInvoices -= 1

	if err := subscriptionRepo.SaveSubscription(subscription, pdb); err != nil {
		return fmt.Errorf("failed to update subscription quota: %w", err)
	}

	return nil
}
