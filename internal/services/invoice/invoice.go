package invoice

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"einvoice-access-point/external/firs"
	"einvoice-access-point/external/firs_models"
	"einvoice-access-point/internal/dtos"
	repository "einvoice-access-point/internal/repository/invoice"
	inst "einvoice-access-point/pkg/dbinit"
	"einvoice-access-point/pkg/models"
	"einvoice-access-point/pkg/utility"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image/png"
	"regexp"
	"strings"
	"time"

	qrcode "github.com/skip2/go-qrcode"
	"gorm.io/gorm"
)

func PrepareIRN(irn string) string {
	timestamp := time.Now().UnixMilli()
	return fmt.Sprintf("%s.%d", irn, timestamp)
}

func GenerateIRNumber(invoiceNumber, serviceID string, timestamp time.Time) (string, error) {

	if !regexp.MustCompile(`^[A-Za-z0-9]+$`).MatchString(invoiceNumber) {
		return "", fmt.Errorf("invalid invoice number: only alphanumeric characters allowed")
	}

	if len(serviceID) != 8 || !regexp.MustCompile(`^[A-Za-z0-9]+$`).MatchString(serviceID) {
		return "", fmt.Errorf("invalid service ID: must be 8 alphanumeric characters")
	}

	dateString := timestamp.Format("20060102")

	irn := fmt.Sprintf("%s-%s-%s", invoiceNumber, serviceID, dateString)

	return irn, nil
}

func ValidateIRN(invoiceReq firs_models.IRNValidationRequest, isSandbox bool) (*firs_models.FirsResponse, *string, error) {

	resp, err := firs.ValidateIRN(invoiceReq, isSandbox)
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

func ValidateInvoice(invoiceReq dtos.UploadInvoiceRequestDto, isSandbox bool) (*firs_models.FirsResponse, *string, error) {

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

func SignIRN(irn string, keys *utility.CryptoKeys) (*firs_models.IRNSigningResponse, error) {
	formattedIRN := PrepareIRN(irn)

	payload := firs_models.IRNSigningData{
		IRN:         formattedIRN,
		Certificate: keys.Certificate,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal JSON: %v", err)
	}

	//encrypted, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, keys.PublicKey, jsonData, nil)
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

	theResp := &firs_models.IRNSigningResponse{
		EncryptedMessage: base64Encrypted,
		QrCodeImage:      base64QRImage,
	}

	//fmt.Printf("signed irn: %v", theResp)
	return theResp, nil
}

func SignInvoice(invoiceReq dtos.UploadInvoiceRequestDto, isSandbox bool) (*firs_models.FirsResponse, *string, error) {

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

func GenerateIRN(invoiceNumber, serviceId string) (*string, error) {
	cleanInvoiceNumber := strings.ReplaceAll(invoiceNumber, "-", "")
	irn, err := GenerateIRNumber(cleanInvoiceNumber, serviceId, time.Now())
	if err != nil {
		return nil, err
	}
	return &irn, nil
}

func AddBulkUploadLog(db *gorm.DB, fileUrl, fileKey string) error {
	pdb := inst.InitDB(db, false)

	payload := &models.BulkUpload{
		ID:      utility.GenerateUUID(),
		FileURL: fileUrl,
		FileKey: fileKey,
	}

	if err := repository.CreateBulkUploadLog(pdb, payload); err != nil {
		errDetails := "failed to save bulk upload log"
		return fmt.Errorf("%s: %w", errDetails, err)
	}
	return nil
}

func UpdateBulkUploadLog(db *gorm.DB, fileKey string, payload interface{}) error {
	pdb := inst.InitDB(db, false)

	repositoryLog, err := repository.GetBulkUploadLogByFileKey(pdb, fileKey)
	if err != nil {
		errDetails := "failed to retrieve bulk upload log"
		return fmt.Errorf("%s: %w", errDetails, err)
	}
	if repositoryLog == nil {
		errDetails := "bulk upload log not found"
		return fmt.Errorf("%s for file key: %s", errDetails, fileKey)
	}

	data, ok := payload.(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid payload type")
	}

	repositoryLog.TotalRecords = data["TotalRows"].(int)
	repositoryLog.ValidRecords = data["ValidRows"].(int)
	repositoryLog.SuccessfulInvoices = data["SuccessfulInvoices"].(int)
	repositoryLog.UnsuccessfulInvoices = data["UnsuccessfulInvoices"].(int)
	repositoryLog.Duration = data["Duration"].(time.Duration)
	repositoryLog.StartedAt = data["StartTime"].(*time.Time)
	repositoryLog.CompletedAt = data["EndTime"].(*time.Time)
	repositoryLog.Status = "completed"

	if err := repository.UpdateBulkUploadLog(pdb, fileKey, repositoryLog); err != nil {

		errDetails := "failed to update bulk upload log"
		return fmt.Errorf("%s: %w", errDetails, err)
	}
	return nil
}
