package repositories

import (
	"errors"
	"strings"

	"einvoice-access-point/internal/data/database"
	"einvoice-access-point/internal/data/entities"

	"gorm.io/gorm"
)

type SubscriptionRepository struct {
	prodDb database.DatabaseManager
	testDb database.DatabaseManager
}

func NewSubscriptionRepository(prodDb, testDb database.DatabaseManager) *SubscriptionRepository {
	return &SubscriptionRepository{prodDb: prodDb, testDb: testDb}
}

func (r *SubscriptionRepository) CreatePlan(plan *entities.SubscriptionPlan, db database.DatabaseManager) error {
	return db.DB().Create(plan).Error
}

func (r *SubscriptionRepository) GetPlans(db database.DatabaseManager) ([]entities.SubscriptionPlan, error) {
	var plans []entities.SubscriptionPlan
	if err := db.DB().Order("created_at asc").Find(&plans).Error; err != nil {
		return nil, err
	}
	return plans, nil
}

func (r *SubscriptionRepository) GetPlanByName(planName string, db database.DatabaseManager) (*entities.SubscriptionPlan, error) {
	var plan entities.SubscriptionPlan
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

func (r *SubscriptionRepository) GetPlanByID(planID string, db database.DatabaseManager) (*entities.SubscriptionPlan, error) {
	var plan entities.SubscriptionPlan
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

func (r *SubscriptionRepository) CreateSubscription(subscription *entities.Subscription, db database.DatabaseManager) error {
	return db.DB().Create(subscription).Error
}

func (r *SubscriptionRepository) GetLatestSubscriptionByBusinessID(db database.DatabaseManager, businessID string) (*entities.Subscription, error) {
	var subscription entities.Subscription

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

func (r *SubscriptionRepository) GetLatestSubscriptionByBusinessAndAggregator(db database.DatabaseManager, businessID, aggregatorID string) (*entities.Subscription, error) {
	var subscription entities.Subscription

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

func (r *SubscriptionRepository) SaveSubscription(subscription *entities.Subscription, db database.DatabaseManager) error {
	return db.DB().Save(subscription).Error
}

func (r *SubscriptionRepository) ReserveSubscriptionInvoices(db database.DatabaseManager, subscriptionID string, count int) (bool, error) {
	result := db.DB().
		Model(&entities.Subscription{}).
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

func (r *SubscriptionRepository) ReleaseSubscriptionInvoices(db database.DatabaseManager, subscriptionID string, count int) error {
	return db.DB().
		Model(&entities.Subscription{}).
		Where("id = ?", subscriptionID).
		Updates(map[string]interface{}{
			"used_invoices":      gorm.Expr("GREATEST(used_invoices - ?, 0)", count),
			"remaining_invoices": gorm.Expr("remaining_invoices + ?", count),
		}).Error
}

func (r *SubscriptionRepository) CreateTransaction(record *entities.Transaction, db database.DatabaseManager) error {
	return db.DB().Create(record).Error
}

func (r *SubscriptionRepository) GetTransactionByReference(reference string, db database.DatabaseManager) (*entities.Transaction, error) {
	var transaction entities.Transaction
	err := db.DB().Where("reference = ?", reference).First(&transaction).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &transaction, nil
}

func (r *SubscriptionRepository) SaveTransaction(record *entities.Transaction, db database.DatabaseManager) error {
	return db.DB().Save(record).Error
}

func (r *SubscriptionRepository) ListAllTransactions(db database.DatabaseManager, page, size int) ([]entities.Transaction, int64, error) {
	var transactions []entities.Transaction
	var total int64

	query := db.DB().Model(&entities.Transaction{})

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * size
	if err := query.Offset(offset).Limit(size).Order("created_at DESC").Find(&transactions).Error; err != nil {
		return nil, 0, err
	}

	return transactions, total, nil
}
