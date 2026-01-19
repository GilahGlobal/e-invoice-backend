package common

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"database/sql/driver"
	"encoding/base64"
	"errors"
	"io"

	"gorm.io/gorm"
)

func getAESKey() ([]byte, error) {
	var masterKey = []byte("2ZPK5UZENzbnsnxiPyvCvcgSmnGN/EtT")

	if len(masterKey) != 32 {
		return nil, errors.New("AES key must be exactly 32 bytes")
	}
	return masterKey, nil
}

func EncryptAES(text string) (string, error) {
	key, err := getAESKey()
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, aesGCM.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	cipherText := aesGCM.Seal(nonce, nonce, []byte(text), nil)
	return base64.StdEncoding.EncodeToString(cipherText), nil
}

func DecryptAES(encrypted string) (string, error) {
	key, err := getAESKey()
	if err != nil {
		return "", err
	}

	data, err := base64.StdEncoding.DecodeString(encrypted)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := aesGCM.NonceSize()
	if len(data) < nonceSize {
		return "", errors.New("invalid data")
	}

	nonce, cipherText := data[:nonceSize], data[nonceSize:]
	plainText, err := aesGCM.Open(nil, nonce, cipherText, nil)
	if err != nil {
		return "", err
	}

	return string(plainText), nil
}

type EncryptedString string

// ----- GORM Hooks -----
func (es *EncryptedString) BeforeSave(tx *gorm.DB) error {
	if es == nil || *es == "" {
		return nil
	}
	enc, err := EncryptAES(string(*es))
	if err != nil {
		return err
	}
	*es = EncryptedString(enc)
	return nil
}

func (es *EncryptedString) AfterFind(tx *gorm.DB) error {
	if es == nil || *es == "" {
		return nil
	}
	// DOn't forget to decrypt after find

	// dec, err := DecryptAES(string(*es))
	// if err != nil {
	// 	return err
	// }
	// *es = EncryptedString(dec)
	return nil
}

// ----- sql.Scanner interface -----
func (es *EncryptedString) Scan(value interface{}) error {
	if value == nil {
		*es = ""
		return nil
	}

	str, ok := value.(string)
	if !ok {
		return errors.New("EncryptedString: Scan source is not string")
	}

	if str == "" {
		*es = ""
		return nil
	}

	// Decrypt on scan
	dec, err := DecryptAES(str)
	if err != nil {
		return err
	}

	*es = EncryptedString(dec)
	return nil
}

// ----- driver.Valuer interface -----
func (es EncryptedString) Value() (driver.Value, error) {
	if es == "" {
		return nil, nil
	}

	// Encrypt on save
	enc, err := EncryptAES(string(es))
	if err != nil {
		return nil, err
	}
	return enc, nil
}
