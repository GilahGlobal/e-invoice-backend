package webhooks

import (
	"einvoice-access-point/external/zoho"
	businessRepository "einvoice-access-point/internal/repository/business"
	repository "einvoice-access-point/internal/repository/invoice"
	"einvoice-access-point/internal/services/token"
	inst "einvoice-access-point/pkg/dbinit"
	"einvoice-access-point/pkg/models"
	"einvoice-access-point/pkg/utility"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

var (
	ErrOrganizationNotFound = errors.New("organization not found")
	ErrZohoAPIUpdateFailed  = errors.New("failed to update invoice in Zoho")
	ErrInvalidSignature     = fmt.Errorf("invalid webhook signature")
)

// HandleZohoWebhookService handles all the webhook logic
func HandleZohoWebhookService(payload zoho.WebhookPayload, rawBody string, signature string,
	db *gorm.DB, logger *utility.Logger, firsKeys *utility.CryptoKeys, orgID string, isSandbox bool) (*zoho.WebhookResponse, *string, error) {
	platform := "zoho"

	business, config, err := GetBuinessConfigs(db, platform, orgID)
	if err != nil {
		return nil, nil, err
	}

	print(string(config.HMACSecret), string(config.AuthToken), string(config.APIKey), string(config.APISecret))

	if utility.VerifyWebhookSignature([]byte(rawBody), string(config.HMACSecret), signature) {
		logger.Error("Invalid webhook signature", zap.String("organization_id", payload.Invoice.InvoiceID))
		return nil, nil, ErrInvalidSignature
	}

	respData, errDetails, err := processZohoWebhook(payload, db, logger, firsKeys, business, *config, isSandbox)
	if err != nil {
		return nil, errDetails, err
	}

	return respData, nil, nil
}

// ProcessWebhook processes the Zoho webhook payload
func processZohoWebhook(payload zoho.WebhookPayload, db *gorm.DB, logger *utility.Logger, firsKeys *utility.CryptoKeys,
	business *models.Business, accConfig models.AccountingPlatformConfig, isSandbox bool) (*zoho.WebhookResponse, *string, error) {

	logger.Info("Processing invoice",
		zap.String("invoice_id", payload.Invoice.InvoiceID),
		zap.String("invoice_number", payload.Invoice.InvoiceNumber),
		zap.String("customer_name", payload.Invoice.CustomerName),
		zap.Float64("total", payload.Invoice.Total))

	pdb := inst.InitDB(db, false)

	platformMetadata := models.PlatformMetadata{
		"zoho": models.InvoicePlatformData{
			InvoiceID:    payload.Invoice.InvoiceID,
			Status:       "sent",
			Total:        payload.Invoice.Total,
			CurrencyCode: "NGN",
		},
	}
	metadataBytes, err := json.Marshal(platformMetadata)
	if err != nil {
		errDetails := "failed to marshal platform metadata"
		logger.Error("Failed to marshal platform metadata", zap.Error(err))
		return nil, &errDetails, fmt.Errorf("failed to marshal platform metadata: %w", err)
	}

	invoiceData, err := json.Marshal(payload.Invoice)
	if err != nil {
		errDetails := "failed to marshal invoice data"
		logger.Error(errDetails, zap.Error(err))
		return nil, &errDetails, fmt.Errorf("%s: %w", errDetails, err)
	}

	currentStatus, statusHistory, err := models.InitPlatformInvoiceStatus()
	if err != nil {
		errDetails := "failed to initialize invoice status"
		logger.Error(errDetails, zap.Error(err))
		return nil, &errDetails, fmt.Errorf("%s: %w", errDetails, err)
	}

	invoice := &models.Invoice{
		InvoiceNumber:    payload.Invoice.InvoiceNumber,
		BusinessID:       business.ID,
		Platform:         "zoho",
		PlatformMetadata: string(metadataBytes),
		InvoiceData:      invoiceData,
		CurrentStatus:    currentStatus,
		StatusHistory:    statusHistory,
		Timestamp:        time.Now(),
	}

	if err := repository.CreateInvoice(pdb, invoice); err != nil {
		errDetails := "failed to save invoice"
		logger.Error("Failed to save invoice", zap.Error(err))
		return nil, &errDetails, fmt.Errorf("failed to save invoice: %w", err)
	}

	theIRN, theQrCode, err := FirsZohoAllInOneProcess(payload, firsKeys, business, invoice, db, isSandbox)
	if err != nil {
		errDetails := "failed to running one or more firs process"
		logger.Error("Failed to running firs processes", zap.Error(err))
		return nil, &errDetails, err
	}

	accessToken, err := token.GetValidAccessToken(db, accConfig, "zoho", accConfig.OrgID)
	if err != nil {
		errDetails := "failed to marshal platform metadata"
		return nil, &errDetails, err
	}

	err = zoho.UpdateZohoInvoice(accessToken, payload.Invoice.InvoiceID, *theIRN, *theQrCode, accConfig)
	if err != nil {
		errDetails := err.Error()
		logger.Error("Failed to update invoice", zap.Error(err), zap.String("invoice_id", payload.Invoice.InvoiceID))
		return nil, &errDetails, fmt.Errorf("%w: %v", ErrZohoAPIUpdateFailed, err)
	}

	resp := &zoho.WebhookResponse{
		InvoiceID:      payload.Invoice.InvoiceID,
		InvoiceNumber:  payload.Invoice.InvoiceNumber,
		CustomerName:   payload.Invoice.CustomerName,
		Total:          payload.Invoice.Total,
		OrganizationID: accConfig.OrgID,
		Updated:        true,
	}

	return resp, nil, nil
}

func GetBuinessConfigs(db *gorm.DB, platform, orgID string) (*models.Business, *models.AccountingPlatformConfig, error) {

	pdb := inst.InitDB(db, false)

	business, err := businessRepository.FindBusinessByPlatformOrgID(pdb, platform, orgID)
	if err != nil {
		fmt.Printf("Business not found, %v, %v, %v", zap.Error(err), zap.String("platform", platform), zap.String("org_id", orgID))
		return nil, nil, fmt.Errorf("business not found: %w", err)
	}

	config, exists := business.PlatformConfigs[platform]
	if !exists {
		fmt.Printf("Platform configuration not found, %v, %v", zap.String("platform", platform), zap.String("org_id", orgID))
		return nil, nil, fmt.Errorf("platform config not found")
	}

	config.HMACSecret.AfterFind(pdb.DB())
	config.AuthToken.AfterFind(pdb.DB())
	config.APIKey.AfterFind(pdb.DB())
	config.APISecret.AfterFind(pdb.DB())

	return business, &config, nil
}
