package utility

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"os"
	"strings"
)

type CryptoKeys struct {
	PublicKey   *rsa.PublicKey
	Certificate string
}

type CryptoKeyDocument struct {
	PublicKey   string `json:"public_key"`
	Certificate string `json:"certificate"`
}

func ParseCryptoKeyDocument(content []byte) (*CryptoKeyDocument, error) {
	var kd CryptoKeyDocument
	if err := json.Unmarshal(content, &kd); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %v", err)
	}

	kd.PublicKey = strings.TrimSpace(kd.PublicKey)
	kd.Certificate = strings.TrimSpace(kd.Certificate)

	if kd.PublicKey == "" {
		return nil, fmt.Errorf("public_key is required")
	}
	if kd.Certificate == "" {
		return nil, fmt.Errorf("certificate is required")
	}

	return &kd, nil
}

func NewCryptoKeys(publicKey, certificate string) (*CryptoKeys, error) {
	pemData, err := base64.StdEncoding.DecodeString(strings.TrimSpace(publicKey))
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64 public_key: %v", err)
	}

	block, _ := pem.Decode(pemData)
	if block == nil || block.Type != "PUBLIC KEY" {
		return nil, fmt.Errorf("invalid PEM block for public key")
	}

	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse public key: %v", err)
	}

	rsaPub, ok := pub.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("not an RSA public key")
	}

	return &CryptoKeys{
		PublicKey:   rsaPub,
		Certificate: strings.TrimSpace(certificate),
	}, nil
}

func LoadCryptoKeys(filename string) (*CryptoKeys, error) {
	content, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %v", filename, err)
	}

	doc, err := ParseCryptoKeyDocument(content)
	if err != nil {
		return nil, err
	}

	return NewCryptoKeys(doc.PublicKey, doc.Certificate)
}
