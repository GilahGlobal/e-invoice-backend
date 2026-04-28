package invoice

import (
	"einvoice-access-point/external/firs"
	"einvoice-access-point/external/firs_models"
	repository "einvoice-access-point/internal/repository/invoice"
	inst "einvoice-access-point/pkg/dbinit"
	"encoding/json"
	"fmt"

	"gorm.io/gorm"
)

func UpdateInvoice(invoiceUpdate firs_models.UpdateInvoice, irn string, isSandbox bool) (*firs_models.FirsResponse, *string, error) {

	resp, err := firs.UpdateInvoice(invoiceUpdate, irn, isSandbox)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to validate irn: %w", err)
	}

	theResp, errDetails, err := firs.ParseFIRSAPIResponse(resp)
	if err != nil {
		return nil, errDetails, fmt.Errorf("failed to parse FIRS API response: %w", err)
	}

	//fmt.Println("IRN validation successful: ", theResp)
	return theResp, nil, nil
}

func UpdateStoredInvoicePaymentStatus(db *gorm.DB, businessID, irn, paymentStatus string) error {
	pdb := inst.InitDB(db, false)

	invoiceRecord, err := repository.FindInvoiceByIRNAndBusinessID(pdb, irn, businessID)
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

	if err := repository.UpdateInvoiceDataByID(pdb, invoiceRecord.ID, updatedInvoiceData); err != nil {
		return fmt.Errorf("failed to update local invoice record: %w", err)
	}

	return nil
}
