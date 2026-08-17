package utility

import (
	"strings"
)

// ExtractRelevantErrorMessage keeps the FIRS-specific validation detail and
// trims the extra parse wrapper that gets appended around it.
func ExtractRelevantErrorMessage(err error) string {
	if err == nil {
		return ""
	}

	msg := strings.TrimSpace(err.Error())
	if msg == "" {
		return ""
	}

	const marker = "failed to parse FIRS API response:"

	if idx := strings.Index(msg, marker); idx != -1 {
		prefix := strings.TrimSpace(strings.TrimSuffix(msg[:idx], " - "))
		if prefix != "" {
			return prefix
		}
	}

	return msg
}
