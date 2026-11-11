package consumer

import (
	"context"
	"einvoice-access-point/internal/data/database"
	"einvoice-access-point/internal/data/dbinit"
	"einvoice-access-point/internal/data/entities"
	"einvoice-access-point/internal/data/repositories"
	"einvoice-access-point/internal/pkg/resend_email"
	"einvoice-access-point/internal/utility"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/hibiken/asynq"
	"gorm.io/gorm"
)

const TypeSubscriptionCheckExpiring = "subscription:check_expiring"

// Standard milestone thresholds for expiration reminders (in days before NextBillingDate)
var defaultReminderDays = []int{7, 3, 1}

const maxRemindersPerCycle = 3

type SubscriptionConsumer struct {
	db      *database.Database
	testDB  *database.Database
	logger  *utility.Logger
	subRepo *repositories.SubscriptionRepository
	bizRepo *repositories.BusinessRepository
}

func NewSubscriptionConsumer(db, testDB *database.Database, logger *utility.Logger, subRepo *repositories.SubscriptionRepository, bizRepo *repositories.BusinessRepository) *SubscriptionConsumer {
	return &SubscriptionConsumer{
		db:      db,
		testDB:  testDB,
		logger:  logger,
		subRepo: subRepo,
		bizRepo: bizRepo,
	}
}

func (c *SubscriptionConsumer) HandleCheckExpiringSubscriptionsTask(ctx context.Context, t *asynq.Task) error {
	c.logger.Info("Starting subscription expiration reminder background task")

	if c.db != nil {
		prodDB := dbinit.InitDB(c.db.Postgresql.DB(), false)
		if err := c.processExpiringSubscriptionsForDB(prodDB); err != nil {
			c.logger.Error("Failed to process expiring subscriptions for production DB", "error", err)
		}
	}

	if c.testDB != nil {
		testDB := dbinit.InitDB(c.testDB.Postgresql.DB(), false)
		if err := c.processExpiringSubscriptionsForDB(testDB); err != nil {
			c.logger.Error("Failed to process expiring subscriptions for sandbox DB", "error", err)
		}
	}

	c.logger.Info("Completed subscription expiration reminder background task")
	return nil
}

func (c *SubscriptionConsumer) processExpiringSubscriptionsForDB(db database.DatabaseManager) error {
	// Query up to 90 days ahead to support long-range reminder milestones (e.g. annual plan 30-day reminders)
	maxScanDays := 90

	subscriptions, err := c.subRepo.GetExpiringSubscriptionsForNotification(db, maxScanDays)
	if err != nil {
		return fmt.Errorf("failed to fetch expiring subscriptions: %w", err)
	}

	now := time.Now().UTC()
	for _, sub := range subscriptions {
		daysRemaining := int(math.Ceil(sub.NextBillingDate.Sub(now).Hours() / 24))

		// If already past next billing date, mark inactive
		if sub.NextBillingDate.Before(now) {
			_ = db.DB().Model(&entities.Subscription{}).
				Where("id = ?", sub.ID).
				Update("is_active", false)
			continue
		}

		// Check if the business has already renewed or has another active subscription with a later billing date
		hasRenewed, err := c.subRepo.HasActiveRenewedSubscription(db, sub.BusinessID, sub.ID, sub.NextBillingDate)
		if err != nil {
			c.logger.Error("Failed to check active renewed subscription", "business_id", sub.BusinessID, "error", err)
		}
		if hasRenewed {
			c.logger.Info("Skipping reminder: business has active renewed subscription", "business_id", sub.BusinessID, "subscription_id", sub.ID)
			continue
		}

		// Use the subscription plan's configured reminder schedule (or default to [7, 3, 1])
		reminderDays := sub.GetReminderDays()
		if len(reminderDays) == 0 {
			reminderDays = defaultReminderDays
		}

		// Check if max allowed reminders reached for this billing cycle
		if sub.RemindersSentCount >= len(reminderDays) {
			continue
		}

		// Prevent multiple reminders within 20 hours
		if sub.LastReminderSentAt != nil && now.Sub(*sub.LastReminderSentAt) < 20*time.Hour {
			continue
		}

		// Verify if current days remaining matches or has crossed the next reminder milestone
		targetThreshold := reminderDays[sub.RemindersSentCount]
		if daysRemaining > targetThreshold {
			continue
		}

		// Check business notification preference
		pref, _ := c.bizRepo.GetNotificationPreference(db, sub.BusinessID)
		allowBusinessNotification := pref == nil || pref.SubscriptionExpiring

		reminderDispatched := false

		// Notify Business
		if allowBusinessNotification && strings.TrimSpace(sub.BusinessEmail) != "" {
			bizName := sub.BusinessCompany
			if bizName == "" {
				bizName = sub.BusinessName
			}
			sendErr := resend_email.SendSubscriptionExpiringReminderEmail(
				sub.BusinessEmail,
				bizName,
				sub.Plan,
				daysRemaining,
				sub.RemainingInvoices,
				sub.NextBillingDate,
				sub.PlanAmount,
				sub.PlanBillingCycle,
				sub.ID,
			)

			status := "sent"
			var errMsg *string
			if sendErr != nil {
				status = "failed"
				msg := sendErr.Error()
				errMsg = &msg
				c.logger.Error("Failed to send subscription reminder to business", "email", sub.BusinessEmail, "error", sendErr)
			} else {
				reminderDispatched = true
			}

			_ = c.subRepo.CreateReminderLog(db, &entities.SubscriptionReminderLog{
				SubscriptionID:   sub.ID,
				BusinessID:       sub.BusinessID,
				AggregatorID:     sub.AggregatorID,
				RecipientEmail:   sub.BusinessEmail,
				RecipientType:    "business",
				ReminderNumber:   sub.RemindersSentCount + 1,
				DaysBeforeExpiry: daysRemaining,
				NextBillingDate:  sub.NextBillingDate,
				Status:           status,
				ErrorMessage:     errMsg,
			})
		}

		// Notify Aggregator if business is managed by one
		if strings.TrimSpace(sub.AggregatorEmail) != "" {
			aggName := sub.AggregatorCompany
			if aggName == "" {
				aggName = sub.AggregatorName
			}
			sendErr := resend_email.SendSubscriptionExpiringReminderEmail(
				sub.AggregatorEmail,
				aggName,
				sub.Plan,
				daysRemaining,
				sub.RemainingInvoices,
				sub.NextBillingDate,
				sub.PlanAmount,
				sub.PlanBillingCycle,
				sub.ID,
			)

			status := "sent"
			var errMsg *string
			if sendErr != nil {
				status = "failed"
				msg := sendErr.Error()
				errMsg = &msg
				c.logger.Error("Failed to send subscription reminder to aggregator", "email", sub.AggregatorEmail, "error", sendErr)
			} else {
				reminderDispatched = true
			}

			_ = c.subRepo.CreateReminderLog(db, &entities.SubscriptionReminderLog{
				SubscriptionID:   sub.ID,
				BusinessID:       sub.BusinessID,
				AggregatorID:     sub.AggregatorID,
				RecipientEmail:   sub.AggregatorEmail,
				RecipientType:    "aggregator",
				ReminderNumber:   sub.RemindersSentCount + 1,
				DaysBeforeExpiry: daysRemaining,
				NextBillingDate:  sub.NextBillingDate,
				Status:           status,
				ErrorMessage:     errMsg,
			})
		}

		if reminderDispatched {
			_ = db.DB().Model(&entities.Subscription{}).
				Where("id = ?", sub.ID).
				Updates(map[string]interface{}{
					"reminders_sent_count":  gorm.Expr("reminders_sent_count + 1"),
					"last_reminder_sent_at": now,
				})
			c.logger.Info("Subscription reminder dispatched", "subscription_id", sub.ID, "days_remaining", daysRemaining)
		}
	}

	return nil
}
