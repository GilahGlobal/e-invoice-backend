package models

import (
	"time"

	"gorm.io/datatypes"
)

type BulkUpload struct {
	ID                          string         `gorm:"type:uuid;primaryKey;unique;not null" json:"id"`
	FileURL                     string         `gorm:"column:file_url;not null" json:"file_url"`
	FileKey                     string         `gorm:"column:file_key;not null" json:"file_key"`
	BusinessID                  string         `gorm:"column:business_id;type:uuid;index" json:"business_id"`
	AggregatorID                *string        `gorm:"column:aggregator_id;type:uuid;index" json:"aggregator_id,omitempty"`
	Status                      string         `gorm:"column:status;default:'pending'" json:"status"`
	TotalRecords                int            `gorm:"column:total_records;default:0" json:"total_records"`
	ValidRecords                int            `gorm:"column:valid_records;default:0" json:"valid_records"`
	SuccessfulInvoices          int            `gorm:"column:successful_invoices;default:0" json:"successful_invoices"`
	PartiallySuccessfulInvoices int            `gorm:"column:partially_successful_invoices;default:0" json:"partially_successful_invoices"`
	UnsuccessfulInvoices        int            `gorm:"column:unsuccessful_invoices;default:0" json:"unsuccessful_invoices"`
	ValidationErrorCount        int            `gorm:"column:validation_error_count;default:0" json:"validation_error_count"`
	ValidationErrors            datatypes.JSON `gorm:"type:jsonb;not null;default:'[]'" json:"validation_errors" swaggertype:"object"`
	StartedAt                   *time.Time     `gorm:"column:started_at" json:"started_at"`
	CompletedAt                 *time.Time     `gorm:"column:completed_at" json:"completed_at"`
	CreatedAt                   time.Time      `gorm:"column:created_at;default:CURRENT_TIMESTAMP" json:"created_at"`
	Duration                    time.Duration  `gorm:"column:duration;default:0" json:"duration" swaggertype:"string"`
}
