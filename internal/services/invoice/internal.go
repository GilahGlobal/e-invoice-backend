package invoice

import (
	"einvoice-access-point/external/firs_models"
	"einvoice-access-point/internal/dtos"
	repository "einvoice-access-point/internal/repository/invoice"
	businessservice "einvoice-access-point/internal/services/business"
	"strings"

	"einvoice-access-point/pkg/database"
	inst "einvoice-access-point/pkg/dbinit"
	"einvoice-access-point/pkg/models"
	"einvoice-access-point/pkg/utility"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func GetAllInvoicesByBusinessID(db *gorm.DB, businessID string, page, size int) ([]models.MinimalInvoiceDTO, database.PaginationResponse, error) {

	pdb := inst.InitDB(db, false)

	pagination := database.Pagination{
		Page:  page,
		Limit: size,
	}

	return repository.FindMinimalInvoicesByBusinessID(pdb, businessID, pagination)
}

func GetInvoiceDetails(db *gorm.DB, businessID, invoiceID string) (*models.Invoice, error) {
	pdb := inst.InitDB(db, false)
	return repository.FindInvoiceByBusinessAndID(pdb, businessID, invoiceID)
}

func CreateInvoice(db *gorm.DB, payload dtos.UploadInvoiceRequestDto, invoiceNumber, businessID, qrCode, encryptedIRN string, invoiceExists *models.Invoice, isSandbox bool, aggregatorID *string) (*models.Invoice, *string, error, bool) {

	pdb := inst.InitDB(db, false)
	isInvoiceSigned := false
	var invoice *models.Invoice

	invoiceData, err := json.Marshal(payload)
	if err != nil {
		errDetails := "failed to marshal invoice data"
		return nil, &errDetails, fmt.Errorf("%s: %w", errDetails, err), isInvoiceSigned
	}

	currentStatus, statusHistory, err := models.InitNewInvoiceStatus()
	if err != nil {
		errDetails := "failed to initialize invoice status"
		return nil, &errDetails, fmt.Errorf("%s: %w", errDetails, err), isInvoiceSigned
	}

	platformMetadata := "{}"

	if invoiceExists != nil {
		err = UpdateInvoiceData(pdb, invoiceExists.InvoiceNumber, invoiceData)
		if err != nil {
			return nil, nil, errors.New("failed to update invoice"), isInvoiceSigned
		}

		invoice, _ = repository.FindInvoiceByNumber(pdb, invoiceExists.InvoiceNumber)
		if err, isInvoiceSigned = UncompletedFirsProcesses(db, invoiceExists.CurrentStatus, payload, invoiceExists, isSandbox); err != nil {
			errDetails := fmt.Sprintf("failed to process invoice through all steps: %v", err)
			return invoice, &errDetails, fmt.Errorf("%s", errDetails), isInvoiceSigned
		}

	} else {
		invoice = &models.Invoice{
			InvoiceNumber:    invoiceNumber,
			IRN:              *payload.IRN,
			QrCode:           qrCode,
			BusinessID:       businessID,
			Platform:         "internal",
			PlatformMetadata: platformMetadata,
			InvoiceData:      invoiceData,
			CurrentStatus:    currentStatus,
			StatusHistory:    statusHistory,
			Timestamp:        time.Now(),
			EncryptedIRN:     encryptedIRN,
			AggregatorID:     aggregatorID,
		}

		if err := repository.CreateInvoice(pdb, invoice); err != nil {
			errDetails := "failed to save invoice"
			return nil, &errDetails, fmt.Errorf("%s: %w", errDetails, err), isInvoiceSigned
		}
		if err, isInvoiceSigned = FirsAllInOneProcess(payload, invoice, db, isSandbox); err != nil {
			errDetails := fmt.Sprintf("failed to process invoice through all steps: %v", err)
			return invoice, &errDetails, fmt.Errorf("%s", errDetails), isInvoiceSigned
		}
	}

	return invoice, nil, nil, isInvoiceSigned
}

func DeleteInvoice(db *gorm.DB, businessID, invoiceID string) error {
	pdb := inst.InitDB(db, false)
	return repository.DeleteInvoiceByBusinessAndID(pdb, businessID, invoiceID)
}

func GetInvoiceByInvoiceNumber(db *gorm.DB, invoiceNumber, businessID string) (*models.Invoice, error) {
	pdb := inst.InitDB(db, false)
	return repository.FindInvoiceByNumberAndBusinessID(pdb, invoiceNumber, businessID)
}

func UpdateInvoiceData(db database.DatabaseManager, invoiceNumber string, invoiceData []byte) error {
	return repository.UpdateInvoice(db, invoiceNumber, invoiceData)
}

func IRNGeneration(db *gorm.DB, ownerID, invoiceNumber, serviceId, businessID string, isSandbox bool) (*dtos.InvoiceData, *models.Response) {
	generatedIRN, err := GenerateIRN(strings.ToUpper(invoiceNumber), serviceId)
	if err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusBadRequest, "error", err.Error(), err, nil)
		return nil, &rd
	}

	_, _, err = ValidateIRN(firs_models.IRNValidationRequest{
		InvoiceReference: invoiceNumber,
		BusinessID:       businessID,
		IRN:              *generatedIRN,
	}, isSandbox)
	if err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusBadRequest, "error", err.Error(), err, nil)
		return nil, &rd
	}

	keys, err := businessservice.ResolveBusinessIRNSigningKeys(db, ownerID, isSandbox, nil)
	if err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusBadRequest, "error", err.Error(), err, nil)
		return nil, &rd
	}

	signedIRNResponse, err := SignIRN(*generatedIRN, keys)
	if err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusBadRequest, "error", err.Error(), err, nil)
		return nil, &rd
	}
	return &dtos.InvoiceData{
		InvoiceNumber: invoiceNumber,
		IRN:           *generatedIRN,
		QRCode:        signedIRNResponse.QrCodeImage,
		QRCode2:       signedIRNResponse.EncryptedIRN,
	}, nil
}
