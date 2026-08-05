package common

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

var irnAlphanumeric = regexp.MustCompile(`^[A-Za-z0-9]+$`)

// GenerateIRN builds the canonical IRN string from an invoice number and service ID.
func GenerateIRN(invoiceNumber, serviceID string, timestamp time.Time) (string, error) {
	cleanInvoiceNumber := strings.ToUpper(strings.ReplaceAll(invoiceNumber, "-", ""))
	if !irnAlphanumeric.MatchString(cleanInvoiceNumber) {
		return "", fmt.Errorf("invalid invoice number: only alphanumeric characters allowed")
	}

	if len(serviceID) != 8 || !irnAlphanumeric.MatchString(serviceID) {
		return "", fmt.Errorf("invalid service ID: must be 8 alphanumeric characters")
	}

	return fmt.Sprintf("%s-%s-%s", cleanInvoiceNumber, serviceID, timestamp.Format("20060102")), nil
}
