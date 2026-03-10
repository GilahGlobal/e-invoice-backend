package invoice

import (
	"einvoice-access-point/internal/repository/sme"
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

func ValidatePluginInvoiceEligibility(db *gorm.DB, smeID string) (bool, error) {

	pdb := inst.InitDB(db, false)
	smeBusiness, err := sme.FindSmeByID(pdb, smeID)
	if err != nil {
		return false, fmt.Errorf("failed to fetch business profile: %w", err)
	}
	if smeBusiness == nil {
		return false, nil
	}

	subscription, err := subscriptionRepo.GetLatestSubscriptionByBusinessID(pdb, smeID)
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

func ConsumePluginInvoiceQuota(db *gorm.DB, smeID string) error {
	if db == nil {
		return fmt.Errorf("database connection is required")
	}

	pdb := inst.InitDB(db, false)
	smeBusiness, err := sme.FindSmeByID(pdb, smeID)
	if err != nil {
		return fmt.Errorf("failed to fetch business profile: %w", err)
	}
	if smeBusiness == nil {
		return nil
	}

	subscription, err := subscriptionRepo.GetLatestSubscriptionByBusinessID(pdb, smeID)
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
