package subscription

import (
	"crypto/hmac"
	"crypto/sha512"
	"einvoice-access-point/internal/config"
	"einvoice-access-point/internal/data/dbinit"
	"einvoice-access-point/internal/data/entities"
	"einvoice-access-point/internal/pkg/paystack"
	"einvoice-access-point/internal/utility"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

var (
	ErrPaystackSecretNotConfigured = errors.New("paystack secret key is not configured")
	ErrInvalidPaystackSignature    = errors.New("invalid paystack signature")
)

type paystackSubscriptionMetadata struct {
	IsSandbox    *bool  `json:"is_sandbox"`
	BusinessID   string `json:"business_id"`
	AggregatorID string `json:"aggregator_id"`
	PlanID       string `json:"plan_id"`
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

func (s *Service) CreatePlan(req CreateSubscriptionPlanDto, db *gorm.DB) (*entities.SubscriptionPlan, error) {
	if db == nil {
		return nil, fmt.Errorf("database connection is required")
	}

	pdb := dbinit.InitDB(db, false)
	planName := strings.TrimSpace(req.Name)

	existingPlan, err := s.repo.GetPlanByName(planName, pdb)
	if err != nil {
		return nil, err
	}
	if existingPlan != nil {
		return nil, fmt.Errorf("plan with name %s already exists", planName)
	}

	plan := &entities.SubscriptionPlan{
		ID:            utility.GenerateUUID(),
		Name:          planName,
		Amount:        req.Amount,
		IsActive:      true,
		TotalInvoices: req.TotalInvoices,
		BillingCycle:  req.BillingCycle,
	}

	if err := s.repo.CreatePlan(plan, pdb); err != nil {
		return nil, err
	}

	return plan, nil
}

func (s *Service) SubscribeBusinessToPlan(req AggregatorSubscribeRequestDto, aggregatorID string, isSandbox bool, db *gorm.DB) (fiber.Map, int, error) {
	if db == nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("database connection is required")
	}

	aggregatorID = strings.TrimSpace(aggregatorID)
	businessID := strings.TrimSpace(req.BusinessID)
	planID := strings.TrimSpace(req.PlanID)

	pdb := dbinit.InitDB(db, false)
	business, err := s.businessRepo.GetBusinessByIDForAggregator(pdb, aggregatorID, businessID)
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to fetch business: %w", err)
	}
	if business == nil {
		return nil, http.StatusNotFound, fmt.Errorf("business not found or not managed by this aggregator")
	}

	plan, found, err := s.GetActivePlanByID(planID, db)
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to fetch plan: %w", err)
	}
	if !found {
		return nil, http.StatusBadRequest, fmt.Errorf("invalid or inactive plan id")
	}

	aggregator, err := s.businessRepo.FindUserByID(pdb, aggregatorID)
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to fetch aggregator: %w", err)
	}

	reference := generateTransactionReference()
	transactionRecord := &entities.Transaction{
		ID:           utility.GenerateUUID(),
		BusinessID:   business.ID,
		AggregatorID: aggregatorID,
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
		Email:     strings.ToLower(strings.TrimSpace(aggregator.Email)),
		Amount:    strconv.Itoa(int(math.Round(plan.Amount * 100))),
		Reference: reference,
		Metadata: &paystack.InitializeTransactionMetadata{
			IsSandbox:    isSandbox,
			BusinessID:   business.ID,
			AggregatorID: aggregatorID,
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
		"business_id":       business.ID,
		"plan_id":           plan.ID,
	}

	return response, http.StatusOK, nil
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

	var metadata paystackSubscriptionMetadata
	if metadata.BusinessID != "" {
		transactionRecord.BusinessID = metadata.BusinessID
	}
	if metadata.AggregatorID != "" {
		transactionRecord.AggregatorID = metadata.AggregatorID
	}
	if metadata.PlanID != "" {
		transactionRecord.PlanID = metadata.PlanID
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
		if transactionRecord.BusinessID == "" || transactionRecord.AggregatorID == "" {
			return nil, http.StatusBadRequest, fmt.Errorf("transaction is missing business or aggregator details")
		}

		plan, found, err := s.getTransactionPlan(transactionRecord, db)
		if err != nil {
			return nil, http.StatusInternalServerError, fmt.Errorf("failed to fetch plan: %w", err)
		}
		if !found {
			return nil, http.StatusBadRequest, fmt.Errorf("invalid plan in transaction")
		}

		subscription, err := s.repo.GetLatestSubscriptionByBusinessAndAggregator(pdb, transactionRecord.BusinessID, transactionRecord.AggregatorID)
		if err != nil {
			return nil, http.StatusInternalServerError, fmt.Errorf("failed to fetch subscription: %w", err)
		}

		if subscription == nil {
			subscription = &entities.Subscription{
				ID:           utility.GenerateUUID(),
				BusinessID:   transactionRecord.BusinessID,
				AggregatorID: transactionRecord.AggregatorID,
			}
		}

		subscription.IsActive = true
		subscription.PlanID = plan.ID
		subscription.Plan = plan.Name
		subscription.TotalInvoices = plan.TotalInvoices
		subscription.UsedInvoices = 0
		subscription.RemainingInvoices = plan.TotalInvoices
		subscription.NextBillingDate = time.Now().UTC().AddDate(0, 0, plan.BillingCycle)

		if err := s.repo.SaveSubscription(subscription, pdb); err != nil {
			return nil, http.StatusInternalServerError, fmt.Errorf("failed to update subscription: %w", err)
		}
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
