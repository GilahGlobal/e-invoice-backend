package utility

import (
	"fmt"
	"math/rand"
	"strings"
	"time"
)

func GenerateRandomServiceID() string {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	rand.Seed(time.Now().UnixNano())

	serviceID := make([]byte, 8)
	for i := range serviceID {
		serviceID[i] = charset[rand.Intn(len(charset))]
	}

	return string(serviceID)
}

func ExtractServiceIDFromIRN(template string) (string, error) {
	parts := strings.Split(template, "-")
	if len(parts) != 3 {
		return "", fmt.Errorf("invalid template format")
	}
	return parts[1], nil
}
