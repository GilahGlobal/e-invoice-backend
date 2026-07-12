package invoice

import (
	"einvoice-access-point/internal/data/database"
	"einvoice-access-point/internal/data/dbinit"
	"einvoice-access-point/internal/data/entities"
	"einvoice-access-point/internal/pkg/firs_models"
	"einvoice-access-point/internal/utility"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func (s *Service) GetAllInvoicesByBusinessID(db *gorm.DB, businessID string, page, size int) ([]entities.MinimalInvoiceDTO, database.PaginationResponse, error) {
	pdb := dbinit.InitDB(db, false)

	pagination := database.Pagination{
		Page:  page,
		Limit: size,
	}

	return s.repo.FindMinimalInvoicesByBusinessID(pdb, businessID, pagination)
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
			errorArray := strings.Split(err.Error(), "-")
			return invoice, nil, errors.New(errorArray[0]), isInvoiceSigned
		}
	} else {
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
			errorArray := strings.Split(err.Error(), "-")
			return invoice, nil, errors.New(errorArray[0]), isInvoiceSigned
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

func (s *Service) UpdateInvoiceData(db database.DatabaseManager, invoiceNumber string, invoiceData []byte) error {
	return s.repo.UpdateInvoice(db, invoiceNumber, invoiceData)
}

func (s *Service) IRNGeneration(db *gorm.DB, ownerID, invoiceNumber, serviceId, businessID string, isSandbox bool) (*InvoiceData, *entities.Response) {
	generatedIRN, err := s.GenerateIRN(strings.ToUpper(invoiceNumber), serviceId)
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

	if err := s.repo.UpdateInvoiceDataByID(pdb, invoiceRecord.ID, updatedInvoiceData); err != nil {
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
