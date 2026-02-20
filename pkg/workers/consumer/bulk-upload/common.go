package bulkupload

import (
	"bytes"
	"encoding/binary"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"
)

// DetermineFileType identifies whether the file is CSV or Excel based on content
// and optionally filename. Returns "csv", "excel", or "unknown"
func DetermineFileType(fileBytes []byte, filename string) string {
	// 1. First try to detect Excel by its signatures
	if isExcelSignature(fileBytes) {
		return "excel"
	}
	// 2. If we have a filename, use extension as hint
	if filename != "" {
		ext := strings.ToLower(filepath.Ext(filename))
		switch ext {
		case ".xlsx", ".xls", ".xlsm", ".xlsb":
			return "excel" // Trust extension if we didn't detect signature
		case ".csv", ".tsv":
			return "csv"
		}
	}

	// 3. Check if content looks like CSV
	if isCSVContent(fileBytes) {
		return "csv"
	}

	return "unknown"
}

// isExcelSignature checks for Excel file signatures
func isExcelSignature(data []byte) bool {
	if len(data) < 8 {
		return false
	}

	// Excel .xlsx/.xlsm/.xlsb (ZIP-based formats)
	// ZIP files start with PK\x03\x04 or PK\x05\x06 or PK\x07\x08
	if bytes.HasPrefix(data, []byte{0x50, 0x4B, 0x03, 0x04}) ||
		bytes.HasPrefix(data, []byte{0x50, 0x4B, 0x05, 0x06}) ||
		bytes.HasPrefix(data, []byte{0x50, 0x4B, 0x07, 0x08}) {
		return true
	}

	// Excel .xls (OLE Compound Document)
	// Signature: D0 CF 11 E0 A1 B1 1A E1
	if len(data) >= 8 {
		signature := binary.LittleEndian.Uint64(data[:8])
		if signature == 0xE11AB1A0E011CFD0 {
			return true
		}
	}

	return false
}

// isCSVContent uses heuristics to determine if content is CSV
func isCSVContent(data []byte) bool {
	if len(data) == 0 {
		return false
	}

	// Sample first part of file (up to 4KB for efficiency)
	sampleSize := min(len(data), 4096)
	sample := data[:sampleSize]

	// Check if sample is valid UTF-8 or ASCII
	if !utf8.Valid(sample) {
		// Could still be other encodings, but less likely for CSV
		return false
	}

	// Count potential delimiters
	commaCount := bytes.Count(sample, []byte{','})
	semicolonCount := bytes.Count(sample, []byte{';'})
	tabCount := bytes.Count(sample, []byte{'\t'})
	pipeCount := bytes.Count(sample, []byte{'|'})
	totalDelimiters := commaCount + semicolonCount + tabCount + pipeCount

	// Need delimiters and line breaks
	newlineCount := bytes.Count(sample, []byte{'\n'})
	if totalDelimiters == 0 || newlineCount == 0 {
		return false
	}

	// Check for consistent delimiter pattern
	lines := bytes.Split(sample, []byte{'\n'})
	validLines := 0
	var prevDelimiterCount int = -1

	for i := 0; i < min(5, len(lines)); i++ {
		if len(bytes.TrimSpace(lines[i])) == 0 {
			continue // Skip empty lines
		}

		lineDelimiters := bytes.Count(lines[i], []byte{','}) +
			bytes.Count(lines[i], []byte{';'}) +
			bytes.Count(lines[i], []byte{'\t'}) +
			bytes.Count(lines[i], []byte{'|'})

		if prevDelimiterCount == -1 {
			prevDelimiterCount = lineDelimiters
		} else if lineDelimiters != prevDelimiterCount {
			return false // Inconsistent delimiter count
		}

		validLines++
	}

	return validLines >= 2 // Need at least 2 non-empty lines with consistent delimiters
}

// Helper function
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func normalizeHeader(header string) string {
	header = strings.ToLower(strings.TrimSpace(header))
	header = strings.ReplaceAll(header, " ", "_")
	header = strings.ReplaceAll(header, "-", "_")
	return header
}

func IsValidDate(dateStr string) bool {
	_, err := time.Parse("2006-01-02", dateStr)
	return err == nil
}

func IsValidTime(timeStr string) bool {
	_, err := time.Parse("15:04:05", timeStr)
	if err != nil {
		_, err = time.Parse("15:04", timeStr)
	}
	return err == nil
}

func IsValidCurrencyCode(code string) bool {
	if len(code) != 3 {
		return false
	}
	for _, r := range code {
		if r < 'A' || r > 'Z' {
			return false
		}
	}
	return true
}

func stringPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func calculateWorkers(totalRows int) int {
	switch {
	case totalRows <= 100:
		return 1
	case totalRows <= 500:
		return 2
	case totalRows <= 2000:
		return 4
	case totalRows <= 5000:
		return 8
	default:
		return 16
	}
}
