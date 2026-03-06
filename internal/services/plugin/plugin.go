package plugin

import (
	"crypto/hmac"
	"crypto/sha256"
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
	userRepo "einvoice-access-point/internal/repository/business"
	subscriptionRepo "einvoice-access-point/internal/repository/subscription"
	transactionRepo "einvoice-access-point/internal/repository/transaction"
	subscriptionService "einvoice-access-point/internal/services/subscription"
	"einvoice-access-point/pkg/common"
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

func CheckBusinessWithSubscription(email string, db *gorm.DB) (fiber.Map, int, error) {
	pdb := inst.InitDB(db, false)

	email = strings.ToLower(strings.TrimSpace(email))
	formattedEmail, ok := utility.EmailValid(email)
	if !ok {
		return nil, http.StatusBadRequest, fmt.Errorf("email address is invalid")
	}

	business, err := userRepo.GetUserByEmail(pdb, formattedEmail)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fiber.Map{
				"exists": false,
				"email":  formattedEmail,
			}, http.StatusOK, nil
		}
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to fetch business: %w", err)
	}

	response := fiber.Map{
		"exists":              true,
		"email":               business.Email,
		"business_id":         business.ID,
		"active_subscription": false,
	}

	subscription, err := subscriptionRepo.GetLatestSubscriptionByBusinessID(pdb, business.ID)
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

	email := strings.ToLower(strings.TrimSpace(req.Email))
	formattedEmail, ok := utility.EmailValid(email)
	if !ok {
		return nil, http.StatusBadRequest, fmt.Errorf("email address is invalid")
	}

	plan, found, err := subscriptionService.GetActivePlanByID(strings.TrimSpace(req.PlanID), db)
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to fetch plan: %w", err)
	}
	if !found {
		return nil, http.StatusBadRequest, fmt.Errorf("invalid or inactive plan id")
	}

	business, err := userRepo.GetUserByEmail(pdb, formattedEmail)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, http.StatusNotFound, fmt.Errorf("business not found")
		}
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to fetch business: %w", err)
	}

	reference := generateTransactionReference()
	transactionRecord := &models.Transaction{
		ID:         utility.GenerateUUID(),
		BusinessID: business.ID,
		Reference:  reference,
		Provider:   "paystack",
		Status:     "pending",
		Amount:     plan.Amount,
		Currency:   "NGN",
		Plan:       plan.Name,
	}

	if err := transactionRepo.CreateTransaction(transactionRecord, pdb); err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to create transaction log: %w", err)
	}

	providerResp, err := paystack.InitializeTransaction(paystack.InitializeTransactionRequest{
		Email:     formattedEmail,
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
	transactionRecord.ProviderPayload = string(rawBody)
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

		subscription, err := subscriptionRepo.GetLatestSubscriptionByBusinessID(pdb, transactionRecord.BusinessID)
		if err != nil {
			return nil, http.StatusInternalServerError, fmt.Errorf("failed to fetch subscription: %w", err)
		}

		if subscription == nil {
			subscription = &models.Subscription{
				ID:         utility.GenerateUUID(),
				BusinessID: transactionRecord.BusinessID,
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

func RegisterUserWithSubscription(req dtos.RegisterDto, db *gorm.DB, isSandbox bool) (fiber.Map, int, error) {
	tx := db.Begin()
	if tx.Error != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to initialize transaction: %w", tx.Error)
	}

	pdb := inst.InitDB(tx, false)

	cfg := config.GetConfig()
	serverSecret := cfg.Server.Secret
	email := strings.ToLower(req.Email)
	name := strings.Title(strings.ToLower(req.Name))

	password, err := utility.HashPassword(req.Password)
	if err != nil {
		tx.Rollback()
		return nil, http.StatusBadRequest, fmt.Errorf("failed to hash password: %w", err)
	}

	apiKey, err := utility.GenerateSecureToken(32, serverSecret)
	if err != nil {
		tx.Rollback()
		return nil, http.StatusBadRequest, fmt.Errorf("failed to generate api key: %w", err)
	}

	encryptedAPIKey, err := common.EncryptAES(apiKey)
	if err != nil {
		tx.Rollback()
		return nil, http.StatusBadRequest, fmt.Errorf("failed to encrypt API key: %w", err)
	}

	apiKeyHash := sha256.Sum256([]byte(apiKey))
	apiKeyHashStr := hex.EncodeToString(apiKeyHash[:])

	platformConfigs := models.PlatformConfigs{}
	for platform, cfg := range req.PlatformConfigs {
		encryptedHMACSecret, err := common.EncryptAES(string(cfg.HMACSecret))
		if err != nil {
			tx.Rollback()
			return nil, http.StatusBadRequest, fmt.Errorf("failed to encrypt HMAC secret for %s: %w", platform, err)
		}

		encryptedPlatformAPIKey, err := common.EncryptAES(string(cfg.APIKey))
		if err != nil {
			tx.Rollback()
			return nil, http.StatusBadRequest, fmt.Errorf("failed to encrypt API key for %s: %w", platform, err)
		}

		encryptedAPISecret, err := common.EncryptAES(string(cfg.APISecret))
		if err != nil {
			tx.Rollback()
			return nil, http.StatusBadRequest, fmt.Errorf("failed to encrypt API secret for %s: %w", platform, err)
		}

		encryptedAuthToken, err := common.EncryptAES(string(cfg.AuthToken))
		if err != nil {
			tx.Rollback()
			return nil, http.StatusBadRequest, fmt.Errorf("failed to encrypt Auth token for %s: %w", platform, err)
		}

		platformConfigs[platform] = models.AccountingPlatformConfig{
			OrgID:      cfg.OrgID,
			HMACSecret: common.EncryptedString(encryptedHMACSecret),
			AuthToken:  common.EncryptedString(encryptedAuthToken),
			APIKey:     common.EncryptedString(encryptedPlatformAPIKey),
			APISecret:  common.EncryptedString(encryptedAPISecret),
		}
	}

	user := models.Business{
		ID:              utility.GenerateUUID(),
		Name:            name,
		Email:           email,
		Password:        password,
		IsPluginUser:    true,
		ServiceID:       "6A2BC898", // userRepo.GenerateUniqueServiceID(pdb.Db)
		APIKey:          common.EncryptedString(encryptedAPIKey),
		APIKeyHash:      apiKeyHashStr,
		PlatformConfigs: platformConfigs,
		AccStatus:       0,
		TIN:             req.TIN,
		PhoneNumber:     req.PhoneNumber,
		CompanyName:     req.CompanyName,
	}

	err = userRepo.CreateBusiness(&user, pdb)
	if err != nil {
		tx.Rollback()
		return nil, http.StatusBadRequest, fmt.Errorf("failed to create business: %w", err)
	}

	subscription := models.Subscription{
		ID:                utility.GenerateUUID(),
		BusinessID:        user.ID,
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
		"id":          user.ID,
		"email":       user.Email,
		"name":        user.Name,
		"business_id": user.BusinessID,
		"service_id":  user.ServiceID,
		"tin":         user.TIN,
		"is_sandbox":  isSandbox,
	}

	return responseData, http.StatusCreated, nil
}
