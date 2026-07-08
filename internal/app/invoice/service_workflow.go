package invoice

import (
	"einvoice-access-point/internal/app/invoice/zohoinvoice"
	"einvoice-access-point/internal/data/dbinit"
	"einvoice-access-point/internal/data/entities"
	"einvoice-access-point/internal/pkg/firs_models"
	"einvoice-access-point/internal/pkg/zoho"
	"einvoice-access-point/internal/utility"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"go.uber.org/zap"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

var (
	ErrOrganizationNotFound = errors.New("organization not found")
	ErrZohoAPIUpdateFailed  = errors.New("failed to update invoice in Zoho")
	ErrInvalidSignature     = errors.New("invalid webhook signature")
)

func (s *Service) FirsAllInOneProcess(payload firs_models.UploadInvoiceRequestDto, invoiceModel *entities.Invoice, db *gorm.DB, isSandbox bool) (error, bool) {
	pdb := dbinit.InitDB(db, false)

	_, theErr, err := s.ValidateInvoice(payload, isSandbox)
	if err != nil {
		_ = s.repo.UpdateInvoiceStatus(pdb, invoiceModel, entities.StatusValidatedInvoice, "failed")
		return fmt.Errorf("failed to validate invoice: %v - %v", *theErr, err), false
	}

	_ = s.repo.UpdateInvoiceStatus(pdb, invoiceModel, entities.StatusValidatedInvoice, "success")

	_, theErr, err = s.SignInvoice(payload, isSandbox)
	if err != nil {
		_ = s.repo.UpdateInvoiceStatus(pdb, invoiceModel, entities.StatusSignedInvoice, "failed")
		return fmt.Errorf("failed to sign invoice: %v - %v", *theErr, err), false
	}
	_ = s.repo.UpdateInvoiceStatus(pdb, invoiceModel, entities.StatusSignedInvoice, "success")

	_, theErr, err = s.TransmitInvoice(*payload.IRN, isSandbox)
	if err != nil {
		_ = s.repo.UpdateInvoiceStatus(pdb, invoiceModel, entities.StatusTransmitted, "failed")
		return fmt.Errorf("failed to transmit invoice: %v - %v", *theErr, err), true
	}
	_ = s.repo.UpdateInvoiceStatus(pdb, invoiceModel, entities.StatusTransmitted, "success")

	confirmInvoiceResp, theErr, err := s.ConfirmInvoice(*payload.IRN, isSandbox)
	if err != nil {
		_ = s.repo.UpdateInvoiceStatus(pdb, invoiceModel, entities.StatusConfirmed, "failed")
		return fmt.Errorf("failed to confirm invoice: %v - %v", *theErr, err), true
	}

	if confirmInvoiceResp.Code != 200 {
		_ = s.repo.UpdateInvoiceStatus(pdb, invoiceModel, entities.StatusConfirmed, "failed")
		return fmt.Errorf("failed to confirm invoice, didnt get 200 or delivered is false"), true
	}

	_ = s.repo.UpdateInvoiceStatus(pdb, invoiceModel, entities.StatusConfirmed, "success")
	return nil, true
}

func (s *Service) UncompletedFirsProcesses(db *gorm.DB, currentStatus string, payload firs_models.UploadInvoiceRequestDto, invoiceModel *entities.Invoice, isSandbox bool) (error, bool) {
	pdb := dbinit.InitDB(db, false)

	switch currentStatus {
	case entities.StatusValidatedInvoice:
		_, theErr, err := s.ValidateInvoice(payload, isSandbox)
		if err != nil {
			_ = s.repo.UpdateInvoiceStatus(pdb, invoiceModel, entities.StatusValidatedInvoice, "failed")
			return fmt.Errorf("failed to validate invoice: %v - %v", *theErr, err), false
		}

		_ = s.repo.UpdateInvoiceStatus(pdb, invoiceModel, entities.StatusValidatedInvoice, "success")

		_, theErr, err = s.SignInvoice(payload, isSandbox)
		if err != nil {
			_ = s.repo.UpdateInvoiceStatus(pdb, invoiceModel, entities.StatusSignedInvoice, "failed")
			return fmt.Errorf("failed to sign invoice: %v - %v", *theErr, err), false
		}
		_ = s.repo.UpdateInvoiceStatus(pdb, invoiceModel, entities.StatusSignedInvoice, "success")

		_, theErr, err = s.TransmitInvoice(*payload.IRN, isSandbox)
		if err != nil {
			_ = s.repo.UpdateInvoiceStatus(pdb, invoiceModel, entities.StatusTransmitted, "failed")
			return fmt.Errorf("failed to transmit invoice: %v - %v", *theErr, err), true
		}
		_ = s.repo.UpdateInvoiceStatus(pdb, invoiceModel, entities.StatusTransmitted, "success")

		confirmInvoiceResp, theErr, err := s.ConfirmInvoice(*payload.IRN, isSandbox)
		if err != nil {
			_ = s.repo.UpdateInvoiceStatus(pdb, invoiceModel, entities.StatusConfirmed, "failed")
			return fmt.Errorf("failed to confirm invoice: %v - %v", *theErr, err), true
		}

		if confirmInvoiceResp.Code != 200 {
			return fmt.Errorf("failed to confirm invoice, didnt get 200 or delivered is false"), true
		}

		_ = s.repo.UpdateInvoiceStatus(pdb, invoiceModel, entities.StatusConfirmed, "success")
		return nil, true
	case entities.StatusSignedIRN:
		return s.FirsAllInOneProcess(payload, invoiceModel, db, isSandbox)
	default:
		return fmt.Errorf("unknown status: %s", currentStatus), false
	}
}

func (s *Service) FirsZohoAllInOneProcess(payload zoho.WebhookPayload, firsKeys *utility.CryptoKeys, business *entities.Business,
	invoiceModel *entities.Invoice, db *gorm.DB, isSandBox bool) (*string, *string, error) {
	pdb := dbinit.InitDB(db, false)

	theIRN, err := s.GenerateIRN(payload.Invoice.InvoiceNumber, *business.ServiceID)
	if err != nil {
		_ = s.repo.UpdateInvoiceStatus(pdb, invoiceModel, entities.StatusGeneratedIRN, "failed")
		return nil, nil, err
	}

	_ = s.repo.UpdateInvoiceStatus(pdb, invoiceModel, entities.StatusGeneratedIRN, "success")

	validateIRN := firs_models.IRNValidationRequest{
		InvoiceReference: payload.Invoice.InvoiceID,
		BusinessID:       *business.BusinessID,
		IRN:              *theIRN,
	}

	_, theErr, err := s.ValidateIRN(validateIRN, isSandBox)
	if err != nil {
		_ = s.repo.UpdateInvoiceStatus(pdb, invoiceModel, entities.StatusValidatedIRN, "failed")
		return nil, nil, fmt.Errorf("failed to validate irn: %v - %v", *theErr, err)
	}
	_ = s.repo.UpdateInvoiceStatus(pdb, invoiceModel, entities.StatusValidatedIRN, "success")

	signIRNResp, err := s.SignIRN(*theIRN, firsKeys)
	if err != nil {
		_ = s.repo.UpdateInvoiceStatus(pdb, invoiceModel, entities.StatusSignedIRN, "failed")
		return nil, nil, err
	}
	_ = s.repo.UpdateInvoiceStatus(pdb, invoiceModel, entities.StatusSignedIRN, "success")
	_ = s.repo.UpdateInvoiceIRN(pdb, invoiceModel, *theIRN)

	go func(p zoho.WebhookPayload, b *entities.Business, inv *entities.Invoice, d *gorm.DB, irn string) {
		if err := s.otherFirsProcesses(p, b, inv, d, irn, isSandBox); err != nil {
			fmt.Println("Error in otherFirsProcesses: ", err)
		}
	}(payload, business, invoiceModel, db, *theIRN)

	return theIRN, &signIRNResp.EncryptedIRN, nil
}

func (s *Service) otherFirsProcesses(payload zoho.WebhookPayload, business *entities.Business, invoiceModel *entities.Invoice, db *gorm.DB, theIRN string, isSandBox bool) error {
	pdb := dbinit.InitDB(db, false)

	newInvoiceResp, err := zohoinvoice.ConvertZohoToFIRS(payload.Invoice, *business.BusinessID, business.Name, theIRN)
	if err != nil {
		return err
	}

	_, theErr, err := s.ValidateInvoice(newInvoiceResp, isSandBox)
	if err != nil {
		_ = s.repo.UpdateInvoiceStatus(pdb, invoiceModel, entities.StatusValidatedInvoice, "failed")
		return fmt.Errorf("failed to validate invoice: %v - %v", *theErr, err)
	}
	_ = s.repo.UpdateInvoiceStatus(pdb, invoiceModel, entities.StatusValidatedInvoice, "success")

	_, theErr, err = s.SignInvoice(newInvoiceResp, isSandBox)
	if err != nil {
		_ = s.repo.UpdateInvoiceStatus(pdb, invoiceModel, entities.StatusSignedInvoice, "failed")
		return fmt.Errorf("failed to sign invoice: %v - %v", *theErr, err)
	}
	_ = s.repo.UpdateInvoiceStatus(pdb, invoiceModel, entities.StatusSignedInvoice, "success")

	_, theErr, err = s.TransmitInvoice(*newInvoiceResp.IRN, isSandBox)
	if err != nil {
		_ = s.repo.UpdateInvoiceStatus(pdb, invoiceModel, entities.StatusTransmitted, "failed")
		return fmt.Errorf("failed to transmit invoice: %v - %v", *theErr, err)
	}
	_ = s.repo.UpdateInvoiceStatus(pdb, invoiceModel, entities.StatusTransmitted, "success")

	confirmInvoiceResp, theErr, err := s.ConfirmInvoice(theIRN, isSandBox)
	if err != nil {
		return fmt.Errorf("failed to confirm invoice: %v - %v", *theErr, err)
	}
	_ = s.repo.UpdateInvoiceStatus(pdb, invoiceModel, entities.StatusConfirmed, "success")

	if confirmInvoiceResp.Code != 200 {
		return fmt.Errorf("failed to confirm invoice, didnt get 200 or delivered is false")
	}

	return nil
}

func (s *Service) ProcessFirsWebhook(payload firs_models.FirsWebhookPayload) error {
	_, errDetails, err := s.TransmitConfirmInvoice(payload.IRN, false)
	if err != nil {
		if errDetails != nil {
			return fmt.Errorf("failed to confirm transmitted invoice: %v - %v", *errDetails, err)
		}
		return fmt.Errorf("failed to confirm transmitted invoice: %v", err)
	}

	fmt.Printf("irn: %s and message: %s", payload.IRN, payload.Message)
	return nil
}

func (s *Service) HandleZohoWebhookService(payload zoho.WebhookPayload, rawBody string, signature string,
	db *gorm.DB, logger *utility.Logger, firsKeys *utility.CryptoKeys, orgID string, isSandbox bool) (*zoho.WebhookResponse, *string, error) {
	platform := "zoho"

	business, config, err := s.GetBusinessConfigs(db, platform, orgID)
	if err != nil {
		return nil, nil, err
	}

	if !utility.VerifyWebhookSignature([]byte(rawBody), string(config.HMACSecret), signature) {
		logger.Error("Invalid webhook signature", zap.String("organization_id", orgID))
		return nil, nil, ErrInvalidSignature
	}

	respData, errDetails, err := s.processZohoWebhook(payload, db, logger, firsKeys, business, *config, isSandbox)
	if err != nil {
		return nil, errDetails, err
	}

	return respData, nil, nil
}

func (s *Service) processZohoWebhook(payload zoho.WebhookPayload, db *gorm.DB, logger *utility.Logger, firsKeys *utility.CryptoKeys,
	business *entities.Business, accConfig entities.AccountingPlatformConfig, isSandbox bool) (*zoho.WebhookResponse, *string, error) {
	logger.Info("Processing invoice",
		zap.String("invoice_id", payload.Invoice.InvoiceID),
		zap.String("invoice_number", payload.Invoice.InvoiceNumber),
		zap.String("customer_name", payload.Invoice.CustomerName),
		zap.Float64("total", payload.Invoice.Total))

	pdb := dbinit.InitDB(db, false)

	platformMetadata := entities.PlatformMetadata{
		"zoho": entities.InvoicePlatformData{
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

	currentStatus, statusHistory, err := entities.InitPlatformInvoiceStatus()
	if err != nil {
		errDetails := "failed to initialize invoice status"
		logger.Error(errDetails, zap.Error(err))
		return nil, &errDetails, fmt.Errorf("%s: %w", errDetails, err)
	}

	invoice := &entities.Invoice{
		InvoiceNumber:    payload.Invoice.InvoiceNumber,
		BusinessID:       business.ID,
		Platform:         "zoho",
		PlatformMetadata: datatypes.JSON(metadataBytes),
		InvoiceData:      invoiceData,
		CurrentStatus:    currentStatus,
		StatusHistory:    statusHistory,
		Timestamp:        time.Now(),
	}

	if err := s.repo.CreateInvoice(pdb, invoice); err != nil {
		errDetails := "failed to save invoice"
		logger.Error("Failed to save invoice", zap.Error(err))
		return nil, &errDetails, fmt.Errorf("failed to save invoice: %w", err)
	}

	theIRN, theQrCode, err := s.FirsZohoAllInOneProcess(payload, firsKeys, business, invoice, db, isSandbox)
	if err != nil {
		errDetails := "failed to running one or more firs process"
		logger.Error("Failed to running firs processes", zap.Error(err))
		return nil, &errDetails, err
	}

	accessToken, err := s.tokenSvc.GetValidAccessToken(db, accConfig, "zoho", accConfig.OrgID)
	if err != nil {
		errDetails := "failed to get access token"
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

func (s *Service) GetBusinessConfigs(db *gorm.DB, platform, orgID string) (*entities.Business, *entities.AccountingPlatformConfig, error) {
	business, err := s.businessSvc.GetBusinessByPlatformOrgID(db, platform, orgID)
	pdb := dbinit.InitDB(db, false)
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
