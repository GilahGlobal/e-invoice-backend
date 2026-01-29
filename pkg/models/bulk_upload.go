package models

import (
	"time"
)

type BulkUpload struct {
	ID                   string        `gorm:"type:uuid;primaryKey;unique;not null" json:"id"`
	FileURL              string        `gorm:"column:file_url;not null" json:"file_url"`
	FileKey              string        `gorm:"column:file_key;not null" json:"file_key"`
	Status               string        `gorm:"column:status;default:'pending'" json:"status"`
	TotalRecords         int           `gorm:"column:total_records;default:0" json:"total_records"`
	ValidRecords         int           `gorm:"column:valid_records;default:0" json:"valid_records"`
	SuccessfulInvoices   int           `gorm:"column:successful_invoices;default:0" json:"successful_invoices"`
	UnsuccessfulInvoices int           `gorm:"column:unsuccessful_invoices;default:0" json:"unsuccessful_invoices"`
	StartedAt            *time.Time    `gorm:"column:started_at" json:"started_at"`
	CompletedAt          *time.Time    `gorm:"column:completed_at" json:"completed_at"`
	CreatedAt            time.Time     `gorm:"column:created_at;default:CURRENT_TIMESTAMP" json:"created_at"`
	Duration             time.Duration `gorm:"column:duration;default:0" json:"duration"`
}
