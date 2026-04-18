package subscription

import (
	"errors"

	"einvoice-access-point/pkg/database"
	"einvoice-access-point/pkg/models"

	"gorm.io/gorm"
)

func CreateSubscription(subscription *models.Subscription, db database.DatabaseManager) error {
	return db.DB().Create(subscription).Error
}

func GetLatestSubscriptionByBusinessID(db database.DatabaseManager, businessID string) (*models.Subscription, error) {
	var subscription models.Subscription

	err := db.DB().
		Where("business_id = ?", businessID).
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

func GetLatestSubscriptionByBusinessAndAggregator(db database.DatabaseManager, businessID, aggregatorID string) (*models.Subscription, error) {
	var subscription models.Subscription

	err := db.DB().
		Where("business_id = ? AND aggregator_id = ?", businessID, aggregatorID).
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

func ReserveSubscriptionInvoices(db database.DatabaseManager, subscriptionID string, count int) (bool, error) {
	result := db.DB().
		Model(&models.Subscription{}).
		Where("id = ? AND is_active = ? AND remaining_invoices >= ?", subscriptionID, true, count).
		Updates(map[string]interface{}{
			"used_invoices":      gorm.Expr("used_invoices + ?", count),
			"remaining_invoices": gorm.Expr("remaining_invoices - ?", count),
		})
	if result.Error != nil {
		return false, result.Error
	}

	return result.RowsAffected == 1, nil
}

func ReleaseSubscriptionInvoices(db database.DatabaseManager, subscriptionID string, count int) error {
	return db.DB().
		Model(&models.Subscription{}).
		Where("id = ?", subscriptionID).
		Updates(map[string]interface{}{
			"used_invoices":      gorm.Expr("GREATEST(used_invoices - ?, 0)", count),
			"remaining_invoices": gorm.Expr("remaining_invoices + ?", count),
		}).Error
}
