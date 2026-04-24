package business

import (
	"einvoice-access-point/pkg/common"
	inst "einvoice-access-point/pkg/dbinit"
	"einvoice-access-point/pkg/utility"
	"errors"
	"fmt"
	"strings"

	"gorm.io/gorm"
)

func SaveBusinessIRNSigningKeys(db *gorm.DB, id string, fileContent []byte) error {
	document, err := utility.ParseCryptoKeyDocument(fileContent)
	if err != nil {
		return err
	}

	if _, err := utility.NewCryptoKeys(document.PublicKey, document.Certificate); err != nil {
		return fmt.Errorf("invalid crypto keys document: %w", err)
	}

	business, err := GetBusinessDetails(db, id)
	if err != nil {
		return err
	}

	business.IRNPublicKey = common.EncryptedString(document.PublicKey)
	business.IRNCertificate = common.EncryptedString(document.Certificate)
	business.KeysSet = true

	pdb := inst.InitDB(db, false)
	if _, err := pdb.SaveAllFields(business); err != nil {
		return fmt.Errorf("failed to save business IRN signing keys: %w", err)
	}

	return nil
}

func ResolveBusinessIRNSigningKeys(db *gorm.DB, id string, isSandbox bool, fallbackKeys *utility.CryptoKeys) (*utility.CryptoKeys, error) {
	if isSandbox {
		if fallbackKeys != nil {
			return fallbackKeys, nil
		}

		return utility.LoadCryptoKeys("crypto_keys.txt")
	}

	business, err := GetBusinessDetails(db, id)
	if err != nil {
		return nil, err
	}

	publicKey := strings.TrimSpace(string(business.IRNPublicKey))
	certificate := strings.TrimSpace(string(business.IRNCertificate))
	if publicKey == "" || certificate == "" {
		return nil, errors.New("business IRN signing keys have not been configured")
	}

	keys, err := utility.NewCryptoKeys(publicKey, certificate)
	if err != nil {
		return nil, fmt.Errorf("failed to parse saved business IRN signing keys: %w", err)
	}

	return keys, nil
}
