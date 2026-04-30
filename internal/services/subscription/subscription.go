package subscription

import (
	"crypto/hmac"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"einvoice-access-point/external/paystack"
	"einvoice-access-point/internal/dtos"
	aggregatorRepo "einvoice-access-point/internal/repository/aggregator"
	businessRepo "einvoice-access-point/internal/repository/business"
	planRepo "einvoice-access-point/internal/repository/plan"
	subscriptionRepo "einvoice-access-point/internal/repository/subscription"
	transactionRepo "einvoice-access-point/internal/repository/transaction"
	"einvoice-access-point/pkg/config"
	inst "einvoice-access-point/pkg/dbinit"
	"einvoice-access-point/pkg/models"
	"einvoice-access-point/pkg/utility"

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
	if !found || plan == nil || !plan.IsActive {
		return nil, false, nil
	}

	return plan, true, nil
}

func GetActivePlanByID(planID string, db *gorm.DB) (*models.SubscriptionPlan, bool, error) {
	plan, found, err := GetPlanByID(planID, db)
	if err != nil {
		return nil, false, err
	}
	if !found || plan == nil || !plan.IsActive {
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

func SubscribeBusinessToPlan(req dtos.AggregatorSubscribeRequestDto, aggregatorID string, isSandbox bool, db *gorm.DB) (fiber.Map, int, error) {
	if db == nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("database connection is required")
	}

	aggregatorID = strings.TrimSpace(aggregatorID)
	businessID := strings.TrimSpace(req.BusinessID)
	planID := strings.TrimSpace(req.PlanID)

	business, err := aggregatorRepo.GetBusinessByIDForAggregator(db, aggregatorID, businessID)
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to fetch business: %w", err)
	}
	if business == nil {
		return nil, http.StatusNotFound, fmt.Errorf("business not found or not managed by this aggregator")
	}

	plan, found, err := GetActivePlanByID(planID, db)
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to fetch plan: %w", err)
	}
	if !found {
		return nil, http.StatusBadRequest, fmt.Errorf("invalid or inactive plan id")
	}

	pdb := inst.InitDB(db, false)
	aggregator, err := businessRepo.FindUserByID(pdb, aggregatorID)
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to fetch aggregator: %w", err)
	}

	reference := generateTransactionReference()
	transactionRecord := &models.Transaction{
		ID:           utility.GenerateUUID(),
		BusinessID:   business.ID,
		AggregatorID: aggregatorID,
		Reference:    reference,
		Provider:     "paystack",
		Status:       models.TransactionStatusInitialized,
		Amount:       plan.Amount,
		Currency:     "NGN",
		PlanID:       plan.ID,
		Plan:         plan.Name,
	}

	if err := transactionRepo.CreateTransaction(transactionRecord, pdb); err != nil {
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
		transactionRecord.Status = models.TransactionStatusFailed
		transactionRecord.ErrorMessage = &errString
		_ = transactionRepo.SaveTransaction(transactionRecord, pdb)
		return nil, http.StatusBadGateway, fmt.Errorf("failed to initialize paystack transaction: %w", err)
	}

	if !providerResp.Status {
		transactionRecord.Status = models.TransactionStatusFailed
		transactionRecord.ErrorMessage = &providerResp.Message
		_ = transactionRepo.SaveTransaction(transactionRecord, pdb)
		return nil, http.StatusBadGateway, fmt.Errorf("paystack initialization failed: %s", providerResp.Message)
	}

	transactionRecord.Status = models.TransactionStatusProcessing

	if err := transactionRepo.SaveTransaction(transactionRecord, pdb); err != nil {
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

func HandlePaystackWebhook(payload *paystack.PaystackWebhookPayload, db *gorm.DB) (fiber.Map, int, error) {

	if payload.Data.Reference == "" {
		return nil, http.StatusBadRequest, fmt.Errorf("reference is required")
	}

	pdb := inst.InitDB(db, false)
	transactionRecord, err := transactionRepo.GetTransactionByReference(payload.Data.Reference, pdb)
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
		transactionRecord.Status = models.TransactionStatusSuccess
	case strings.Contains(strings.ToLower(payload.Event), "failed") || strings.EqualFold(payload.Data.Status, "failed"):
		transactionRecord.Status = models.TransactionStatusFailed
	default:
		transactionRecord.Status = models.TransactionStatusProcessing
	}

	if err := transactionRepo.SaveTransaction(transactionRecord, pdb); err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to update transaction status: %w", err)
	}

	if transactionRecord.Status == "success" && previousStatus != "success" {
		if transactionRecord.BusinessID == "" || transactionRecord.AggregatorID == "" {
			return nil, http.StatusBadRequest, fmt.Errorf("transaction is missing business or aggregator details")
		}

		plan, found, err := getTransactionPlan(transactionRecord, db)
		if err != nil {
			return nil, http.StatusInternalServerError, fmt.Errorf("failed to fetch plan: %w", err)
		}
		if !found {
			return nil, http.StatusBadRequest, fmt.Errorf("invalid plan in transaction")
		}

		subscription, err := subscriptionRepo.GetLatestSubscriptionByBusinessAndAggregator(pdb, transactionRecord.BusinessID, transactionRecord.AggregatorID)
		if err != nil {
			return nil, http.StatusInternalServerError, fmt.Errorf("failed to fetch subscription: %w", err)
		}

		if subscription == nil {
			subscription = &models.Subscription{
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

		if err := subscriptionRepo.SaveSubscription(subscription, pdb); err != nil {
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

func RequireAggregatorBusinessSubscription(db *gorm.DB, aggregatorID, businessID string) (*models.Subscription, int, error) {
	if db == nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("database connection is required")
	}

	aggregatorID = strings.TrimSpace(aggregatorID)
	businessID = strings.TrimSpace(businessID)

	business, err := aggregatorRepo.GetBusinessByIDForAggregator(db, aggregatorID, businessID)
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to fetch business: %w", err)
	}
	if business == nil {
		return nil, http.StatusNotFound, fmt.Errorf("business not found or not managed by this aggregator")
	}

	pdb := inst.InitDB(db, false)
	subscription, err := subscriptionRepo.GetLatestSubscriptionByBusinessAndAggregator(pdb, businessID, aggregatorID)
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
		_ = subscriptionRepo.SaveSubscription(subscription, pdb)
		return nil, http.StatusForbidden, fmt.Errorf("subscription has expired for this business")
	}

	return subscription, http.StatusOK, nil
}

func ReserveAggregatorInvoiceQuota(db *gorm.DB, aggregatorID, businessID string, count int) (string, int, error) {
	subscription, status, err := RequireAggregatorBusinessSubscription(db, aggregatorID, businessID)
	if err != nil {
		return "", status, err
	}
	if count <= 0 {
		return subscription.ID, http.StatusOK, nil
	}
	if subscription.RemainingInvoices < count {
		return "", http.StatusForbidden, fmt.Errorf("subscription invoice limit exhausted for this business")
	}

	pdb := inst.InitDB(db, false)
	reserved, err := subscriptionRepo.ReserveSubscriptionInvoices(pdb, subscription.ID, count)
	if err != nil {
		return "", http.StatusInternalServerError, fmt.Errorf("failed to reserve subscription quota: %w", err)
	}
	if !reserved {
		return "", http.StatusForbidden, fmt.Errorf("subscription invoice limit exhausted for this business")
	}

	return subscription.ID, http.StatusOK, nil
}

func ReleaseReservedInvoices(db *gorm.DB, subscriptionID string, count int) error {
	if db == nil || subscriptionID == "" || count <= 0 {
		return nil
	}

	pdb := inst.InitDB(db, false)
	return subscriptionRepo.ReleaseSubscriptionInvoices(pdb, subscriptionID, count)
}

func ValidatePaystackSignature(rawBody []byte, signature string) error {
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

func getTransactionPlan(transactionRecord *models.Transaction, db *gorm.DB) (*models.SubscriptionPlan, bool, error) {
	if transactionRecord.PlanID != "" {
		plan, found, err := GetPlanByID(transactionRecord.PlanID, db)
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

	return GetPlanByName(transactionRecord.Plan, db)
}

func generateTransactionReference() string {
	return fmt.Sprintf(
		"aggsub_%d_%s",
		time.Now().UTC().UnixNano(),
		strings.ReplaceAll(utility.GenerateUUID(), "-", ""),
	)
}
