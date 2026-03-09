package models

import (
	"time"
)

// Business represents a business entity with dynamic platform support
type SME struct {
	ID           string    `gorm:"type:uuid;primaryKey;unique;not null" json:"id"`
	Name         string    `gorm:"column:name;type:varchar(250);not null" json:"name"`
	Email        string    `gorm:"column:email;type:varchar(100);unique" json:"email"`
	Password     string    `gorm:"column:password;type:text;not null" json:"-"`
	TIN          string    `gorm:"column:tin;type:varchar(20)" json:"tin"`
	PhoneNumber  string    `gorm:"column:phone_number;type:varchar(13)" json:"phone_number"`
	CompanyName  string    `gorm:"column:company_name;type:varchar(250)" json:"company_name"`
	AggregatorID string    `gorm:"column:aggregator_id;type:uuid" json:"aggregator_id"`
	CreatedAt    time.Time `gorm:"column:created_at;not null;autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time `gorm:"column:updated_at;null;autoUpdateTime" json:"updated_at"`
}
