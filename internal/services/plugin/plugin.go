package plugin

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
	smeRepo "einvoice-access-point/internal/repository/sme"
	subscriptionRepo "einvoice-access-point/internal/repository/subscription"
	transactionRepo "einvoice-access-point/internal/repository/transaction"
	subscriptionService "einvoice-access-point/internal/services/subscription"
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

func ValidateCreateUserRequest(req dtos.SmeRegistrationDto, db *gorm.DB) (dtos.SmeRegistrationDto, error) {

	pdb := inst.InitDB(db, false)
	sme := models.SME{}

	if req.Email != "" {
		req.Email = strings.ToLower(req.Email)
		formattedMail, checkBool := utility.EmailValid(req.Email)
		if !checkBool {
			return req, fmt.Errorf("email address is invalid")
		}
		req.Email = formattedMail
		exists := pdb.CheckExists(&sme, "email = ?", req.Email)
		if exists {
			return req, errors.New("user already exists with the given email")
		}
	}
	if exists := pdb.CheckExists(&sme, "company_name = ?", req.CompanyName); exists {
		return req, errors.New("Business already exists with the given company name")
	}

	return req, nil
}

func CheckBusinessWithSubscription(email, aggregatorId string, db *gorm.DB) (fiber.Map, int, error) {
	pdb := inst.InitDB(db, false)

	email = strings.ToLower(strings.TrimSpace(email))
	formattedEmail, ok := utility.EmailValid(email)
	if !ok {
		return nil, http.StatusBadRequest, fmt.Errorf("email address is invalid")
	}

	sme, err := smeRepo.FindSMEByAggregatorId(pdb, aggregatorId, formattedEmail)
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to fetch business: %w", err)
	}

	var response fiber.Map
	if sme == nil {
		response = fiber.Map{
			"exists":              false,
			"active_subscription": false,
		}
		return response, http.StatusOK, nil
	}

	if sme != nil {
		response = fiber.Map{
			"exists":              true,
			"email":               sme.Email,
			"id":                  sme.ID,
			"active_subscription": false,
		}
	}
	subscription, err := subscriptionRepo.GetLatestSubscriptionByBusinessID(pdb, sme.ID)
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to fetch subscription: %w", err)
	}

	if subscription != nil && subscription.IsActive {
		response["active_subscription"] = true
		response["subscription"] = fiber.Map{
			"plan":               subscription.Plan,
			"total_invoices":     subscription.TotalInvoices,
			"remaining_invoices": subscription.RemainingInvoices,
			"used_invoices":      subscription.UsedInvoices,
			"next_billing_date":  subscription.NextBillingDate,
		}
	}

	return response, http.StatusOK, nil
}

func GetAvailablePlans(db *gorm.DB) ([]models.SubscriptionPlan, error) {
	return subscriptionService.ListActivePlans(db)
}

func generateTransactionReference() string {
	return fmt.Sprintf("txn_%d_%s", time.Now().Unix(), strings.ToLower(utility.RandomString(10)))
}

func SubscribeBusinessToPlan(req dtos.PluginSubscribeRequestDto, db *gorm.DB) (fiber.Map, int, error) {
	pdb := inst.InitDB(db, false)

	plan, found, err := subscriptionService.GetActivePlanByID(strings.TrimSpace(req.PlanID), db)
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to fetch plan: %w", err)
	}
	if !found {
		return nil, http.StatusBadRequest, fmt.Errorf("invalid or inactive plan id")
	}

	sme, err := smeRepo.FindSmeByID(pdb, req.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, http.StatusNotFound, fmt.Errorf("business not found")
		}
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to fetch business: %w", err)
	}

	reference := generateTransactionReference()
	transactionRecord := &models.Transaction{
		ID:        utility.GenerateUUID(),
		SmeID:     sme.ID,
		Reference: reference,
		Provider:  "paystack",
		Status:    "pending",
		Amount:    plan.Amount,
		Currency:  "NGN",
		Plan:      plan.Name,
	}

	if err := transactionRepo.CreateTransaction(transactionRecord, pdb); err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to create transaction log: %w", err)
	}

	providerResp, err := paystack.InitializeTransaction(paystack.InitializeTransactionRequest{
		Email:     sme.Email,
		Amount:    strconv.Itoa(int(math.Round(plan.Amount * 100))),
		Reference: reference,
		Metadata: &paystack.InitializeTransactionMetadata{
			IsSandbox: req.IsSandbox,
		},
	})
	if err != nil {
		transactionRecord.Status = "failed"
		transactionRecord.GatewayResponse = err.Error()
		_ = transactionRepo.SaveTransaction(transactionRecord, pdb)
		return nil, http.StatusBadGateway, fmt.Errorf("failed to initialize paystack transaction: %w", err)
	}

	if !providerResp.Status {
		transactionRecord.Status = "failed"
		transactionRecord.GatewayResponse = providerResp.Message
		_ = transactionRepo.SaveTransaction(transactionRecord, pdb)
		return nil, http.StatusBadGateway, fmt.Errorf("paystack initialization failed: %s", providerResp.Message)
	}

	transactionRecord.Status = "initialized"
	transactionRecord.ProviderReference = providerResp.Data.Reference
	transactionRecord.AuthorizationURL = providerResp.Data.AuthorizationURL
	transactionRecord.AccessCode = providerResp.Data.AccessCode
	transactionRecord.GatewayResponse = providerResp.Message
	if err := transactionRepo.SaveTransaction(transactionRecord, pdb); err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to update transaction log: %w", err)
	}

	response := fiber.Map{
		"provider":          "paystack",
		"transaction_id":    transactionRecord.ID,
		"transaction_ref":   transactionRecord.Reference,
		"authorization_url": providerResp.Data.AuthorizationURL,
	}

	return response, http.StatusOK, nil
}

func HandlePaystackWebhook(rawBody []byte, signature string, db *gorm.DB) (fiber.Map, int, error) {
	if err := ValidatePaystackSignature(rawBody, signature); err != nil {
		if errors.Is(err, ErrInvalidPaystackSignature) {
			return nil, http.StatusUnauthorized, err
		}
		return nil, http.StatusInternalServerError, err
	}

	var payload paystack.WebhookPayload
	if err := json.Unmarshal(rawBody, &payload); err != nil {
		return nil, http.StatusBadRequest, fmt.Errorf("invalid webhook payload: %w", err)
	}

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

	transactionRecord.ProviderReference = payload.Data.Reference
	transactionRecord.ProviderPayload = rawBody
	if payload.Data.Currency != "" {
		transactionRecord.Currency = payload.Data.Currency
	}
	if payload.Data.Amount > 0 {
		transactionRecord.Amount = payload.Data.Amount / 100.0
	}
	transactionRecord.GatewayResponse = payload.Data.GatewayResponse

	switch {
	case payload.Event == "charge.success" && strings.EqualFold(payload.Data.Status, "success"):
		transactionRecord.Status = "success"
	case strings.Contains(strings.ToLower(payload.Event), "failed") || strings.EqualFold(payload.Data.Status, "failed"):
		transactionRecord.Status = "failed"
	default:
		transactionRecord.Status = "processing"
	}

	if err := transactionRepo.SaveTransaction(transactionRecord, pdb); err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to update transaction status: %w", err)
	}

	if transactionRecord.Status == "success" {
		plan, found, err := subscriptionService.GetPlanByName(transactionRecord.Plan, db)
		if err != nil {
			return nil, http.StatusInternalServerError, fmt.Errorf("failed to fetch plan: %w", err)
		}
		if !found {
			return nil, http.StatusBadRequest, fmt.Errorf("invalid plan in transaction")
		}

		subscription, err := subscriptionRepo.GetLatestSubscriptionByBusinessID(pdb, transactionRecord.SmeID)
		if err != nil {
			return nil, http.StatusInternalServerError, fmt.Errorf("failed to fetch subscription: %w", err)
		}

		if subscription == nil {
			subscription = &models.Subscription{
				ID:    utility.GenerateUUID(),
				SmeID: transactionRecord.SmeID,
			}
		}

		subscription.IsActive = true
		subscription.Plan = plan.Name
		subscription.TotalInvoices = plan.TotalInvoices
		subscription.UsedInvoices = 0
		subscription.RemainingInvoices = plan.TotalInvoices
		subscription.NextBillingDate = time.Now().AddDate(0, 0, plan.BillingCycle)

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

func RegisterUserWithSubscription(req dtos.SmeRegistrationDto, db *gorm.DB, isSandbox bool) (fiber.Map, int, error) {
	tx := db.Begin()
	if tx.Error != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to initialize transaction: %w", tx.Error)
	}

	pdb := inst.InitDB(tx, false)

	email := strings.ToLower(req.Email)
	name := strings.Title(strings.ToLower(req.Name))

	password, err := utility.HashPassword(req.Password)
	if err != nil {
		tx.Rollback()
		return nil, http.StatusBadRequest, fmt.Errorf("failed to hash password: %w", err)
	}

	sme := models.SME{
		ID:           utility.GenerateUUID(),
		Name:         name,
		Email:        email,
		Password:     password,
		TIN:          req.TIN,
		PhoneNumber:  req.PhoneNumber,
		AggregatorID: req.AggregatorID,
		CompanyName:  req.CompanyName,
	}

	err = smeRepo.CreateSme(&sme, pdb)
	if err != nil {
		tx.Rollback()
		return nil, http.StatusBadRequest, fmt.Errorf("failed to create business: %w", err)
	}

	subscription := models.Subscription{
		ID:                utility.GenerateUUID(),
		SmeID:             sme.ID,
		IsActive:          false,
		Plan:              "free",
		TotalInvoices:     0,
		RemainingInvoices: 0,
		UsedInvoices:      0,
		NextBillingDate:   time.Now().AddDate(0, 1, 0),
	}

	err = subscriptionRepo.CreateSubscription(&subscription, pdb)
	if err != nil {
		tx.Rollback()
		return nil, http.StatusBadRequest, fmt.Errorf("failed to create subscription: %w", err)
	}

	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to commit registration transaction: %w", err)
	}

	responseData := fiber.Map{
		"id":         sme.ID,
		"email":      sme.Email,
		"name":       sme.Name,
		"tin":        sme.TIN,
		"is_sandbox": isSandbox,
	}

	return responseData, http.StatusCreated, nil
}
