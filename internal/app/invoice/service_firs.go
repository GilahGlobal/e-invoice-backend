package invoice

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"einvoice-access-point/internal/config"
	"einvoice-access-point/internal/data/entities"
	"einvoice-access-point/internal/pkg/firs"
	"einvoice-access-point/internal/pkg/firs_models"
	"einvoice-access-point/internal/utility"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"image/png"
	"log"
	"regexp"
	"strings"
	"time"

	"golang.org/x/image/bmp"

	qrcode "github.com/skip2/go-qrcode"
)

func (s *Service) PrepareIRN(irn string) string {
	timestamp := time.Now().UnixMilli()
	return fmt.Sprintf("%s.%d", irn, timestamp)
}

func (s *Service) GenerateIRNumber(invoiceNumber, serviceID string, timestamp time.Time) (string, error) {
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
	formattedIRN := s.PrepareIRN(irn)

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
	cleanInvoiceNumber := strings.ReplaceAll(invoiceNumber, "-", "")
	irn, err := s.GenerateIRNumber(cleanInvoiceNumber, serviceId, time.Now())
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

func (s *Service) LookUpTIN(tin string) (*firs_models.FirsResponse, *string, error) {
	resp, err := firs.LookUpByTIN(tin)
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
