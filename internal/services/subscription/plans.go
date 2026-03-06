package subscription

import (
	"fmt"
	"strings"

	"einvoice-access-point/internal/dtos"
	planRepo "einvoice-access-point/internal/repository/plan"
	inst "einvoice-access-point/pkg/dbinit"
	"einvoice-access-point/pkg/models"
	"einvoice-access-point/pkg/utility"

	"gorm.io/gorm"
)

func ListPlans(db *gorm.DB) ([]models.SubscriptionPlan, error) {
	if db == nil {
		return nil, fmt.Errorf("database connection is required")
	}

	pdb := inst.InitDB(db, false)
	return planRepo.GetPlans(pdb)
}

func ListActivePlans(db *gorm.DB) ([]models.SubscriptionPlan, error) {
	plans, err := ListPlans(db)
	if err != nil {
		return nil, err
	}

	activePlans := make([]models.SubscriptionPlan, 0, len(plans))
	for _, plan := range plans {
		if !plan.IsActive {
			continue
		}
		activePlans = append(activePlans, plan)
	}

	return activePlans, nil
}

func GetPlanByName(planName string, db *gorm.DB) (*models.SubscriptionPlan, bool, error) {
	if db == nil {
		return nil, false, fmt.Errorf("database connection is required")
	}

	pdb := inst.InitDB(db, false)
	plan, err := planRepo.GetPlanByName(planName, pdb)
	if err != nil {
		return nil, false, err
	}
	if plan != nil {
		return plan, true, nil
	}
	return nil, false, nil
}

func GetPlanByID(planID string, db *gorm.DB) (*models.SubscriptionPlan, bool, error) {
	if db == nil {
		return nil, false, fmt.Errorf("database connection is required")
	}

	pdb := inst.InitDB(db, false)
	plan, err := planRepo.GetPlanByID(planID, pdb)
	if err != nil {
		return nil, false, err
	}
	if plan != nil {
		return plan, true, nil
	}
	return nil, false, nil
}

func GetActivePlanByName(planName string, db *gorm.DB) (*models.SubscriptionPlan, bool, error) {
	plan, found, err := GetPlanByName(planName, db)
	if err != nil {
		return nil, false, err
	}
	if !found || plan == nil {
		return nil, false, nil
	}
	if !plan.IsActive {
		return nil, false, nil
	}

	return plan, true, nil
}

func GetActivePlanByID(planID string, db *gorm.DB) (*models.SubscriptionPlan, bool, error) {
	plan, found, err := GetPlanByID(planID, db)
	if err != nil {
		return nil, false, err
	}
	if !found || plan == nil {
		return nil, false, nil
	}
	if !plan.IsActive {
		return nil, false, nil
	}

	return plan, true, nil
}

func CreatePlan(req dtos.CreateSubscriptionPlanDto, db *gorm.DB) (*models.SubscriptionPlan, error) {
	if db == nil {
		return nil, fmt.Errorf("database connection is required")
	}

	pdb := inst.InitDB(db, false)
	planName := strings.TrimSpace(req.Name)

	existingPlan, err := planRepo.GetPlanByName(planName, pdb)
	if err != nil {
		return nil, err
	}
	if existingPlan != nil {
		return nil, fmt.Errorf("plan with name %s already exists", planName)
	}

	plan := &models.SubscriptionPlan{
		ID:            utility.GenerateUUID(),
		Name:          planName,
		Amount:        req.Amount,
		IsActive:      true,
		TotalInvoices: req.TotalInvoices,
		BillingCycle:  req.BillingCycle,
	}

	if err := planRepo.CreatePlan(plan, pdb); err != nil {
		return nil, err
	}

	return plan, nil
}
