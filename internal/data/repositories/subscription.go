package repositories

import (
	"encoding/json"
	"errors"
	"sort"
	"strings"
	"time"

	"einvoice-access-point/internal/data/database"
	"einvoice-access-point/internal/data/entities"

	"gorm.io/datatypes"
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

func (r *SubscriptionRepository) UpdatePlan(plan *entities.SubscriptionPlan, db database.DatabaseManager) error {
	return db.DB().Save(plan).Error
}

func (r *SubscriptionRepository) UpdatePlanStatus(planID string, isActive bool, db database.DatabaseManager) error {
	return db.DB().Model(&entities.SubscriptionPlan{}).
		Where("id = ?", strings.TrimSpace(planID)).
		Update("is_active", isActive).Error
}

func (r *SubscriptionRepository) CountSubscriptionsByPlanID(planID string, activeOnly bool, db database.DatabaseManager) (int64, error) {
	var count int64
	q := db.DB().Model(&entities.Subscription{}).Where("plan_id = ?", strings.TrimSpace(planID))
	if activeOnly {
		q = q.Where("is_active = ? AND next_billing_date > ?", true, time.Now().UTC())
	}
	err := q.Count(&count).Error
	return count, err
}

func (r *SubscriptionRepository) DeletePlan(planID string, db database.DatabaseManager) error {
	return db.DB().Where("id = ?", strings.TrimSpace(planID)).Delete(&entities.SubscriptionPlan{}).Error
}

type AdminPlanItem struct {
	entities.SubscriptionPlan
	ActiveSubscribersCount int64 `json:"active_subscribers_count"`
	TotalSubscribersCount  int64 `json:"total_subscribers_count"`
}

func (r *SubscriptionRepository) GetPlansWithSubscriberCounts(db database.DatabaseManager) ([]AdminPlanItem, error) {
	plans, err := r.GetPlans(db)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	items := make([]AdminPlanItem, 0, len(plans))
	for _, p := range plans {
		var activeCount int64
		var totalCount int64
		_ = db.DB().Model(&entities.Subscription{}).
			Where("plan_id = ? AND is_active = ? AND next_billing_date > ?", p.ID, true, now).
			Count(&activeCount).Error
		_ = db.DB().Model(&entities.Subscription{}).
			Where("plan_id = ?", p.ID).
			Count(&totalCount).Error
		items = append(items, AdminPlanItem{
			SubscriptionPlan:       p,
			ActiveSubscribersCount: activeCount,
			TotalSubscribersCount:  totalCount,
		})
	}
	return items, nil
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

type AdminSubscriptionFilter struct {
	Status       string // "all", "active", "expiring_soon", "expired"
	ExpiringDays int
	Search       string
	BusinessID   string
	AggregatorID string
	PlanID       string
}

type AdminSubscriptionItem struct {
	ID                 string         `json:"id"`
	BusinessID         string         `json:"business_id"`
	BusinessName       string         `json:"business_name"`
	BusinessEmail      string         `json:"business_email"`
	BusinessCompany    string         `json:"business_company"`
	BusinessTIN        string         `json:"business_tin"`
	BusinessPhone      string         `json:"business_phone"`
	AggregatorID       string         `json:"aggregator_id"`
	AggregatorName     string         `json:"aggregator_name"`
	AggregatorEmail    string         `json:"aggregator_email"`
	AggregatorCompany  string         `json:"aggregator_company"`
	PlanID             string         `json:"plan_id"`
	Plan               string         `json:"plan"`
	IsActive           bool           `json:"is_active"`
	TotalInvoices      int            `json:"total_invoices"`
	RemainingInvoices  int            `json:"remaining_invoices"`
	UsedInvoices       int            `json:"used_invoices"`
	NextBillingDate    time.Time      `json:"next_billing_date"`
	RemindersSentCount int            `json:"reminders_sent_count"`
	LastReminderSentAt *time.Time     `json:"last_reminder_sent_at"`
	PlanReminderDays   datatypes.JSON `json:"plan_reminder_days,omitempty"`
	PlanBillingCycle   int            `json:"plan_billing_cycle,omitempty"`
	PlanAmount         float64        `json:"plan_amount,omitempty"`
	CreatedAt          time.Time      `json:"created_at"`
	UpdatedAt          time.Time      `json:"updated_at"`
}

func (item *AdminSubscriptionItem) GetReminderDays() []int {
	if len(item.PlanReminderDays) == 0 {
		return []int{7, 3, 1}
	}
	var days []int
	if err := json.Unmarshal(item.PlanReminderDays, &days); err != nil || len(days) == 0 {
		return []int{7, 3, 1}
	}
	sort.Sort(sort.Reverse(sort.IntSlice(days)))
	return days
}

type AdminSubscriptionStats struct {
	TotalSubscriptions   int64 `json:"total_subscriptions"`
	ActiveSubscriptions  int64 `json:"active_subscriptions"`
	ExpiringIn3Days      int64 `json:"expiring_in_3_days"`
	ExpiringIn7Days      int64 `json:"expiring_in_7_days"`
	ExpiringIn14Days     int64 `json:"expiring_in_14_days"`
	ExpiredSubscriptions int64 `json:"expired_subscriptions"`
}

const adminSubscriptionSelect = `
	subscriptions.id,
	subscriptions.business_id,
	COALESCE(b.name, '') as business_name,
	COALESCE(b.email, '') as business_email,
	COALESCE(b.company_name, '') as business_company,
	COALESCE(b.tin, '') as business_tin,
	COALESCE(b.phone_number, '') as business_phone,
	subscriptions.aggregator_id,
	COALESCE(a.name, '') as aggregator_name,
	COALESCE(a.email, '') as aggregator_email,
	COALESCE(a.company_name, '') as aggregator_company,
	subscriptions.plan_id,
	subscriptions.plan,
	subscriptions.is_active,
	subscriptions.total_invoices,
	subscriptions.remaining_invoices,
	subscriptions.used_invoices,
	subscriptions.next_billing_date,
	subscriptions.reminders_sent_count,
	subscriptions.last_reminder_sent_at,
	COALESCE(sp.reminder_days_before, '[]'::jsonb) as plan_reminder_days,
	COALESCE(sp.billing_cycle, 0) as plan_billing_cycle,
	COALESCE(sp.amount, 0) as plan_amount,
	subscriptions.created_at,
	subscriptions.updated_at
`

func (r *SubscriptionRepository) ListSubscriptions(db database.DatabaseManager, filter AdminSubscriptionFilter, page, size int) ([]AdminSubscriptionItem, int64, error) {
	var items []AdminSubscriptionItem
	var total int64

	now := time.Now().UTC()
	query := db.DB().Table("subscriptions").
		Joins("LEFT JOIN businesses b ON subscriptions.business_id = b.id").
		Joins("LEFT JOIN businesses a ON subscriptions.aggregator_id = a.id").
		Joins("LEFT JOIN subscription_plans sp ON subscriptions.plan_id = sp.id")

	if filter.BusinessID != "" {
		query = query.Where("subscriptions.business_id = ?", strings.TrimSpace(filter.BusinessID))
	}
	if filter.AggregatorID != "" {
		query = query.Where("subscriptions.aggregator_id = ?", strings.TrimSpace(filter.AggregatorID))
	}
	if filter.PlanID != "" {
		query = query.Where("subscriptions.plan_id = ?", strings.TrimSpace(filter.PlanID))
	}

	if filter.Search != "" {
		searchTerm := "%" + strings.ToLower(strings.TrimSpace(filter.Search)) + "%"
		query = query.Where("(LOWER(b.name) LIKE ? OR LOWER(b.email) LIKE ? OR LOWER(b.company_name) LIKE ? OR LOWER(a.name) LIKE ? OR LOWER(a.company_name) LIKE ? OR LOWER(subscriptions.plan) LIKE ?)",
			searchTerm, searchTerm, searchTerm, searchTerm, searchTerm, searchTerm)
	}

	switch strings.ToLower(filter.Status) {
	case "active":
		query = query.Where("subscriptions.is_active = ? AND subscriptions.next_billing_date > ?", true, now)
	case "expired":
		query = query.Where("subscriptions.is_active = ? OR subscriptions.next_billing_date <= ?", false, now)
	case "expiring_soon":
		days := filter.ExpiringDays
		if days <= 0 {
			days = 7
		}
		targetDate := now.AddDate(0, 0, days)
		query = query.Where("subscriptions.is_active = ? AND subscriptions.next_billing_date > ? AND subscriptions.next_billing_date <= ?", true, now, targetDate)
	}

	if filter.ExpiringDays > 0 && filter.Status != "expiring_soon" {
		targetDate := now.AddDate(0, 0, filter.ExpiringDays)
		query = query.Where("subscriptions.is_active = ? AND subscriptions.next_billing_date > ? AND subscriptions.next_billing_date <= ?", true, now, targetDate)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * size
	err := query.Select(adminSubscriptionSelect).
		Order("subscriptions.next_billing_date ASC, subscriptions.created_at DESC").
		Offset(offset).
		Limit(size).
		Scan(&items).Error
	if err != nil {
		return nil, 0, err
	}

	return items, total, nil
}

func (r *SubscriptionRepository) GetSubscriptionByIDWithDetails(db database.DatabaseManager, subscriptionID string) (*AdminSubscriptionItem, error) {
	var item AdminSubscriptionItem
	err := db.DB().Table("subscriptions").
		Joins("LEFT JOIN businesses b ON subscriptions.business_id = b.id").
		Joins("LEFT JOIN businesses a ON subscriptions.aggregator_id = a.id").
		Joins("LEFT JOIN subscription_plans sp ON subscriptions.plan_id = sp.id").
		Where("subscriptions.id = ?", strings.TrimSpace(subscriptionID)).
		Select(adminSubscriptionSelect).
		Scan(&item).Error
	if err != nil {
		return nil, err
	}
	if item.ID == "" {
		return nil, nil
	}
	return &item, nil
}

func (r *SubscriptionRepository) GetSubscriptionStats(db database.DatabaseManager) (*AdminSubscriptionStats, error) {
	now := time.Now().UTC()
	var stats AdminSubscriptionStats

	db.DB().Model(&entities.Subscription{}).Count(&stats.TotalSubscriptions)

	db.DB().Model(&entities.Subscription{}).
		Where("is_active = ? AND next_billing_date > ?", true, now).
		Count(&stats.ActiveSubscriptions)

	db.DB().Model(&entities.Subscription{}).
		Where("is_active = ? OR next_billing_date <= ?", false, now).
		Count(&stats.ExpiredSubscriptions)

	in3Days := now.AddDate(0, 0, 3)
	db.DB().Model(&entities.Subscription{}).
		Where("is_active = ? AND next_billing_date > ? AND next_billing_date <= ?", true, now, in3Days).
		Count(&stats.ExpiringIn3Days)

	in7Days := now.AddDate(0, 0, 7)
	db.DB().Model(&entities.Subscription{}).
		Where("is_active = ? AND next_billing_date > ? AND next_billing_date <= ?", true, now, in7Days).
		Count(&stats.ExpiringIn7Days)

	in14Days := now.AddDate(0, 0, 14)
	db.DB().Model(&entities.Subscription{}).
		Where("is_active = ? AND next_billing_date > ? AND next_billing_date <= ?", true, now, in14Days).
		Count(&stats.ExpiringIn14Days)

	return &stats, nil
}

func (r *SubscriptionRepository) CreateReminderLog(db database.DatabaseManager, log *entities.SubscriptionReminderLog) error {
	return db.DB().Create(log).Error
}

func (r *SubscriptionRepository) GetReminderLogsBySubscriptionID(db database.DatabaseManager, subscriptionID string) ([]entities.SubscriptionReminderLog, error) {
	var logs []entities.SubscriptionReminderLog
	err := db.DB().
		Where("subscription_id = ?", strings.TrimSpace(subscriptionID)).
		Order("sent_at desc").
		Find(&logs).Error
	if err != nil {
		return nil, err
	}
	return logs, nil
}

func (r *SubscriptionRepository) GetExpiringSubscriptionsForNotification(db database.DatabaseManager, windowDays int) ([]AdminSubscriptionItem, error) {
	var items []AdminSubscriptionItem
	now := time.Now().UTC()
	targetDate := now.AddDate(0, 0, windowDays)

	err := db.DB().Table("subscriptions").
		Joins("LEFT JOIN businesses b ON subscriptions.business_id = b.id").
		Joins("LEFT JOIN businesses a ON subscriptions.aggregator_id = a.id").
		Joins("LEFT JOIN subscription_plans sp ON subscriptions.plan_id = sp.id").
		Where("subscriptions.is_active = ? AND subscriptions.next_billing_date > ? AND subscriptions.next_billing_date <= ?", true, now, targetDate).
		Select(adminSubscriptionSelect).
		Order("subscriptions.next_billing_date ASC").
		Scan(&items).Error
	if err != nil {
		return nil, err
	}
	return items, nil
}

func (r *SubscriptionRepository) HasActiveRenewedSubscription(db database.DatabaseManager, businessID, currentSubscriptionID string, currentNextBillingDate time.Time) (bool, error) {
	var count int64
	err := db.DB().Model(&entities.Subscription{}).
		Where("business_id = ? AND id != ? AND is_active = ? AND next_billing_date > ?", businessID, currentSubscriptionID, true, currentNextBillingDate).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
