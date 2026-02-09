package invoice

import (
	"einvoice-access-point/external/firs"
	"einvoice-access-point/external/firs_models"
	"einvoice-access-point/pkg/config"
	"einvoice-access-point/pkg/utility"

	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
)

func ConfirmInvoice(irn string, isSandbox bool) (*firs_models.FirsResponse, *string, error) {
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

func DownloadInvoice(irn string, isSandbox bool) (*string, *string, error) {
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

	//fmt.Println("Invoice downloaded successfully: ", theResp)
	return &decrypted, nil, nil
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
