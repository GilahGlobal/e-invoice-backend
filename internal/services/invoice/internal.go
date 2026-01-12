package invoice

import (
	"einvoice-access-point/external/firs_models"
	repository "einvoice-access-point/internal/repository/invoice"
	"einvoice-access-point/pkg/database"
	inst "einvoice-access-point/pkg/dbinit"
	"einvoice-access-point/pkg/models"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
)

func GetAllInvoicesByBusinessID(db *gorm.DB, businessID string) ([]models.MinimalInvoiceDTO, error) {

	pdb := inst.InitDB(db, true)

	return repository.FindMinimalInvoicesByBusinessID(pdb, businessID)
}

func GetInvoiceDetails(db *gorm.DB, businessID, invoiceID string) (*models.Invoice, error) {
	pdb := inst.InitDB(db, true)
	return repository.FindInvoiceByBusinessAndID(pdb, businessID, invoiceID)
}

func CreateInvoice(db *gorm.DB, payload firs_models.InvoiceRequest, invoiceNumber, businessID string, invoiceExists *models.Invoice) (*models.Invoice, *string, error, bool) {

	pdb := inst.InitDB(db, true)
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
		if err, isInvoiceSigned = UncompletedFirsProcesses(db, invoiceExists.CurrentStatus, payload, invoiceExists); err != nil {
			errDetails := fmt.Sprintf("failed to process invoice through all steps: %v", err)
			return invoice, &errDetails, fmt.Errorf("%s", errDetails), isInvoiceSigned
		}

	} else {
		invoice = &models.Invoice{
			InvoiceNumber:    invoiceNumber,
			IRN:              *payload.IRN,
			BusinessID:       businessID,
			Platform:         "internal",
			PlatformMetadata: platformMetadata,
			InvoiceData:      invoiceData,
			CurrentStatus:    currentStatus,
			StatusHistory:    statusHistory,
			Timestamp:        time.Now(),
		}

		if err := repository.CreateInvoice(pdb, invoice); err != nil {
			errDetails := "failed to save invoice"
			return nil, &errDetails, fmt.Errorf("%s: %w", errDetails, err), isInvoiceSigned
		}
		if err, isInvoiceSigned = FirsAllInOneProcess(payload, invoice, db); err != nil {
			errDetails := fmt.Sprintf("failed to process invoice through all steps: %v", err)
			return invoice, &errDetails, fmt.Errorf("%s", errDetails), isInvoiceSigned
		}
	}

	return invoice, nil, nil, isInvoiceSigned
}

func DeleteInvoice(db *gorm.DB, businessID, invoiceID string) error {
	pdb := inst.InitDB(db, true)
	return repository.DeleteInvoiceByBusinessAndID(pdb, businessID, invoiceID)
}

func GetInvoiceByInvoiceNumber(db *gorm.DB, invoiceNumber, businessID string) (*models.Invoice, error) {
	pdb := inst.InitDB(db, true)
	return repository.FindInvoiceByNumberAndBusinessID(pdb, invoiceNumber, businessID)
}

func UpdateInvoiceData(db database.DatabaseManager, invoiceNumber string, invoiceData []byte) error {
	return repository.UpdateInvoice(db, invoiceNumber, invoiceData)
}
