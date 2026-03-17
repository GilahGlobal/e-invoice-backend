package webhooks

import (
	"einvoice-access-point/external/firs_models"
	"einvoice-access-point/external/zoho"
	repository "einvoice-access-point/internal/repository/invoice"
	"einvoice-access-point/internal/services/converter"
	"einvoice-access-point/internal/services/invoice"
	inst "einvoice-access-point/pkg/dbinit"
	"einvoice-access-point/pkg/models"
	"einvoice-access-point/pkg/utility"
	"fmt"

	"gorm.io/gorm"
)

func FirsZohoAllInOneProcess(payload zoho.WebhookPayload, firsKeys *utility.CryptoKeys, business *models.Business,
	invoiceModel *models.Invoice, db *gorm.DB, isSandBox bool) (*string, *string, error) {

	pdb := inst.InitDB(db, false)

	theIRN, err := invoice.GenerateIRN(payload.Invoice.InvoiceNumber, business.ServiceID)
	if err != nil {
		_ = repository.UpdateInvoiceStatus(pdb, invoiceModel, models.StatusGeneratedIRN, "failed")
		return nil, nil, err
	}

	_ = repository.UpdateInvoiceStatus(pdb, invoiceModel, models.StatusGeneratedIRN, "success")

	validateIrn := firs_models.IRNValidationRequest{
		InvoiceReference: payload.Invoice.InvoiceID,
		BusinessID:       *business.BusinessID,
		IRN:              *theIRN,
	}

	_, theErr, err := invoice.ValidateIRN(validateIrn, isSandBox)
	if err != nil {
		_ = repository.UpdateInvoiceStatus(pdb, invoiceModel, models.StatusValidatedIRN, "failed")
		return nil, nil, fmt.Errorf("failed to validate irn: %v - %v", *theErr, err)
	}
	_ = repository.UpdateInvoiceStatus(pdb, invoiceModel, models.StatusValidatedIRN, "success")

	signIRNResp, err := invoice.SignIRN(*theIRN, firsKeys)
	if err != nil {
		_ = repository.UpdateInvoiceStatus(pdb, invoiceModel, models.StatusSignedIRN, "failed")
		return nil, nil, err
	}
	_ = repository.UpdateInvoiceStatus(pdb, invoiceModel, models.StatusSignedIRN, "success")
	_ = repository.UpdateInvoiceIRN(pdb, invoiceModel, *theIRN)

	go func(p zoho.WebhookPayload, b *models.Business, inv *models.Invoice, d *gorm.DB, irn string) {
		if err := otherFirsProcesses(p, b, inv, d, irn, isSandBox); err != nil {
			fmt.Println("Error in otherFirsProcesses: ", err)
		}
	}(payload, business, invoiceModel, db, *theIRN)

	return theIRN, &signIRNResp.EncryptedIRN, nil

}

func otherFirsProcesses(payload zoho.WebhookPayload, business *models.Business, invoiceModel *models.Invoice, db *gorm.DB, theIRN string, isSandBox bool) error {

	pdb := inst.InitDB(db, false)

	newInvoiceResp, err := converter.ConvertZohoToFIRS(payload.Invoice, *business.BusinessID, business.Name, theIRN)
	if err != nil {
		return err
	}

	_, theErr, err := invoice.ValidateInvoice(newInvoiceResp, isSandBox)
	if err != nil {
		_ = repository.UpdateInvoiceStatus(pdb, invoiceModel, models.StatusValidatedInvoice, "failed")
		return fmt.Errorf("failed to validate invoice: %v - %v", *theErr, err)
	}
	_ = repository.UpdateInvoiceStatus(pdb, invoiceModel, models.StatusValidatedInvoice, "success")

	_, theErr, err = invoice.SignInvoice(newInvoiceResp, isSandBox)
	if err != nil {
		_ = repository.UpdateInvoiceStatus(pdb, invoiceModel, models.StatusSignedInvoice, "failed")
		return fmt.Errorf("failed to sign invoice: %v - %v", *theErr, err)
	}
	_ = repository.UpdateInvoiceStatus(pdb, invoiceModel, models.StatusSignedInvoice, "success")

	_, theErr, err = invoice.TransmitInvoice(*newInvoiceResp.IRN)
	if err != nil {
		_ = repository.UpdateInvoiceStatus(pdb, invoiceModel, models.StatusTransmitted, "failed")
		return fmt.Errorf("failed to transmit invoice: %v - %v", *theErr, err)
	}
	_ = repository.UpdateInvoiceStatus(pdb, invoiceModel, models.StatusTransmitted, "success")

	_, theErr, err = invoice.TransmitConfirmInvoice(*newInvoiceResp.IRN)
	if err != nil {
		_ = repository.UpdateInvoiceStatus(pdb, invoiceModel, models.StatusConfirmed, "failed")
		return fmt.Errorf("failed to confirm transmit invoice: %v - %v", *theErr, err)
	}
	_ = repository.UpdateInvoiceStatus(pdb, invoiceModel, models.StatusConfirmed, "success")

	confirmInvoiceResp, theErr, err := invoice.ConfirmInvoice(theIRN, isSandBox)
	if err != nil {
		return fmt.Errorf("failed to confirm invoice: %v - %v", *theErr, err)
	}

	// data, ok := confirmInvoiceResp.Data.(map[string]interface{})
	// if !ok {
	// 	return theIRN, &signIRNResp.QrCodeImage, fmt.Errorf("unexpected data format: %#v", confirmInvoiceResp.Data)
	// }

	// delivered, _ := utility.GetBool(data, "delivered")

	if confirmInvoiceResp.Code != 200 {
		return fmt.Errorf("failed to confirm invoice, didnt get 200 or delivered is false")
	}

	return nil

}
