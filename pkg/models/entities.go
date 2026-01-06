package models

import (
	"database/sql/driver"
	"einvoice-access-point/pkg/common"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"gorm.io/gorm"
)

// Business represents a business entity with dynamic platform support
type Business struct {
	ID              string                 `gorm:"type:uuid;primaryKey;unique;not null" json:"id"`
	Name            string                 `gorm:"column:name;type:varchar(250);not null;unique" json:"name"`
	Email           string                 `gorm:"column:email;type:varchar(100);unique" json:"email"`
	Password        string                 `gorm:"column:password;type:text;not null" json:"-"`
	AccStatus       int                    `gorm:"column:acc_status;type:int;default:0" json:"acc_status"`
	APIKey          common.EncryptedString `gorm:"type:text" json:"api_key"`
	APIKeyHash      string                 `gorm:"type:text;index" json:"-"`
	BusinessID      string                 `gorm:"column:business_id;type:uuid;not null;index" json:"business_id"`
	ServiceID       string                 `gorm:"column:service_id;type:varchar(20);not null;index" json:"service_id"`
	TIN             string                 `gorm:"column:tin;type:varchar(20)" json:"tin"`
	PhoneNumber     string                 `gorm:"column:phone_number;type:varchar(13)" json:"phone_number"`
	CompanyName     string                 `gorm:"column:company_name;type:varchar(250)" json:"company_name"`
	PlatformConfigs PlatformConfigs        `gorm:"type:jsonb;not null;default:'{}'" json:"platform_configs"`
	Invoices        []Invoice              `gorm:"foreignKey:BusinessID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"invoices"`
	CreatedAt       time.Time              `gorm:"column:created_at;not null;autoCreateTime" json:"created_at"`
	UpdatedAt       time.Time              `gorm:"column:updated_at;null;autoUpdateTime" json:"updated_at"`
	DeletedAt       gorm.DeletedAt         `gorm:"index" json:"-"`
}

// PlatformConfigs represents the structure for platform-specific configurations
type PlatformConfigs map[string]AccountingPlatformConfig

// AccountingPlatformConfig holds configuration for a single accounting platform
type AccountingPlatformConfig struct {
	OrgID      string                 `json:"org_id"`
	AuthToken  common.EncryptedString `json:"auth_token"`
	HMACSecret common.EncryptedString `json:"hmac_secret"`
	APIKey     common.EncryptedString `json:"api_key"`
	APISecret  common.EncryptedString `json:"api_secret"`
}

// BeforeCreate sets the ID if not provided
func (b *Business) BeforeCreate(tx *gorm.DB) error {
	if b.ID == "" {
		b.ID = uuid.New().String()
	}
	if b.BusinessID == "" {
		b.BusinessID = uuid.New().String()
	}
	return nil
}

func (pc PlatformConfigs) Decrypt() (PlatformConfigs, error) {
	result := make(PlatformConfigs)
	for key, cfg := range pc {
		if err := cfg.AuthToken.AfterFind(nil); err != nil {
			return nil, fmt.Errorf("failed to decrypt AuthToken for platform %s: %w", key, err)
		}
		if err := cfg.HMACSecret.AfterFind(nil); err != nil {
			return nil, fmt.Errorf("failed to decrypt HMACSecret for platform %s: %w", key, err)
		}
		if err := cfg.APIKey.AfterFind(nil); err != nil {
			return nil, fmt.Errorf("failed to decrypt APIKey for platform %s: %w", key, err)
		}
		if err := cfg.APISecret.AfterFind(nil); err != nil {
			return nil, fmt.Errorf("failed to decrypt APISecret for platform %s: %w", key, err)
		}

		result[key] = cfg
	}
	return result, nil
}

// Value implements the driver.Valuer interface, so PlatformConfigs can be saved into DB
func (pc PlatformConfigs) Value() (driver.Value, error) {
	if len(pc) == 0 {
		return "{}", nil
	}
	return json.Marshal(pc)
}

// Scan implements the sql.Scanner interface, so PlatformConfigs can be read from DB
func (pc *PlatformConfigs) Scan(value interface{}) error {
	if value == nil {
		*pc = make(PlatformConfigs)
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("failed to unmarshal PlatformConfigs: %v", value)
	}

	var result map[string]AccountingPlatformConfig
	if err := json.Unmarshal(bytes, &result); err != nil {
		return err
	}

	*pc = result
	return nil
}

type PaginationQuery struct {
	Size              int    `query:"size"`
	Page              int    `query:"page"`
	SortBy            string `query:"sort_by"`
	SortDirectionDesc bool   `query:"sort_direction_desc"`
	Reference         string `query:"reference"`
}

type PullDataQuery struct {
	Confirmed string `query:"confirmed"`
	From      string `query:"from"`
	To        string `query:"to"`
}
