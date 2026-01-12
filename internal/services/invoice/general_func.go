package invoice

import (
	"einvoice-access-point/external/firs_models"
	repository "einvoice-access-point/internal/repository/invoice"
	inst "einvoice-access-point/pkg/dbinit"
	"einvoice-access-point/pkg/models"
	"fmt"

	"gorm.io/gorm"
)

func FirsAllInOneProcess(payload firs_models.InvoiceRequest, invoiceModel *models.Invoice, db *gorm.DB) (error, bool) {

	pdb := inst.InitDB(db, true)

	_, theErr, err := ValidateInvoice(payload)
	if err != nil {
		_ = repository.UpdateInvoiceStatus(pdb, invoiceModel, models.StatusValidatedInvoice, "failed")
		return fmt.Errorf("failed to validate invoice: %v - %v", *theErr, err), false
	}

	err = repository.UpdateInvoiceStatus(pdb, invoiceModel, models.StatusValidatedInvoice, "success")
	if err != nil {
		return fmt.Errorf("failed to update invoice status: %v", err), false
	}

	_, theErr, err = SignInvoice(payload)
	if err != nil {
		_ = repository.UpdateInvoiceStatus(pdb, invoiceModel, models.StatusSignedInvoice, "failed")
		return fmt.Errorf("failed to sign invoice: %v - %v", *theErr, err), false
	}
	err = repository.UpdateInvoiceStatus(pdb, invoiceModel, models.StatusSignedInvoice, "success")
	if err != nil {
		return fmt.Errorf("failed to update invoice status: %v", err), false
	}

	_, theErr, err = TransmitInvoice(*payload.IRN)
	if err != nil {
		_ = repository.UpdateInvoiceStatus(pdb, invoiceModel, models.StatusTransmitted, "failed")
		return fmt.Errorf("failed to transmit invoice: %v - %v", *theErr, err), true
	}
	err = repository.UpdateInvoiceStatus(pdb, invoiceModel, models.StatusTransmitted, "success")
	if err != nil {
		return fmt.Errorf("failed to update invoice status: %v", err), true
	}

	_, theErr, err = TransmitConfirmInvoice(*payload.IRN)
	if err != nil {
		_ = repository.UpdateInvoiceStatus(pdb, invoiceModel, models.StatusConfirmed, "failed")
		return fmt.Errorf("failed to confirm transmit invoice: %v - %v", *theErr, err), true
	}
	err = repository.UpdateInvoiceStatus(pdb, invoiceModel, models.StatusConfirmed, "success")
	if err != nil {
		return fmt.Errorf("failed to update invoice status: %v", err), true
	}

	confirmInvoiceResp, theErr, err := ConfirmInvoice(*payload.IRN)
	if err != nil {
		return fmt.Errorf("failed to confirm invoice: %v - %v", *theErr, err), true
	}

	if confirmInvoiceResp.Code != 200 {
		return fmt.Errorf("failed to confirm invoice, didnt get 200 or delivered is false"), true
	}

	return nil, true
}

func UncompletedFirsProcesses(db *gorm.DB, currentStatus string, payload firs_models.InvoiceRequest, invoiceModel *models.Invoice) (error, bool) {
	pdb := inst.InitDB(db, true)
	switch currentStatus {
	case models.StatusValidatedInvoice:
		_, theErr, err := ValidateInvoice(payload)
		if err != nil {
			_ = repository.UpdateInvoiceStatus(pdb, invoiceModel, models.StatusValidatedInvoice, "failed")
			return fmt.Errorf("failed to validate invoice: %v - %v", *theErr, err), false
		}

		err = repository.UpdateInvoiceStatus(pdb, invoiceModel, models.StatusValidatedInvoice, "success")
		if err != nil {
			return fmt.Errorf("failed to update invoice status: %v", err), false
		}

		_, theErr, err = SignInvoice(payload)
		if err != nil {
			_ = repository.UpdateInvoiceStatus(pdb, invoiceModel, models.StatusSignedInvoice, "failed")
			return fmt.Errorf("failed to sign invoice: %v - %v", *theErr, err), false
		}
		err = repository.UpdateInvoiceStatus(pdb, invoiceModel, models.StatusSignedInvoice, "success")
		if err != nil {
			return fmt.Errorf("failed to update invoice status: %v", err), false
		}

		_, theErr, err = TransmitInvoice(*payload.IRN)
		if err != nil {
			_ = repository.UpdateInvoiceStatus(pdb, invoiceModel, models.StatusTransmitted, "failed")
			return fmt.Errorf("failed to transmit invoice: %v - %v", *theErr, err), true
		}
		err = repository.UpdateInvoiceStatus(pdb, invoiceModel, models.StatusTransmitted, "success")
		if err != nil {
			return fmt.Errorf("failed to update invoice status: %v", err), true
		}

		_, theErr, err = TransmitConfirmInvoice(*payload.IRN)
		if err != nil {
			_ = repository.UpdateInvoiceStatus(pdb, invoiceModel, models.StatusConfirmed, "failed")
			return fmt.Errorf("failed to confirm transmit invoice: %v - %v", *theErr, err), true
		}
		err = repository.UpdateInvoiceStatus(pdb, invoiceModel, models.StatusConfirmed, "success")
		if err != nil {
			return fmt.Errorf("failed to update invoice status: %v", err), true
		}

		confirmInvoiceResp, theErr, err := ConfirmInvoice(*payload.IRN)
		if err != nil {
			return fmt.Errorf("failed to confirm invoice: %v - %v", *theErr, err), true
		}

		if confirmInvoiceResp.Code != 200 {
			return fmt.Errorf("failed to confirm invoice, didnt get 200 or delivered is false"), true
		}

		return nil, true
	case models.StatusSignedIRN:
		return FirsAllInOneProcess(payload, invoiceModel, db)
	default:
		return fmt.Errorf("unknown status: %s", currentStatus), false
	}
}
