package invoice

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"einvoice-access-point/internal/app/business"
	"einvoice-access-point/internal/app/invoice/zohoinvoice"
	"einvoice-access-point/internal/app/token"
	"einvoice-access-point/internal/common"
	"einvoice-access-point/internal/config"
	"einvoice-access-point/internal/data/database"
	"einvoice-access-point/internal/data/dbinit"
	"einvoice-access-point/internal/data/entities"
	"einvoice-access-point/internal/data/repositories"
	"einvoice-access-point/internal/pkg/firs"
	"einvoice-access-point/internal/pkg/firs_models"
	"einvoice-access-point/internal/pkg/zoho"
	"einvoice-access-point/internal/utility"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"image/png"
	"log"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/skip2/go-qrcode"
	"go.uber.org/zap"
	"golang.org/x/image/bmp"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type Service struct {
	repo        *repositories.InvoiceRepository
	tokenSvc    *token.Service
	businessSvc *business.Service
}

func NewService(repo *repositories.InvoiceRepository, tokenSvc *token.Service, businessSvc *business.Service) *Service {
	return &Service{repo: repo, tokenSvc: tokenSvc, businessSvc: businessSvc}
}

func NewServiceWithDB(prodDb, testDb database.DatabaseManager, tokenSvc *token.Service, businessSvc *business.Service) *Service {
	repo := repositories.NewInvoiceRepository(prodDb, testDb)
	return NewService(repo, tokenSvc, businessSvc)
}

func (s *Service) BusinessSvc() *business.Service {
	return s.businessSvc
}

func (s *Service) GetAllInvoicesByBusinessID(db *gorm.DB, businessID string, page, size int) ([]InvoiceListItem, database.PaginationResponse, error) {
	pdb := dbinit.InitDB(db, false)

	pagination := database.Pagination{
		Page:  page,
		Limit: size,
	}

	rows, paginationResponse, err := s.repo.FindInvoicesWithMetadataByBusinessID(pdb, businessID, pagination)
	if err != nil {
		return nil, paginationResponse, err
	}

	result := make([]InvoiceListItem, 0, len(rows))
	for _, row := range rows {
		metadata := make([]InvoiceStepMetadata, 0)
		if len(row.StatusHistory) > 0 {
			if err := json.Unmarshal(row.StatusHistory, &metadata); err != nil {
				return nil, paginationResponse, fmt.Errorf("failed to parse invoice metadata for %s: %w", row.InvoiceNumber, err)
			}
		}

		result = append(result, InvoiceListItem{
			ID:            row.ID,
			InvoiceNumber: row.InvoiceNumber,
			IRN:           row.IRN,
			Platform:      row.Platform,
			CurrentStatus: row.CurrentStatus,
			PaymentStatus: row.PaymentStatus,
			StatusText:    row.StatusText,
			Metadata:      metadata,
			QrCodeBmpUrl:  row.QrCodeBmpUrl,
			QrCode:        row.QrCode,
			CreatedAt:     row.CreatedAt,
		})
	}

	return result, paginationResponse, nil
}

func (s *Service) GetInvoiceDetails(db *gorm.DB, businessID, invoiceID string) (*entities.Invoice, error) {
	pdb := dbinit.InitDB(db, false)
	return s.repo.FindInvoiceByBusinessAndID(pdb, businessID, invoiceID)
}

func (s *Service) CreateInvoice(db *gorm.DB, payload firs_models.UploadInvoiceRequestDto, invoiceNumber, businessID, qrCode, qrCodeBMPURL, encryptedIRN string, invoiceExists *entities.Invoice, isSandbox bool, aggregatorID *string, client string) (*entities.Invoice, *string, error, bool) {
	pdb := dbinit.InitDB(db, false)
	isInvoiceSigned := false
	var invoice *entities.Invoice
	platform := "internal"
	if client == "" {
		platform = "API"
	}

	invoiceData, err := json.Marshal(payload)
	if err != nil {
		errDetails := "failed to marshal invoice data"
		return nil, &errDetails, fmt.Errorf("%s: %w", errDetails, err), isInvoiceSigned
	}

	currentStatus, statusHistory, err := entities.InitNewInvoiceStatus()
	if err != nil {
		errDetails := "failed to initialize invoice status"
		return nil, &errDetails, fmt.Errorf("%s: %w", errDetails, err), isInvoiceSigned
	}

	platformMetadata := "{}"

	if invoiceExists != nil {
		err = s.UpdateInvoiceData(pdb, invoiceExists.InvoiceNumber, invoiceData)
		if err != nil {
			return nil, nil, errors.New("failed to update invoice"), isInvoiceSigned
		}

		invoice, _ = s.repo.FindInvoiceByNumber(pdb, invoiceExists.InvoiceNumber)
		if err, isInvoiceSigned = s.UncompletedFirsProcesses(db, invoiceExists.CurrentStatus, payload, invoiceExists, isSandbox); err != nil {
			return invoice, nil, errors.New(utility.ExtractRelevantErrorMessage(err)), isInvoiceSigned
		}
	} else {
		paymentStatus := "PENDING"
		if payload.PaymentStatus != nil && *payload.PaymentStatus != "" {
			paymentStatus = strings.ToUpper(*payload.PaymentStatus)
		}

		invoice = &entities.Invoice{
			InvoiceNumber:    invoiceNumber,
			IRN:              *payload.IRN,
			QrCode:           qrCode,
			QrCodeBmpUrl:     qrCodeBMPURL,
			BusinessID:       businessID,
			Platform:         platform,
			PlatformMetadata: datatypes.JSON(platformMetadata),
			InvoiceData:      invoiceData,
			CurrentStatus:    currentStatus,
			PaymentStatus:    paymentStatus,
			StatusHistory:    datatypes.JSON(statusHistory),
			Timestamp:        time.Now(),
			EncryptedIRN:     encryptedIRN,
			AggregatorID:     aggregatorID,
		}

		if err := s.repo.CreateInvoice(pdb, invoice); err != nil {
			errDetails := "failed to save invoice"
			return nil, &errDetails, fmt.Errorf("%s: %w", errDetails, err), isInvoiceSigned
		}
		if err, isInvoiceSigned = s.FirsAllInOneProcess(payload, invoice, db, isSandbox); err != nil {
			return invoice, nil, errors.New(utility.ExtractRelevantErrorMessage(err)), isInvoiceSigned
		}
	}

	return invoice, nil, nil, isInvoiceSigned
}

func (s *Service) DeleteInvoice(db *gorm.DB, businessID, invoiceID string) error {
	pdb := dbinit.InitDB(db, false)
	return s.repo.DeleteInvoiceByBusinessAndID(pdb, businessID, invoiceID)
}

func (s *Service) GetInvoiceByInvoiceNumber(db *gorm.DB, invoiceNumber, businessID string) (*entities.Invoice, error) {
	pdb := dbinit.InitDB(db, false)
	return s.repo.FindInvoiceByNumberAndBusinessID(pdb, invoiceNumber, businessID)
}

func (s *Service) GetInvoiceByIRN(db *gorm.DB, irn, businessID string) (*entities.Invoice, error) {
	pdb := dbinit.InitDB(db, false)
	return s.repo.FindInvoiceByIRNAndBusinessID(pdb, irn, businessID)
}

func (s *Service) UpdateInvoiceData(db database.DatabaseManager, invoiceNumber string, invoiceData []byte) error {
	return s.repo.UpdateInvoice(db, invoiceNumber, invoiceData)
}

func (s *Service) IRNGeneration(db *gorm.DB, ownerID, invoiceNumber, serviceId, businessID string, isSandbox bool) (*InvoiceData, *entities.Response) {
	generatedIRN, err := s.GenerateIRN(invoiceNumber, serviceId)
	if err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusBadRequest, "error", err.Error(), err, nil)
		return nil, &rd
	}

	keys, err := s.businessSvc.ResolveBusinessIRNSigningKeys(db, ownerID, isSandbox, nil)
	if err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusBadRequest, "error", err.Error(), err, nil)
		return nil, &rd
	}

	signedIRNResponse, err := s.SignIRN(*generatedIRN, keys)
	if err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusBadRequest, "error", err.Error(), err, nil)
		return nil, &rd
	}

	return &InvoiceData{
		InvoiceNumber: invoiceNumber,
		IRN:           *generatedIRN,
		QRCode:        signedIRNResponse.QrCodeImage,
		QRCode2:       signedIRNResponse.EncryptedIRN,
		QRCodeBMP:     signedIRNResponse.QrCodeImageBMP,
	}, nil
}

func (s *Service) DeprecateInvoiceOnNRS(oldIRN string, isSandbox bool) error {
	theResp, _, err := s.UpdateInvoice(firs_models.UpdateInvoice{
		PaymentStatus: "REJECTED",
	}, oldIRN, isSandbox)

	fmt.Println("Invoice depreciated successful: ", theResp)
	return err
}

func (s *Service) ReplaceInvoiceRecord(db *gorm.DB, existing *entities.Invoice, payload firs_models.UploadInvoiceRequestDto, newIRN, qrCode, qrCodeBMPURL, encryptedIRN, client string) (*entities.Invoice, error) {
	pdb := dbinit.InitDB(db, false)

	invoiceData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal invoice data: %w", err)
	}

	currentStatus, statusHistory, err := entities.InitNewInvoiceStatus()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize invoice status: %w", err)
	}

	platform := "internal"
	if client == "" {
		platform = "API"
	}

	newInvoice := &entities.Invoice{
		ID:               existing.ID,
		InvoiceNumber:    existing.InvoiceNumber,
		BusinessID:       existing.BusinessID,
		AggregatorID:     existing.AggregatorID,
		CreatedAt:        existing.CreatedAt,
		IRN:              newIRN,
		QrCode:           qrCode,
		QrCodeBmpUrl:     qrCodeBMPURL,
		EncryptedIRN:     encryptedIRN,
		InvoiceData:      invoiceData,
		CurrentStatus:    currentStatus,
		StatusHistory:    datatypes.JSON(statusHistory),
		Platform:         platform,
		PlatformMetadata: datatypes.JSON("{}"),
		Timestamp:        time.Now(),
	}

	if err := s.repo.SaveInvoice(pdb, newInvoice); err != nil {
		return nil, fmt.Errorf("failed to save replaced invoice: %w", err)
	}

	return newInvoice, nil
}

func (s *Service) GetInvoiceStats(db *gorm.DB, businessID *string, aggregatorID *string) (*entities.InvoiceStatsResponseData, error) {
	return s.repo.GetInvoiceStats(db, businessID, aggregatorID)
}

func (s *Service) UpdateStoredInvoicePaymentStatus(db *gorm.DB, businessID, irn, paymentStatus string) error {
	pdb := dbinit.InitDB(db, false)

	invoiceRecord, err := s.repo.FindInvoiceByIRNAndBusinessID(pdb, irn, businessID)
	if err != nil {
		return fmt.Errorf("failed to find local invoice record: %w", err)
	}
	if invoiceRecord == nil {
		return fmt.Errorf("local invoice record not found for irn %s", irn)
	}

	var invoiceData map[string]interface{}
	if err := json.Unmarshal(invoiceRecord.InvoiceData, &invoiceData); err != nil {
		return fmt.Errorf("failed to unmarshal invoice data: %w", err)
	}
	if invoiceData == nil {
		invoiceData = make(map[string]interface{})
	}

	invoiceData["payment_status"] = paymentStatus

	updatedInvoiceData, err := json.Marshal(invoiceData)
	if err != nil {
		return fmt.Errorf("failed to marshal invoice data: %w", err)
	}

	if err := s.repo.UpdateInvoiceDataAndPaymentStatusByID(pdb, invoiceRecord.ID, updatedInvoiceData, paymentStatus); err != nil {
		return fmt.Errorf("failed to update local invoice record: %w", err)
	}

	return nil
}

func (s *Service) BulkUpdateInvoice(db *gorm.DB, userID string, req firs_models.BulkUpdateInvoiceRequest, isSandbox bool) (*firs_models.BulkUpdateInvoiceResponse, error) {
	response := &firs_models.BulkUpdateInvoiceResponse{
		Successful: make([]string, 0),
		Failed:     make([]firs_models.BulkUpdateFailedItem, 0),
	}

	for _, item := range req.Invoices {
		updateReq := firs_models.UpdateInvoice{
			PaymentStatus: item.PaymentStatus,
			Reference:     item.Reference,
		}

		_, errDetails, err := s.UpdateInvoice(updateReq, item.IRN, isSandbox)
		if err != nil {
			errMsg := err.Error()
			if errDetails != nil {
				errMsg = fmt.Sprintf("%s: %s", errMsg, *errDetails)
			}
			response.Failed = append(response.Failed, firs_models.BulkUpdateFailedItem{
				IRN:   item.IRN,
				Error: errMsg,
			})
			continue
		}

		if err := s.UpdateStoredInvoicePaymentStatus(db, userID, item.IRN, item.PaymentStatus); err != nil {
			response.Failed = append(response.Failed, firs_models.BulkUpdateFailedItem{
				IRN:   item.IRN,
				Error: fmt.Sprintf("invoice updated on FIRS but failed to update local record: %v", err),
			})
			continue
		}

		response.Successful = append(response.Successful, item.IRN)
	}

	return response, nil
}

func (s *Service) GenerateIRNumber(invoiceNumber, serviceID string, timestamp time.Time) (string, error) {
	return common.GenerateIRN(invoiceNumber, serviceID, timestamp)
}

func (s *Service) ValidateIRN(invoiceReq firs_models.IRNValidationRequest, isSandbox bool) (*firs_models.FirsResponse, *string, error) {
	payload, _ := json.Marshal(invoiceReq)
	log.Printf("Validating IRN with payload: %s", string(payload))

	resp, err := firs.ValidateIRN(invoiceReq, isSandbox)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to validate irn: %w", err)
	}

	theResp, errDetails, err := firs.ParseFIRSAPIResponse(resp)
	if err != nil {
		return nil, errDetails, fmt.Errorf("failed to parse FIRS API response: %w", err)
	}

	fmt.Println("IRN validation successful: ", theResp)
	return theResp, nil, nil
}

func (s *Service) ValidateInvoice(invoiceReq firs_models.UploadInvoiceRequestDto, isSandbox bool) (*firs_models.FirsResponse, *string, error) {
	resp, err := firs.ValidateInvoice(invoiceReq, isSandbox)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to validate invoice: %w", err)
	}

	theResp, errDetails, err := firs.ParseFIRSAPIResponse(resp)
	if err != nil {
		return nil, errDetails, fmt.Errorf("failed to parse FIRS API response: %w", err)
	}

	fmt.Println("Invoice validation successful: ", theResp)
	return theResp, nil, nil
}

func (s *Service) SignIRN(irn string, keys *utility.CryptoKeys) (*firs_models.IRNSigningResponse, error) {
	timestamp := time.Now().UnixMilli()
	formattedIRN := fmt.Sprintf("%s.%d", irn, timestamp)

	payload := firs_models.IRNSigningData{
		IRN:         formattedIRN,
		Certificate: keys.Certificate,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal JSON: %v", err)
	}

	log.Println("sign irn payload: ", string(jsonData))

	encrypted, err := rsa.EncryptPKCS1v15(rand.Reader, keys.PublicKey, jsonData)
	if err != nil {
		return nil, fmt.Errorf("encryption failed: %v", err)
	}

	base64Encrypted := base64.StdEncoding.EncodeToString(encrypted)

	qr, err := qrcode.New(base64Encrypted, qrcode.Medium)
	if err != nil {
		return nil, fmt.Errorf("failed to generate QR code: %v", err)
	}

	buf := new(bytes.Buffer)
	if err := png.Encode(buf, qr.Image(256)); err != nil {
		return nil, fmt.Errorf("failed to encode QR code: %v", err)
	}

	base64QRImage := base64.StdEncoding.EncodeToString(buf.Bytes())

	bmpBuf := new(bytes.Buffer)
	if err := bmp.Encode(bmpBuf, qr.Image(256)); err != nil {
		return nil, fmt.Errorf("failed to encode BMP QR code: %v", err)
	}
	base64BMPImage := base64.StdEncoding.EncodeToString(bmpBuf.Bytes())

	theResp := &firs_models.IRNSigningResponse{
		EncryptedIRN:   base64Encrypted,
		QrCodeImage:    base64QRImage,
		QrCodeImageBMP: base64BMPImage,
	}

	return theResp, nil
}

func (s *Service) SignInvoice(invoiceReq firs_models.UploadInvoiceRequestDto, isSandbox bool) (*firs_models.FirsResponse, *string, error) {
	resp, err := firs.SignInvoice(invoiceReq, isSandbox)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to sign invoice: %w", err)
	}

	theResp, errDetails, err := firs.ParseFIRSAPIResponse(resp)
	if err != nil {
		return nil, errDetails, fmt.Errorf("failed to parse FIRS API response: %w", err)
	}

	fmt.Println("Invoice sign successful: ", theResp)
	return theResp, nil, nil
}

func (s *Service) GenerateIRN(invoiceNumber, serviceId string) (*string, error) {
	irn, err := common.GenerateIRN(invoiceNumber, serviceId, time.Now())
	if err != nil {
		return nil, err
	}
	return &irn, nil
}

func (s *Service) LookUpIRN(irn string) (*firs_models.FirsResponse, *string, error) {
	resp, err := firs.LookUpByIRN(irn)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get invoice: %w", err)
	}

	theResp, errDetails, err := firs.ParseFIRSAPIResponse(resp)
	if err != nil {
		return nil, errDetails, fmt.Errorf("failed to parse FIRS API response: %w", err)
	}

	return theResp, nil, nil
}

func (s *Service) LookUpTIN(tin string, isSandbox bool) (*firs_models.FirsResponse, *string, error) {
	resp, err := firs.LookUpByTIN(tin, isSandbox)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get invoices with TIN: %w", err)
	}

	theResp, errDetails, err := firs.ParseFIRSAPIResponse(resp)
	if err != nil {
		return nil, errDetails, fmt.Errorf("failed to parse FIRS API response: %w", err)
	}

	return theResp, nil, nil
}

func (s *Service) LookUpPartyID(partyId string) (*firs_models.FirsResponse, *string, error) {
	resp, err := firs.LookUpByPartyID(partyId)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get invoices with PartyID: %w", err)
	}

	theResp, errDetails, err := firs.ParseFIRSAPIResponse(resp)
	if err != nil {
		return nil, errDetails, fmt.Errorf("failed to parse FIRS API response: %w", err)
	}

	return theResp, nil, nil
}

func (s *Service) TransmitInvoice(irn string, isSandbox bool) (*firs_models.FirsResponse, *string, error) {
	resp, err := firs.TransmitInvoice(irn, isSandbox)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to transmit invoice: %w", err)
	}

	theResp, errDetails, err := firs.ParseFIRSAPIResponse(resp)
	if err != nil {
		return nil, errDetails, fmt.Errorf("failed to parse FIRS API response: %w", err)
	}

	return theResp, nil, nil
}

func (s *Service) TransmitConfirmInvoice(irn string, isSandbox bool) (*firs_models.FirsResponse, *string, error) {
	resp, err := firs.TransmitConfirmInvoice(irn, isSandbox)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to confirm transmitted invoice: %w", err)
	}

	theResp, errDetails, err := firs.ParseFIRSAPIResponse(resp)
	if err != nil {
		return nil, errDetails, fmt.Errorf("failed to parse FIRS API response: %w", err)
	}

	return theResp, nil, nil
}

func (s *Service) TransmitPull(query entities.PullDataQuery) (*firs_models.FirsResponse, *string, error) {
	resp, err := firs.TransmitPull(query)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get transmit pull: %w", err)
	}

	theResp, errDetails, err := firs.ParseFIRSAPIResponse(resp)
	if err != nil {
		return nil, errDetails, fmt.Errorf("failed to parse FIRS API response: %w", err)
	}

	return theResp, nil, nil
}

func (s *Service) DebugHealthCheck() (*firs_models.FirsResponse, *string, error) {
	resp, err := firs.DebugHealthCheck()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get health status: %w", err)
	}

	theResp, errDetails, err := firs.ParseFIRSAPIResponse(resp)
	if err != nil {
		return nil, errDetails, fmt.Errorf("failed to parse FIRS API response: %w", err)
	}

	return theResp, nil, nil
}

func (s *Service) ConfirmInvoice(irn string, isSandbox bool) (*firs_models.FirsResponse, *string, error) {
	resp, err := firs.ConfirmInvoice(irn, isSandbox)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to confirm invoice status with irn: %w", err)
	}

	theResp, errDetails, err := firs.ParseFIRSAPIResponse(resp)
	if err != nil {
		return nil, errDetails, fmt.Errorf("failed to parse FIRS API response: %w", err)
	}

	fmt.Println("Invoice confirmed successfully: ", theResp)
	return theResp, nil, nil
}

func (s *Service) DownloadInvoice(irn string, isSandbox bool) (*string, *string, error) {
	configs := config.GetConfig()
	resp, err := firs.DownloadInvoice(irn, isSandbox)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to download invoice with irn: %w", err)
	}

	theResp, errDetails, err := firs.ParseFIRSAPIResponse(resp)
	if err != nil {
		return nil, errDetails, fmt.Errorf("failed to parse FIRS API response: %w", err)
	}

	dataMap, ok := theResp.Data.(map[string]interface{})
	if !ok {
		return nil, nil, fmt.Errorf("unexpected type for response data: expected map[string]interface{}")
	}

	ivHex, ok := utility.GetString(dataMap, "iv_hex")
	if !ok {
		return nil, nil, fmt.Errorf("iv_hex not found or not a string")
	}

	pub, ok := utility.GetString(dataMap, "pub")
	if !ok {
		return nil, nil, fmt.Errorf("pub not found or not a string")
	}

	encryptedData, ok := utility.GetString(dataMap, "data")
	if !ok {
		return nil, nil, fmt.Errorf("data not found or not a string")
	}

	decrypted, err := decryptInvoice(
		ivHex,
		pub,
		encryptedData,
		configs.Firs.FirsApiKey,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to decrypt invoice: %w", err)
	}

	fmt.Println("Decrypted Invoice:\n", decrypted)
	return &decrypted, nil, nil
}

func (s *Service) UpdateInvoice(invoiceUpdate firs_models.UpdateInvoice, irn string, isSandbox bool) (*firs_models.FirsResponse, *string, error) {
	resp, err := firs.UpdateInvoice(invoiceUpdate, irn, isSandbox)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to update invoice: %w", err)
	}

	theResp, errDetails, err := firs.ParseFIRSAPIResponse(resp)
	if err != nil {
		return nil, errDetails, fmt.Errorf("failed to parse FIRS API response: %w", err)
	}

	return theResp, nil, nil
}

func decryptInvoice(ivHex string, pub string, encryptedData string, apiKey string) (string, error) {
	parts := strings.Split(apiKey, "-")
	if len(parts) < 1 {
		return "", errors.New("invalid API key format")
	}
	keyPart := parts[0]

	keyString := pub + keyPart
	if len(keyString) != 32 {
		return "", fmt.Errorf("invalid decryption key length: expected 32 bytes, got %d", len(keyString))
	}
	key := []byte(keyString)

	iv, err := hex.DecodeString(ivHex)
	if err != nil {
		return "", fmt.Errorf("failed to decode IV: %w", err)
	}

	ciphertextBytes, err := base64.URLEncoding.DecodeString(encryptedData)
	if err != nil {
		return "", fmt.Errorf("failed to decode encrypted data: %w", err)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	cfb := cipher.NewCFBDecrypter(block, iv)
	plaintext := make([]byte, len(ciphertextBytes))
	cfb.XORKeyStream(plaintext, ciphertextBytes)

	return string(plaintext), nil
}

var (
	ErrOrganizationNotFound = errors.New("organization not found")
	ErrZohoAPIUpdateFailed  = errors.New("failed to update invoice in Zoho")
	ErrInvalidSignature     = errors.New("invalid webhook signature")
)

func formatFirsError(msg string, theErr *string, err error) error {
	if theErr != nil {
		return fmt.Errorf("%s: %s - %v", msg, *theErr, err)
	}
	return fmt.Errorf("%s: %v", msg, err)
}

func (s *Service) FirsAllInOneProcess(payload firs_models.UploadInvoiceRequestDto, invoiceModel *entities.Invoice, db *gorm.DB, isSandbox bool) (error, bool) {
	pdb := dbinit.InitDB(db, false)

	_, theErr, err := s.ValidateInvoice(payload, isSandbox)
	if err != nil {
		stageErr := formatFirsError("failed to validate invoice", theErr, err)
		_ = s.repo.UpdateInvoiceStatus(pdb, invoiceModel, entities.StatusValidatedInvoice, "failed", utility.ExtractRelevantErrorMessage(stageErr))
		return stageErr, false
	}

	_ = s.repo.UpdateInvoiceStatus(pdb, invoiceModel, entities.StatusValidatedInvoice, "success")

	_, theErr, err = s.SignInvoice(payload, isSandbox)
	if err != nil {
		stageErr := formatFirsError("failed to sign invoice", theErr, err)
		_ = s.repo.UpdateInvoiceStatus(pdb, invoiceModel, entities.StatusSignedInvoice, "failed", utility.ExtractRelevantErrorMessage(stageErr))
		return stageErr, false
	}
	_ = s.repo.UpdateInvoiceStatus(pdb, invoiceModel, entities.StatusSignedInvoice, "success")

	_, theErr, err = s.TransmitInvoice(*payload.IRN, isSandbox)
	if err != nil {
		stageErr := formatFirsError("failed to transmit invoice", theErr, err)
		_ = s.repo.UpdateInvoiceStatus(pdb, invoiceModel, entities.StatusTransmitted, "failed", utility.ExtractRelevantErrorMessage(stageErr))
		return stageErr, true
	}
	_ = s.repo.UpdateInvoiceStatus(pdb, invoiceModel, entities.StatusTransmitted, "success")

	confirmInvoiceResp, theErr, err := s.ConfirmInvoice(*payload.IRN, isSandbox)
	if err != nil {
		stageErr := formatFirsError("failed to confirm invoice", theErr, err)
		_ = s.repo.UpdateInvoiceStatus(pdb, invoiceModel, entities.StatusConfirmed, "failed", utility.ExtractRelevantErrorMessage(stageErr))
		return stageErr, true
	}

	if confirmInvoiceResp.Code != 200 {
		stageErr := fmt.Errorf("failed to confirm invoice, didnt get 200 or delivered is false")
		_ = s.repo.UpdateInvoiceStatus(pdb, invoiceModel, entities.StatusConfirmed, "failed", utility.ExtractRelevantErrorMessage(stageErr))
		return stageErr, true
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
			stageErr := formatFirsError("failed to validate invoice", theErr, err)
			_ = s.repo.UpdateInvoiceStatus(pdb, invoiceModel, entities.StatusValidatedInvoice, "failed", utility.ExtractRelevantErrorMessage(stageErr))
			return stageErr, false
		}

		_ = s.repo.UpdateInvoiceStatus(pdb, invoiceModel, entities.StatusValidatedInvoice, "success")

		_, theErr, err = s.SignInvoice(payload, isSandbox)
		if err != nil {
			stageErr := formatFirsError("failed to sign invoice", theErr, err)
			_ = s.repo.UpdateInvoiceStatus(pdb, invoiceModel, entities.StatusSignedInvoice, "failed", utility.ExtractRelevantErrorMessage(stageErr))
			return stageErr, false
		}
		_ = s.repo.UpdateInvoiceStatus(pdb, invoiceModel, entities.StatusSignedInvoice, "success")

		_, theErr, err = s.TransmitInvoice(*payload.IRN, isSandbox)
		if err != nil {
			stageErr := formatFirsError("failed to transmit invoice", theErr, err)
			_ = s.repo.UpdateInvoiceStatus(pdb, invoiceModel, entities.StatusTransmitted, "failed", utility.ExtractRelevantErrorMessage(stageErr))
			return stageErr, true
		}
		_ = s.repo.UpdateInvoiceStatus(pdb, invoiceModel, entities.StatusTransmitted, "success")

		confirmInvoiceResp, theErr, err := s.ConfirmInvoice(*payload.IRN, isSandbox)
		if err != nil {
			stageErr := formatFirsError("failed to confirm invoice", theErr, err)
			_ = s.repo.UpdateInvoiceStatus(pdb, invoiceModel, entities.StatusConfirmed, "failed", utility.ExtractRelevantErrorMessage(stageErr))
			return stageErr, true
		}

		if confirmInvoiceResp.Code != 200 {
			stageErr := fmt.Errorf("failed to confirm invoice, didnt get 200 or delivered is false")
			_ = s.repo.UpdateInvoiceStatus(pdb, invoiceModel, entities.StatusConfirmed, "failed", utility.ExtractRelevantErrorMessage(stageErr))
			return stageErr, true
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
		_ = s.repo.UpdateInvoiceStatus(pdb, invoiceModel, entities.StatusGeneratedIRN, "failed", utility.ExtractRelevantErrorMessage(err))
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
		stageErr := formatFirsError("failed to validate irn", theErr, err)
		_ = s.repo.UpdateInvoiceStatus(pdb, invoiceModel, entities.StatusValidatedIRN, "failed", utility.ExtractRelevantErrorMessage(stageErr))
		return nil, nil, stageErr
	}
	_ = s.repo.UpdateInvoiceStatus(pdb, invoiceModel, entities.StatusValidatedIRN, "success")

	signIRNResp, err := s.SignIRN(*theIRN, firsKeys)
	if err != nil {
		_ = s.repo.UpdateInvoiceStatus(pdb, invoiceModel, entities.StatusSignedIRN, "failed", utility.ExtractRelevantErrorMessage(err))
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
		stageErr := formatFirsError("failed to validate invoice", theErr, err)
		_ = s.repo.UpdateInvoiceStatus(pdb, invoiceModel, entities.StatusValidatedInvoice, "failed", utility.ExtractRelevantErrorMessage(stageErr))
		return stageErr
	}
	_ = s.repo.UpdateInvoiceStatus(pdb, invoiceModel, entities.StatusValidatedInvoice, "success")

	_, theErr, err = s.SignInvoice(newInvoiceResp, isSandBox)
	if err != nil {
		stageErr := formatFirsError("failed to sign invoice", theErr, err)
		_ = s.repo.UpdateInvoiceStatus(pdb, invoiceModel, entities.StatusSignedInvoice, "failed", utility.ExtractRelevantErrorMessage(stageErr))
		return stageErr
	}
	_ = s.repo.UpdateInvoiceStatus(pdb, invoiceModel, entities.StatusSignedInvoice, "success")

	_, theErr, err = s.TransmitInvoice(*newInvoiceResp.IRN, isSandBox)
	if err != nil {
		stageErr := formatFirsError("failed to transmit invoice", theErr, err)
		_ = s.repo.UpdateInvoiceStatus(pdb, invoiceModel, entities.StatusTransmitted, "failed", utility.ExtractRelevantErrorMessage(stageErr))
		return stageErr
	}
	_ = s.repo.UpdateInvoiceStatus(pdb, invoiceModel, entities.StatusTransmitted, "success")

	confirmInvoiceResp, theErr, err := s.ConfirmInvoice(theIRN, isSandBox)
	if err != nil {
		stageErr := formatFirsError("failed to confirm invoice", theErr, err)
		_ = s.repo.UpdateInvoiceStatus(pdb, invoiceModel, entities.StatusConfirmed, "failed", utility.ExtractRelevantErrorMessage(stageErr))
		return stageErr
	}
	_ = s.repo.UpdateInvoiceStatus(pdb, invoiceModel, entities.StatusConfirmed, "success")

	if confirmInvoiceResp.Code != 200 {
		stageErr := fmt.Errorf("failed to confirm invoice, didnt get 200 or delivered is false")
		_ = s.repo.UpdateInvoiceStatus(pdb, invoiceModel, entities.StatusConfirmed, "failed", utility.ExtractRelevantErrorMessage(stageErr))
		return stageErr
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
