package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

const (
	StatusCreated          = "created"
	StatusGeneratedIRN     = "generated_irn"
	StatusValidatedIRN     = "validated_irn"
	StatusSignedIRN        = "signed_irn"
	StatusValidatedInvoice = "validated_invoice"
	StatusSignedInvoice    = "signed_invoice"
	StatusTransmitted      = "transmitted_invoice"
	StatusConfirmed        = "confirmed_invoice"
)

// StatusHistoryEntry represents one step in the invoice process
type StatusHistoryEntry struct {
	Step      string    `json:"step"`
	Status    string    `json:"status"` // success | pending | failed
	Timestamp time.Time `json:"timestamp"`
}

// Invoice represents an invoice with support for multiple accounting platforms
type Invoice struct {
	ID               string         `gorm:"type:uuid;primaryKey;unique;not null" json:"id"`
	InvoiceNumber    string         `gorm:"column:invoice_number;type:varchar(50);not null;index" json:"invoice_number"`
	IRN              string         `gorm:"column:irn;type:varchar(50);null" json:"irn"`
	QrCode           string         `gorm:"column:qr_code;type:text;null" json:"qr_code"`
	EncryptedIRN     string         `gorm:"column:encrypted_irn;type:text;null" json:"encrypted_irn"`
	BusinessID       string         `gorm:"column:business_id;type:uuid;not null" json:"business_id"`
	Platform         string         `gorm:"column:platform;type:varchar(20);not null" json:"platform"` // e.g., zoho, quickbooks
	PlatformMetadata string         `gorm:"type:jsonb;not null;default:'{}'" json:"platform_metadata"`
	InvoiceData      datatypes.JSON `gorm:"type:jsonb;not null;default:'{}'" json:"invoice_data"`
	CurrentStatus    string         `gorm:"column:current_status;type:varchar(50);not null;default:'created'" json:"current_status"`
	StatusHistory    datatypes.JSON `gorm:"type:jsonb;not null;default:'[]'" json:"status_history"`
	Timestamp        time.Time      `gorm:"column:timestamp;not null" json:"timestamp"`
	CreatedAt        time.Time      `gorm:"column:created_at;not null;autoCreateTime" json:"created_at"`
	UpdatedAt        time.Time      `gorm:"column:updated_at;null;autoUpdateTime" json:"updated_at"`
	DeletedAt        gorm.DeletedAt `gorm:"index" json:"-"`
}

// PlatformMetadata represents platform-specific invoice data
type PlatformMetadata map[string]InvoicePlatformData

// InvoicePlatformData holds platform-specific invoice details
type InvoicePlatformData struct {
	InvoiceID     string  `json:"invoice_id"`                // Platform-specific invoice/document ID
	Status        string  `json:"status"`                    // e.g., sent, paid
	Total         float64 `json:"total"`                     // Invoice total
	CurrencyCode  string  `json:"currency_code"`             // e.g., USD
	ExternalRefID string  `json:"external_ref_id,omitempty"` // Optional external reference
}

type MinimalInvoiceDTO struct {
	ID            string    `json:"id"`
	InvoiceNumber string    `json:"invoice_number"`
	IRN           string    `json:"irn"`
	Platform      string    `json:"platform"`
	CurrentStatus string    `json:"current_status"`
	StatusText    string    `json:"status_text"`
	CreatedAt     time.Time `json:"created_at"`
}

// BeforeCreate sets the ID if not provided
func (i *Invoice) BeforeCreate(tx *gorm.DB) error {
	if i.ID == "" {
		i.ID = uuid.New().String()
	}
	return nil
}

var defaultSteps = []string{
	StatusCreated,
	StatusGeneratedIRN,
	StatusValidatedIRN,
	StatusSignedIRN,
	StatusValidatedInvoice,
	StatusSignedInvoice,
	StatusTransmitted,
	StatusConfirmed,
}

func InitInvoiceStatus() (string, datatypes.JSON, error) {
	var history []StatusHistoryEntry
	for _, step := range defaultSteps {
		history = append(history, StatusHistoryEntry{
			Step:      step,
			Status:    "pending",
			Timestamp: time.Now(),
		})
	}

	historyJSON, err := json.Marshal(history)
	if err != nil {
		return "", nil, err
	}

	return StatusCreated, historyJSON, nil
}

// InitNewInvoiceStatus initializes CurrentStatus and StatusHistory for a new invoice
func InitNewInvoiceStatus() (string, datatypes.JSON, error) {
	var history []StatusHistoryEntry

	successSteps := map[string]bool{
		StatusCreated:      true,
		StatusGeneratedIRN: true,
		StatusValidatedIRN: true,
		StatusSignedIRN:    true,
	}

	for _, step := range defaultSteps {
		status := "pending"
		if successSteps[step] {
			status = "success"
		}

		history = append(history, StatusHistoryEntry{
			Step:      step,
			Status:    status,
			Timestamp: time.Now(),
		})
	}

	historyJSON, err := json.Marshal(history)
	if err != nil {
		return "", nil, err
	}

	return StatusSignedIRN, historyJSON, nil
}

func InitPlatformInvoiceStatus() (string, datatypes.JSON, error) {
	var history []StatusHistoryEntry

	successSteps := map[string]bool{
		StatusCreated: true,
	}

	for _, step := range defaultSteps {
		status := "pending"
		if successSteps[step] {
			status = "success"
		}

		history = append(history, StatusHistoryEntry{
			Step:      step,
			Status:    status,
			Timestamp: time.Now(),
		})
	}

	historyJSON, err := json.Marshal(history)
	if err != nil {
		return "", nil, err
	}

	return StatusSignedIRN, historyJSON, nil
}
