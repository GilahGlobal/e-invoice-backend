package plan

import (
	"errors"
	"strings"

	"einvoice-access-point/pkg/database"
	"einvoice-access-point/pkg/models"

	"gorm.io/gorm"
)

func CreatePlan(plan *models.SubscriptionPlan, db database.DatabaseManager) error {
	return db.DB().Create(plan).Error
}

func GetPlans(db database.DatabaseManager) ([]models.SubscriptionPlan, error) {
	var plans []models.SubscriptionPlan
	if err := db.DB().Order("created_at asc").Find(&plans).Error; err != nil {
		return nil, err
	}
	return plans, nil
}

func GetPlanByName(planName string, db database.DatabaseManager) (*models.SubscriptionPlan, error) {
	var plan models.SubscriptionPlan
	err := db.DB().
		Where("LOWER(name) = ?", strings.ToLower(strings.TrimSpace(planName))).
		First(&plan).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &plan, nil
}

func GetPlanByID(planID string, db database.DatabaseManager) (*models.SubscriptionPlan, error) {
	var plan models.SubscriptionPlan
	err := db.DB().
		Where("id = ?", strings.TrimSpace(planID)).
		First(&plan).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &plan, nil
}
