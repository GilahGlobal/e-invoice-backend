package business

import (
	"fmt"
	"strings"

	"gorm.io/gorm"
)

type InvoiceUploadSetup struct {
	BusinessID string
	ServiceID  string
}

func ValidateInvoiceUploadSetup(db *gorm.DB, ownerID string) (*InvoiceUploadSetup, error) {
	business, err := GetBusinessDetails(db, ownerID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve business details: %w", err)
	}

	missing := make([]string, 0, 3)

	businessID := ""
	if business.BusinessID != nil {
		businessID = strings.TrimSpace(*business.BusinessID)
	}
	if businessID == "" {
		missing = append(missing, "business id")
	}

	serviceID := ""
	if business.ServiceID != nil {
		serviceID = strings.TrimSpace(*business.ServiceID)
	}
	if serviceID == "" {
		missing = append(missing, "service id")
	}

	irnPublicKey := strings.TrimSpace(string(business.IRNPublicKey))
	irnCertificate := strings.TrimSpace(string(business.IRNCertificate))
	if !business.KeysSet || irnPublicKey == "" || irnCertificate == "" {
		missing = append(missing, "crypto keys")
	}

	if len(missing) > 0 {
		return nil, fmt.Errorf("cannot upload invoice: missing required setup: %s", strings.Join(missing, ", "))
	}

	return &InvoiceUploadSetup{
		BusinessID: businessID,
		ServiceID:  serviceID,
	}, nil
}
