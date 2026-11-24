package subscription

import (
	"crypto/hmac"
	"crypto/sha512"
	"einvoice-access-point/internal/config"
	"einvoice-access-point/internal/data/database"
	"einvoice-access-point/internal/data/dbinit"
	"einvoice-access-point/internal/data/entities"
	"einvoice-access-point/internal/data/repositories"
	"einvoice-access-point/internal/middleware"
	"einvoice-access-point/internal/pkg/paystack"
	"einvoice-access-point/internal/pkg/resend_email"
	"einvoice-access-point/internal/utility"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

var (
	ErrPaystackSecretNotConfigured = errors.New("paystack secret key is not configured")
	ErrInvalidPaystackSignature    = errors.New("invalid paystack signature")
	directSubscriptionEnforced     atomic.Bool
)

func init() {
	val := strings.TrimSpace(os.Getenv("ENFORCE_DIRECT_SUBSCRIPTIONS"))
	if strings.EqualFold(val, "true") || val == "1" {
		directSubscriptionEnforced.Store(true)
	} else {
		directSubscriptionEnforced.Store(false)
	}
}

func IsDirectSubscriptionEnforced() bool {
	return directSubscriptionEnforced.Load()
}

func SetDirectSubscriptionEnforced(enabled bool) {
	directSubscriptionEnforced.Store(enabled)
}

type paystackSubscriptionMetadata struct {
	IsSandbox    *bool  `json:"is_sandbox"`
	BusinessID   string `json:"business_id"`
	AggregatorID string `json:"aggregator_id"`
	PlanID       string `json:"plan_id"`
}

type Service struct {
	repo         *repositories.SubscriptionRepository
	businessRepo *repositories.BusinessRepository
}

func NewService(repo *repositories.SubscriptionRepository, businessRepo *repositories.BusinessRepository) *Service {
	return &Service{repo: repo, businessRepo: businessRepo}
}

func NewServiceWithDB(prodDb, testDb database.DatabaseManager) *Service {
	repo := repositories.NewSubscriptionRepository(prodDb, testDb)
	businessRepo := repositories.NewBusinessRepository(prodDb, testDb)
	return NewService(repo, businessRepo)
}

func (s *Service) ListPlans(db *gorm.DB) ([]entities.SubscriptionPlan, error) {
	if db == nil {
		return nil, fmt.Errorf("database connection is required")
	}

	pdb := dbinit.InitDB(db, false)
	return s.repo.GetPlans(pdb)
}

func (s *Service) ListActivePlans(db *gorm.DB) ([]entities.SubscriptionPlan, error) {
	plans, err := s.ListPlans(db)
	if err != nil {
		return nil, err
	}

	activePlans := make([]entities.SubscriptionPlan, 0, len(plans))
	for _, plan := range plans {
		if !plan.IsActive {
			continue
		}
		activePlans = append(activePlans, plan)
	}

	return activePlans, nil
}

func (s *Service) GetPlanByName(planName string, db *gorm.DB) (*entities.SubscriptionPlan, bool, error) {
	if db == nil {
		return nil, false, fmt.Errorf("database connection is required")
	}

	pdb := dbinit.InitDB(db, false)
	plan, err := s.repo.GetPlanByName(planName, pdb)
	if err != nil {
		return nil, false, err
	}
	if plan != nil {
		return plan, true, nil
	}
	return nil, false, nil
}

func (s *Service) GetPlanByID(planID string, db *gorm.DB) (*entities.SubscriptionPlan, bool, error) {
	if db == nil {
		return nil, false, fmt.Errorf("database connection is required")
	}

	pdb := dbinit.InitDB(db, false)
	plan, err := s.repo.GetPlanByID(planID, pdb)
	if err != nil {
		return nil, false, err
	}
	if plan != nil {
		return plan, true, nil
	}
	return nil, false, nil
}

func (s *Service) GetActivePlanByName(planName string, db *gorm.DB) (*entities.SubscriptionPlan, bool, error) {
	plan, found, err := s.GetPlanByName(planName, db)
	if err != nil {
		return nil, false, err
	}
	if !found || plan == nil || !plan.IsActive {
		return nil, false, nil
	}

	return plan, true, nil
}

func (s *Service) GetActivePlanByID(planID string, db *gorm.DB) (*entities.SubscriptionPlan, bool, error) {
	plan, found, err := s.GetPlanByID(planID, db)
	if err != nil {
		return nil, false, err
	}
	if !found || plan == nil || !plan.IsActive {
		return nil, false, nil
	}

	return plan, true, nil
}



func (s *Service) InitializeSubscription(req SubscribeRequestDto, userDetails *middleware.UserDataClaims, db *gorm.DB) (fiber.Map, int, error) {
	if db == nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("database connection is required")
	}

	planID := strings.TrimSpace(req.PlanID)
	if planID == "" {
		return nil, http.StatusBadRequest, fmt.Errorf("plan_id is required")
	}

	plan, found, err := s.GetActivePlanByID(planID, db)
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to fetch plan: %w", err)
	}
	if !found {
		return nil, http.StatusBadRequest, fmt.Errorf("invalid or inactive plan id")
	}

	pdb := dbinit.InitDB(db, false)

	var (
		targetBusinessID   string
		targetAggregatorID *string
		payerEmail         string
		metaAggregatorID   string
	)

	if userDetails.IsAggregator {
		// Aggregator subscription on behalf of a managed business
		businessID := strings.TrimSpace(req.BusinessID)
		if businessID == "" {
			return nil, http.StatusUnprocessableEntity, fmt.Errorf("business_id is required for aggregator subscriptions")
		}

		aggregatorID := strings.TrimSpace(userDetails.ID)
		business, err := s.businessRepo.GetBusinessByIDForAggregator(pdb, aggregatorID, businessID)
		if err != nil {
			return nil, http.StatusInternalServerError, fmt.Errorf("failed to fetch business: %w", err)
		}
		if business == nil {
			return nil, http.StatusNotFound, fmt.Errorf("business not found or not managed by this aggregator")
		}

		aggregator, err := s.businessRepo.FindUserByID(pdb, aggregatorID)
		if err != nil {
			return nil, http.StatusInternalServerError, fmt.Errorf("failed to fetch aggregator: %w", err)
		}
		if aggregator == nil {
			return nil, http.StatusNotFound, fmt.Errorf("aggregator user not found")
		}

		targetBusinessID = business.ID
		targetAggregatorID = &aggregatorID
		metaAggregatorID = aggregatorID
		payerEmail = strings.ToLower(strings.TrimSpace(aggregator.Email))
		if payerEmail == "" {
			payerEmail = strings.ToLower(strings.TrimSpace(business.Email))
		}
	} else {
		// Direct business subscription for itself
		businessID := strings.TrimSpace(userDetails.ID)
		if req.BusinessID != "" {
			reqBiz := strings.TrimSpace(req.BusinessID)
			matchesID := reqBiz == businessID
			matchesClaimBiz := userDetails.BusinessID != nil && reqBiz == strings.TrimSpace(*userDetails.BusinessID)
			if !matchesID && !matchesClaimBiz {
				return nil, http.StatusForbidden, fmt.Errorf("cannot subscribe on behalf of another business")
			}
			businessID = reqBiz
		}

		business, err := s.businessRepo.FindUserByID(pdb, businessID)
		if err != nil || business == nil {
			business, err = s.businessRepo.GetBusinessByID(pdb, businessID)
		}
		if (err != nil || business == nil) && strings.TrimSpace(userDetails.ID) != "" && strings.TrimSpace(userDetails.ID) != businessID {
			business, err = s.businessRepo.FindUserByID(pdb, strings.TrimSpace(userDetails.ID))
			if err != nil || business == nil {
				business, err = s.businessRepo.GetBusinessByID(pdb, strings.TrimSpace(userDetails.ID))
			}
		}
		if err != nil {
			return nil, http.StatusInternalServerError, fmt.Errorf("failed to fetch business: %w", err)
		}
		if business == nil {
			return nil, http.StatusNotFound, fmt.Errorf("business not found")
		}

		targetBusinessID = business.ID
		targetAggregatorID = nil
		metaAggregatorID = "" // Explicitly empty so Paystack metadata omits aggregator_id
		payerEmail = strings.ToLower(strings.TrimSpace(business.Email))
		if payerEmail == "" {
			payerEmail = strings.ToLower(strings.TrimSpace(userDetails.Email))
		}
	}

	reference := generateTransactionReference()
	transactionRecord := &entities.Transaction{
		ID:           utility.GenerateUUID(),
		BusinessID:   targetBusinessID,
		AggregatorID: targetAggregatorID,
		Reference:    reference,
		Provider:     "paystack",
		Status:       entities.TransactionStatusInitialized,
		Amount:       plan.Amount,
		Currency:     "NGN",
		PlanID:       plan.ID,
		Plan:         plan.Name,
	}

	if err := s.repo.CreateTransaction(transactionRecord, pdb); err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to create transaction log: %w", err)
	}

	providerResp, err := paystack.InitializeTransaction(paystack.InitializeTransactionRequest{
		Email:     payerEmail,
		Amount:    strconv.Itoa(int(math.Round(plan.Amount * 100))),
		Reference: reference,
		Metadata: &paystack.InitializeTransactionMetadata{
			IsSandbox:    userDetails.IsSandbox,
			BusinessID:   targetBusinessID,
			AggregatorID: metaAggregatorID,
			PlanID:       plan.ID,
		},
	})
	if err != nil {
		errString := err.Error()
		transactionRecord.Status = entities.TransactionStatusFailed
		transactionRecord.ErrorMessage = &errString
		_ = s.repo.SaveTransaction(transactionRecord, pdb)
		return nil, http.StatusBadGateway, fmt.Errorf("failed to initialize paystack transaction: %w", err)
	}

	if !providerResp.Status {
		transactionRecord.Status = entities.TransactionStatusFailed
		transactionRecord.ErrorMessage = &providerResp.Message
		_ = s.repo.SaveTransaction(transactionRecord, pdb)
		return nil, http.StatusBadGateway, fmt.Errorf("paystack initialization failed: %s", providerResp.Message)
	}

	transactionRecord.Status = entities.TransactionStatusProcessing

	if err := s.repo.SaveTransaction(transactionRecord, pdb); err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to update transaction log: %w", err)
	}

	response := fiber.Map{
		"provider":          "paystack",
		"transaction_id":    transactionRecord.ID,
		"transaction_ref":   transactionRecord.Reference,
		"authorization_url": providerResp.Data.AuthorizationURL,
		"business_id":       targetBusinessID,
		"plan_id":           plan.ID,
	}
	if transactionRecord.AggregatorID != nil {
		response["aggregator_id"] = *transactionRecord.AggregatorID
	}

	return response, http.StatusOK, nil
}

// SubscribeBusinessToPlan keeps compatibility for existing aggregator caller signatures
func (s *Service) SubscribeBusinessToPlan(req AggregatorSubscribeRequestDto, aggregatorID string, isSandbox bool, db *gorm.DB) (fiber.Map, int, error) {
	claims := &middleware.UserDataClaims{
		ID:           aggregatorID,
		IsAggregator: true,
		IsSandbox:    isSandbox,
	}
	return s.InitializeSubscription(req, claims, db)
}

func (s *Service) HandlePaystackWebhook(payload *paystack.PaystackWebhookPayload, db *gorm.DB) (fiber.Map, int, error) {
	if payload.Data.Reference == "" {
		return nil, http.StatusBadRequest, fmt.Errorf("reference is required")
	}

	pdb := dbinit.InitDB(db, false)
	transactionRecord, err := s.repo.GetTransactionByReference(payload.Data.Reference, pdb)
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to fetch transaction: %w", err)
	}
	if transactionRecord == nil {
		return nil, http.StatusNotFound, fmt.Errorf("transaction not found")
	}

	previousStatus := transactionRecord.Status

	if payload.Data.Metadata.BusinessID != "" {
		transactionRecord.BusinessID = strings.TrimSpace(payload.Data.Metadata.BusinessID)
	}
	if payload.Data.Metadata.AggregatorID != "" {
		aggID := strings.TrimSpace(payload.Data.Metadata.AggregatorID)
		transactionRecord.AggregatorID = &aggID
	}
	if payload.Data.Metadata.PlanID != "" {
		transactionRecord.PlanID = strings.TrimSpace(payload.Data.Metadata.PlanID)
	}

	rawBody, _ := json.Marshal(payload.Data.Metadata)

	transactionRecord.ProviderResponseMetadata = rawBody
	if payload.Data.Currency != "" {
		transactionRecord.Currency = payload.Data.Currency
	}
	if payload.Data.Amount > 0 {
		transactionRecord.Amount = float64(payload.Data.Amount) / 100.0
	}

	switch {
	case payload.Event == "charge.success" && strings.EqualFold(payload.Data.Status, "success"):
		transactionRecord.Status = entities.TransactionStatusSuccess
	case strings.Contains(strings.ToLower(payload.Event), "failed") || strings.EqualFold(payload.Data.Status, "failed"):
		transactionRecord.Status = entities.TransactionStatusFailed
	default:
		transactionRecord.Status = entities.TransactionStatusProcessing
	}

	if err := s.repo.SaveTransaction(transactionRecord, pdb); err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to update transaction status: %w", err)
	}

	if transactionRecord.Status == "success" && previousStatus != "success" {
		if transactionRecord.BusinessID == "" {
			return nil, http.StatusBadRequest, fmt.Errorf("transaction is missing business details")
		}

		plan, found, err := s.getTransactionPlan(transactionRecord, db)
		if err != nil {
			return nil, http.StatusInternalServerError, fmt.Errorf("failed to fetch plan: %w", err)
		}
		if !found {
			return nil, http.StatusBadRequest, fmt.Errorf("invalid plan in transaction")
		}

		isAggregatorSubscription := transactionRecord.AggregatorID != nil && strings.TrimSpace(*transactionRecord.AggregatorID) != ""
		var subscription *entities.Subscription

		if isAggregatorSubscription {
			aggID := strings.TrimSpace(*transactionRecord.AggregatorID)
			sub, err := s.repo.GetLatestSubscriptionByBusinessAndAggregator(pdb, transactionRecord.BusinessID, aggID)
			if err != nil {
				return nil, http.StatusInternalServerError, fmt.Errorf("failed to fetch subscription: %w", err)
			}
			if sub == nil {
				sub = &entities.Subscription{
					ID:           utility.GenerateUUID(),
					BusinessID:   transactionRecord.BusinessID,
					AggregatorID: &aggID,
				}
			}
			subscription = sub
		} else {
			sub, err := s.repo.GetLatestSubscriptionByBusinessID(pdb, transactionRecord.BusinessID)
			if err != nil {
				return nil, http.StatusInternalServerError, fmt.Errorf("failed to fetch subscription: %w", err)
			}
			if sub == nil {
				sub = &entities.Subscription{
					ID:           utility.GenerateUUID(),
					BusinessID:   transactionRecord.BusinessID,
					AggregatorID: nil,
				}
			}
			subscription = sub
		}

		applyPlanToSubscription(subscription, plan, time.Now().UTC(), false)

		if err := s.repo.SaveSubscription(subscription, pdb); err != nil {
			return nil, http.StatusInternalServerError, fmt.Errorf("failed to update subscription: %w", err)
		}

		receiptAmount := transactionRecord.Amount
		if receiptAmount <= 0 && payload.Data.Amount > 0 {
			receiptAmount = float64(payload.Data.Amount) / 100.0
		}

		aggID := ""
		if transactionRecord.AggregatorID != nil {
			aggID = *transactionRecord.AggregatorID
		}

		// Dispatch payment receipt
		s.dispatchPaymentReceipts(pdb, paymentReceiptInput{
			BusinessID:         transactionRecord.BusinessID,
			AggregatorID:       aggID,
			FallbackPayerEmail: payload.Data.Customer.Email,
			Plan:               plan,
			Reference:          transactionRecord.Reference,
			Channel:            payload.Data.Channel,
			Amount:             receiptAmount,
			PaidAt:             parsePaystackDate(payload.Data.PaidAt),
			NextBillingDate:    subscription.NextBillingDate,
		})
	}

	response := fiber.Map{
		"event":              payload.Event,
		"reference":          payload.Data.Reference,
		"transaction_status": transactionRecord.Status,
	}
	return response, http.StatusOK, nil
}

func (s *Service) RequireAggregatorBusinessSubscription(db *gorm.DB, aggregatorID, businessID string) (*entities.Subscription, int, error) {
	if db == nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("database connection is required")
	}

	aggregatorID = strings.TrimSpace(aggregatorID)
	businessID = strings.TrimSpace(businessID)

	pdb := dbinit.InitDB(db, false)
	business, err := s.businessRepo.GetBusinessByIDForAggregator(pdb, aggregatorID, businessID)
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to fetch business: %w", err)
	}
	if business == nil {
		return nil, http.StatusNotFound, fmt.Errorf("business not found or not managed by this aggregator")
	}

	subscription, err := s.repo.GetLatestSubscriptionByBusinessAndAggregator(pdb, businessID, aggregatorID)
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to fetch subscription: %w", err)
	}
	if subscription == nil {
		return nil, http.StatusForbidden, fmt.Errorf("active subscription required before uploading invoices for this business")
	}
	if !subscription.IsActive {
		return nil, http.StatusForbidden, fmt.Errorf("subscription is inactive for this business")
	}
	if !subscription.NextBillingDate.IsZero() && time.Now().UTC().After(subscription.NextBillingDate) {
		subscription.IsActive = false
		_ = s.repo.SaveSubscription(subscription, pdb)
		return nil, http.StatusForbidden, fmt.Errorf("subscription has expired for this business")
	}

	return subscription, http.StatusOK, nil
}

func (s *Service) ReserveAggregatorInvoiceQuota(db *gorm.DB, aggregatorID, businessID string, count int) (string, int, error) {
	subscription, status, err := s.RequireAggregatorBusinessSubscription(db, aggregatorID, businessID)
	if err != nil {
		return "", status, err
	}
	if count <= 0 {
		return subscription.ID, http.StatusOK, nil
	}
	if subscription.RemainingInvoices < count {
		return "", http.StatusForbidden, fmt.Errorf("subscription invoice limit exhausted for this business")
	}

	pdb := dbinit.InitDB(db, false)
	reserved, err := s.repo.ReserveSubscriptionInvoices(pdb, subscription.ID, count)
	if err != nil {
		return "", http.StatusInternalServerError, fmt.Errorf("failed to reserve subscription quota: %w", err)
	}
	if !reserved {
		return "", http.StatusForbidden, fmt.Errorf("subscription invoice limit exhausted for this business")
	}

	return subscription.ID, http.StatusOK, nil
}

func (s *Service) ReleaseReservedInvoices(db *gorm.DB, subscriptionID string, count int) error {
	if db == nil || subscriptionID == "" || count <= 0 {
		return nil
	}

	pdb := dbinit.InitDB(db, false)
	return s.repo.ReleaseSubscriptionInvoices(pdb, subscriptionID, count)
}

// RecordManualSubscription activates (or extends) a business subscription for a payment
// received outside Paystack, e.g. a bank transfer confirmed by a Super Admin.
// The payment is recorded as a successful "manual" transaction for auditability.
func (s *Service) RecordManualSubscription(db database.DatabaseManager, in ManualSubscriptionInput) (*ManualSubscriptionResult, int, error) {
	if db == nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("database connection is required")
	}

	businessID := strings.TrimSpace(in.BusinessID)
	planID := strings.TrimSpace(in.PlanID)
	paymentRef := strings.TrimSpace(in.PaymentReference)
	if businessID == "" || planID == "" || paymentRef == "" {
		return nil, http.StatusBadRequest, fmt.Errorf("business_id, plan_id and payment_reference are required")
	}

	business, err := s.businessRepo.GetBusinessByID(db, businessID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, http.StatusNotFound, fmt.Errorf("business not found")
		}
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to fetch business: %w", err)
	}
	if business.IsAggregator {
		return nil, http.StatusBadRequest, fmt.Errorf("subscriptions are assigned to managed businesses, not aggregators")
	}
	var aggregatorID *string
	if business.AggregatorID != nil && strings.TrimSpace(*business.AggregatorID) != "" {
		trimmed := strings.TrimSpace(*business.AggregatorID)
		aggregatorID = &trimmed
	}

	plan, err := s.repo.GetPlanByID(planID, db)
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to fetch plan: %w", err)
	}
	if plan == nil {
		return nil, http.StatusNotFound, fmt.Errorf("subscription plan not found")
	}
	if !plan.IsActive {
		return nil, http.StatusBadRequest, fmt.Errorf("subscription plan is inactive")
	}

	reference := manualReferencePrefix + paymentRef
	existing, err := s.repo.GetTransactionByReference(reference, db)
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to check payment reference: %w", err)
	}
	if existing != nil {
		return nil, http.StatusConflict, fmt.Errorf("payment reference %q has already been recorded", paymentRef)
	}

	now := time.Now().UTC()

	amount := plan.Amount
	if in.AmountPaid != nil {
		amount = *in.AmountPaid
	}

	paymentDate := now
	if in.PaymentDate != nil && !in.PaymentDate.IsZero() {
		paymentDate = in.PaymentDate.UTC()
		if paymentDate.After(now.Add(24 * time.Hour)) {
			return nil, http.StatusBadRequest, fmt.Errorf("payment_date cannot be in the future")
		}
	}

	paymentMethod := strings.TrimSpace(in.PaymentMethod)
	if paymentMethod == "" {
		paymentMethod = "bank_transfer"
	}

	metadata, _ := json.Marshal(map[string]any{
		"source":            "admin_manual",
		"recorded_by_id":    in.AdminID,
		"recorded_by_email": in.AdminEmail,
		"payment_reference": paymentRef,
		"payment_method":    paymentMethod,
		"payment_date":      paymentDate.Format(time.RFC3339),
		"notes":             strings.TrimSpace(in.Notes),
		"recorded_at":       now.Format(time.RFC3339),
	})

	transactionRecord := &entities.Transaction{
		ID:                       utility.GenerateUUID(),
		BusinessID:               business.ID,
		AggregatorID:             aggregatorID,
		Reference:                reference,
		Provider:                 ManualPaymentProvider,
		Status:                   entities.TransactionStatusSuccess,
		Amount:                   amount,
		Currency:                 "NGN",
		PlanID:                   plan.ID,
		Plan:                     plan.Name,
		ProviderResponseMetadata: metadata,
	}

	var subscription *entities.Subscription
	renewed := false

	err = db.DB().Transaction(func(tx *gorm.DB) error {
		tdb := dbinit.InitDB(tx, false)

		if err := s.repo.CreateTransaction(transactionRecord, tdb); err != nil {
			return fmt.Errorf("failed to record transaction: %w", err)
		}

		var sub *entities.Subscription
		if aggregatorID != nil {
			sub, err = s.repo.GetLatestSubscriptionByBusinessAndAggregator(tdb, business.ID, *aggregatorID)
		} else {
			sub, err = s.repo.GetLatestSubscriptionByBusinessID(tdb, business.ID)
		}
		if err != nil {
			return fmt.Errorf("failed to fetch subscription: %w", err)
		}
		if sub == nil {
			sub = &entities.Subscription{
				ID:           utility.GenerateUUID(),
				BusinessID:   business.ID,
				AggregatorID: aggregatorID,
			}
		}

		renewed = applyPlanToSubscription(sub, plan, now, true)

		if err := s.repo.SaveSubscription(sub, tdb); err != nil {
			return fmt.Errorf("failed to save subscription: %w", err)
		}
		subscription = sub
		return nil
	})
	if err != nil {
		// A concurrent request may have inserted the same reference between our check and insert.
		if errors.Is(err, gorm.ErrDuplicatedKey) || strings.Contains(strings.ToLower(err.Error()), "duplicate key") {
			return nil, http.StatusConflict, fmt.Errorf("payment reference %q has already been recorded", paymentRef)
		}
		return nil, http.StatusInternalServerError, err
	}

	aggIDResult := ""
	if aggregatorID != nil {
		aggIDResult = *aggregatorID
	}

	if in.SendReceipt {
		s.dispatchPaymentReceipts(db, paymentReceiptInput{
			BusinessID:      business.ID,
			AggregatorID:    aggIDResult,
			Plan:            plan,
			Reference:       reference,
			Channel:         paymentMethod,
			Amount:          amount,
			PaidAt:          paymentDate,
			NextBillingDate: subscription.NextBillingDate,
		})
	}

	return &ManualSubscriptionResult{
		SubscriptionID:    subscription.ID,
		TransactionID:     transactionRecord.ID,
		Reference:         reference,
		BusinessID:        business.ID,
		AggregatorID:      aggIDResult,
		PlanID:            plan.ID,
		Plan:              plan.Name,
		AmountPaid:        amount,
		PaymentMethod:     paymentMethod,
		TotalInvoices:     subscription.TotalInvoices,
		RemainingInvoices: subscription.RemainingInvoices,
		NextBillingDate:   subscription.NextBillingDate,
		Renewed:           renewed,
	}, http.StatusCreated, nil
}

// applyPlanToSubscription activates sub on plan as of now.
// When carryOver is true and sub is still active, the new cycle is appended to the current
// expiry and unused invoices are carried forward; otherwise the cycle starts now with a fresh quota.
// It reports whether an active subscription was extended.
func applyPlanToSubscription(sub *entities.Subscription, plan *entities.SubscriptionPlan, now time.Time, carryOver bool) bool {
	start := now
	quota := plan.TotalInvoices
	extended := false

	if carryOver && sub.IsActive && sub.NextBillingDate.After(now) {
		start = sub.NextBillingDate
		if sub.RemainingInvoices > 0 {
			quota += sub.RemainingInvoices
		}
		extended = true
	}

	sub.IsActive = true
	sub.PlanID = plan.ID
	sub.Plan = plan.Name
	sub.TotalInvoices = quota
	sub.UsedInvoices = 0
	sub.RemainingInvoices = quota
	sub.NextBillingDate = start.AddDate(0, 0, plan.BillingCycle)
	sub.RemindersSentCount = 0
	sub.LastReminderSentAt = nil

	return extended
}

type paymentReceiptInput struct {
	BusinessID         string
	AggregatorID       string
	FallbackPayerEmail string
	Plan               *entities.SubscriptionPlan
	Reference          string
	Channel            string
	Amount             float64
	PaidAt             time.Time
	NextBillingDate    time.Time
}

// dispatchPaymentReceipts asynchronously emails a payment receipt to the payer
// and, if different and applicable, to the covered business.
func (s *Service) dispatchPaymentReceipts(db database.DatabaseManager, in paymentReceiptInput) {
	bizName, bizEmail := "", ""
	if biz, _ := s.businessRepo.GetBusinessByID(db, in.BusinessID); biz != nil {
		bizName = biz.CompanyName
		if bizName == "" {
			bizName = biz.Name
		}
		bizEmail = strings.TrimSpace(biz.Email)
	}

	planName, totalInvoices, billingCycle := in.Plan.Name, in.Plan.TotalInvoices, in.Plan.BillingCycle

	if strings.TrimSpace(in.AggregatorID) == "" {
		// Direct business subscription: receipt sent directly to business
		targetEmail := bizEmail
		if targetEmail == "" {
			targetEmail = strings.TrimSpace(in.FallbackPayerEmail)
		}
		if targetEmail != "" {
			recipient := bizName
			if recipient == "" {
				recipient = "Valued Customer"
			}
			go func() {
				_ = resend_email.SendSubscriptionPaymentReceiptEmail(
					targetEmail, recipient, planName, bizName, in.Reference, in.Channel,
					in.Amount, totalInvoices, billingCycle, in.PaidAt, in.NextBillingDate,
				)
			}()
		}
		return
	}

	agg, _ := s.businessRepo.FindUserByID(db, in.AggregatorID)
	if agg == nil {
		agg, _ = s.businessRepo.GetBusinessByID(db, in.AggregatorID)
	}
	aggName, aggEmail := "", ""
	if agg != nil {
		aggName = agg.CompanyName
		if aggName == "" {
			aggName = agg.Name
		}
		aggEmail = strings.TrimSpace(agg.Email)
	}
	if aggEmail == "" {
		aggEmail = strings.TrimSpace(in.FallbackPayerEmail)
	}

	go func() {
		// Send official payment receipt to the payer (aggregator)
		if aggEmail != "" {
			recipient := aggName
			if recipient == "" {
				recipient = "Valued Customer"
			}
			_ = resend_email.SendSubscriptionPaymentReceiptEmail(
				aggEmail, recipient, planName, bizName, in.Reference, in.Channel,
				in.Amount, totalInvoices, billingCycle, in.PaidAt, in.NextBillingDate,
			)
		}
		// Also notify covered business if distinct email exists
		if bizEmail != "" && !strings.EqualFold(bizEmail, aggEmail) {
			recipient := bizName
			if recipient == "" {
				recipient = "Valued Business"
			}
			_ = resend_email.SendSubscriptionPaymentReceiptEmail(
				bizEmail, recipient, planName, bizName, in.Reference, in.Channel,
				in.Amount, totalInvoices, billingCycle, in.PaidAt, in.NextBillingDate,
			)
		}
	}()
}

func (s *Service) ValidatePaystackSignature(rawBody []byte, signature string) error {
	cfg := config.GetConfig()
	if cfg.Paystack.SecretKey == "" {
		return ErrPaystackSecretNotConfigured
	}

	hash := hmac.New(sha512.New, []byte(cfg.Paystack.SecretKey))
	hash.Write(rawBody)
	expectedSignature := hex.EncodeToString(hash.Sum(nil))
	if !hmac.Equal([]byte(expectedSignature), []byte(signature)) {
		return ErrInvalidPaystackSignature
	}

	return nil
}

func (s *Service) getTransactionPlan(transactionRecord *entities.Transaction, db *gorm.DB) (*entities.SubscriptionPlan, bool, error) {
	if transactionRecord.PlanID != "" {
		plan, found, err := s.GetPlanByID(transactionRecord.PlanID, db)
		if err != nil {
			return nil, false, err
		}
		if found {
			return plan, true, nil
		}
	}

	if strings.TrimSpace(transactionRecord.Plan) == "" {
		return nil, false, nil
	}

	return s.GetPlanByName(transactionRecord.Plan, db)
}

func generateTransactionReference() string {
	return fmt.Sprintf(
		"aggsub_%d_%s",
		time.Now().UTC().UnixNano(),
		strings.ReplaceAll(utility.GenerateUUID(), "-", ""),
	)
}

func parsePaystackDate(ts string) time.Time {
	ts = strings.TrimSpace(ts)
	if ts == "" {
		return time.Now().UTC()
	}
	for _, layout := range []string{
		time.RFC3339,
		"2006-01-02T15:04:05.000Z",
		"2006-01-02T15:04:05Z",
		"2006-01-02 15:04:05",
	} {
		if t, err := time.Parse(layout, ts); err == nil {
			return t.UTC()
		}
	}
	return time.Now().UTC()
}

func (s *Service) GetBusinessSubscriptionStatus(db *gorm.DB, businessID string) (*BusinessSubscriptionStatusDto, int, error) {
	if db == nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("database connection is required")
	}

	businessID = strings.TrimSpace(businessID)
	pdb := dbinit.InitDB(db, false)

	sub, err := s.repo.GetLatestSubscriptionByBusinessID(pdb, businessID)
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to fetch subscription: %w", err)
	}

	if sub == nil {
		return &BusinessSubscriptionStatusDto{
			HasActiveSubscription: false,
		}, http.StatusOK, nil
	}

	now := time.Now().UTC()
	isActive := sub.IsActive && !sub.NextBillingDate.IsZero() && sub.NextBillingDate.After(now)
	daysRemaining := 0
	if isActive && sub.NextBillingDate.After(now) {
		daysRemaining = int(math.Ceil(sub.NextBillingDate.Sub(now).Hours() / 24))
	}

	var nextBillingDate *time.Time
	if !sub.NextBillingDate.IsZero() {
		nextBillingDate = &sub.NextBillingDate
	}

	return &BusinessSubscriptionStatusDto{
		HasActiveSubscription: isActive,
		SubscriptionID:        sub.ID,
		PlanID:                sub.PlanID,
		Plan:                  sub.Plan,
		TotalInvoices:         sub.TotalInvoices,
		RemainingInvoices:     sub.RemainingInvoices,
		UsedInvoices:          sub.UsedInvoices,
		NextBillingDate:       nextBillingDate,
		DaysRemaining:         daysRemaining,
	}, http.StatusOK, nil
}

func (s *Service) RequireBusinessSubscription(db *gorm.DB, businessID string) (*entities.Subscription, int, error) {
	if db == nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("database connection is required")
	}

	businessID = strings.TrimSpace(businessID)
	pdb := dbinit.InitDB(db, false)

	subscription, err := s.repo.GetLatestSubscriptionByBusinessID(pdb, businessID)
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to fetch subscription: %w", err)
	}
	if subscription == nil {
		return nil, http.StatusForbidden, fmt.Errorf("active subscription required before uploading invoices")
	}
	if !subscription.IsActive {
		return nil, http.StatusForbidden, fmt.Errorf("subscription is inactive")
	}
	if !subscription.NextBillingDate.IsZero() && time.Now().UTC().After(subscription.NextBillingDate) {
		subscription.IsActive = false
		_ = s.repo.SaveSubscription(subscription, pdb)
		return nil, http.StatusForbidden, fmt.Errorf("subscription has expired")
	}
	if subscription.RemainingInvoices <= 0 {
		return nil, http.StatusForbidden, fmt.Errorf("subscription invoice quota exhausted")
	}

	return subscription, http.StatusOK, nil
}

func (s *Service) ReserveBusinessSubscriptionInvoice(db *gorm.DB, subscriptionID string) (bool, error) {
	if db == nil || strings.TrimSpace(subscriptionID) == "" {
		return false, fmt.Errorf("database connection and subscription id are required")
	}
	pdb := dbinit.InitDB(db, false)
	return s.repo.ReserveSubscriptionInvoices(pdb, strings.TrimSpace(subscriptionID), 1)
}

func (s *Service) GetActiveBusinessSubscription(db *gorm.DB, businessID string) (*entities.Subscription, error) {
	if db == nil {
		return nil, fmt.Errorf("database connection is required")
	}

	businessID = strings.TrimSpace(businessID)
	if businessID == "" {
		return nil, nil
	}
	pdb := dbinit.InitDB(db, false)

	sub, err := s.repo.GetLatestSubscriptionByBusinessID(pdb, businessID)
	if err != nil {
		return nil, err
	}
	if sub == nil {
		return nil, nil
	}

	now := time.Now().UTC()
	if !sub.IsActive {
		return nil, nil
	}
	if !sub.NextBillingDate.IsZero() && now.After(sub.NextBillingDate) {
		sub.IsActive = false
		_ = s.repo.SaveSubscription(sub, pdb)
		return nil, nil
	}

	return sub, nil
}

func (s *Service) ReserveBusinessInvoiceQuota(db *gorm.DB, businessID string, count int) (string, int, error) {
	subscription, status, err := s.RequireBusinessSubscription(db, businessID)
	if err != nil {
		return "", status, err
	}
	if count <= 0 {
		return subscription.ID, http.StatusOK, nil
	}
	if subscription.RemainingInvoices < count {
		return "", http.StatusForbidden, fmt.Errorf("subscription invoice quota exhausted")
	}

	pdb := dbinit.InitDB(db, false)
	reserved, err := s.repo.ReserveSubscriptionInvoices(pdb, subscription.ID, count)
	if err != nil {
		return "", http.StatusInternalServerError, fmt.Errorf("failed to reserve subscription quota: %w", err)
	}
	if !reserved {
		return "", http.StatusForbidden, fmt.Errorf("subscription invoice quota exhausted")
	}

	return subscription.ID, http.StatusOK, nil
}


