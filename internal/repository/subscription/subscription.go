package subscription

import (
	"errors"

	"einvoice-access-point/pkg/database"
	"einvoice-access-point/pkg/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func CreateSubscription(subscription *models.Subscription, db database.DatabaseManager) error {
	return db.DB().Create(subscription).Error
}

func GetLatestSubscriptionByBusinessID(db database.DatabaseManager, smeID string) (*models.Subscription, error) {
	var subscription models.Subscription

	smeUUID, _ := uuid.Parse(smeID)
	err := db.DB().
		Where("sme_id = ? AND is_active = true", smeUUID).
		Order("created_at desc").
		First(&subscription).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	return &subscription, nil
}

func SaveSubscription(subscription *models.Subscription, db database.DatabaseManager) error {
	return db.DB().Save(subscription).Error
}
