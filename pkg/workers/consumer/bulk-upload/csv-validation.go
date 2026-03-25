package bulkupload

import (
	"bytes"
	"context"
	"einvoice-access-point/internal/dtos"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-playground/validator/v10"
)

// CSVProcessor handles CSV file parsing and processing
type CSVProcessor struct {
	validator *validator.Validate
}

// NewCSVProcessor creates a new CSV processor
func NewCSVProcessor(validator *validator.Validate) *CSVProcessor {
	return &CSVProcessor{
		validator: validator,
	}
}

// ProcessCSV processes CSV data with automatic strategy selection
func (cp *CSVProcessor) ProcessCSV(data []byte, businessID string) ([]dtos.UploadInvoiceRequestDto, *ProcessingStats, []error) {
	stats := &ProcessingStats{
		StartTime: time.Now(),
	}

	// Read CSV and count rows
	totalRows, err := cp.countCSVRows(data)
	if err != nil {
		stats.EndTime = time.Now()
		stats.Duration = stats.EndTime.Sub(stats.StartTime)
		return nil, stats, []error{fmt.Errorf("failed to read CSV: %w", err)}
	}

	stats.TotalRows = totalRows

	// Choose processing strategy based on row count
	var invoices []dtos.UploadInvoiceRequestDto
	var errors []error

	if totalRows == 0 {
		stats.EndTime = time.Now()
		stats.Duration = stats.EndTime.Sub(stats.StartTime)
		return nil, stats, []error{fmt.Errorf("no data found in CSV file")}
	} else if totalRows <= 100 {
		// Small files: sequential processing
		invoices, errors = cp.processSequentially(data, businessID)
	} else if totalRows <= 5000 {
		// Medium files: concurrent processing
		workers := calculateWorkers(totalRows)
		log.Printf("Processing %d rows with %d concurrent workers", totalRows, workers)
		invoices, errors = cp.processConcurrently(data, businessID, workers)
	} else {
		// Large files: chunked processing
		chunkSize := 1000
		workersPerChunk := 8
		log.Printf("Processing %d rows with chunked strategy (chunk size: %d, workers per chunk: %d)",
			totalRows, chunkSize, workersPerChunk)

		ctx := context.Background()
		invoices, errors = cp.processChunked(ctx, data, businessID, chunkSize, workersPerChunk)
	}

	// Update statistics
	stats.EndTime = time.Now()
	stats.Duration = stats.EndTime.Sub(stats.StartTime)
	stats.ValidRows = len(invoices)
	stats.InvalidRows = stats.TotalRows - stats.ValidRows
	stats.TotalErrors = len(errors)

	log.Printf("Processing complete: %d valid invoices, %d errors, duration: %v",
		stats.ValidRows, stats.TotalErrors, stats.Duration)

	return invoices, stats, errors
}

// countCSVRows counts rows without loading entire file
func (cp *CSVProcessor) countCSVRows(data []byte) (int, error) {
	reader := csv.NewReader(bytes.NewReader(data))
	reader.FieldsPerRecord = -1

	rows := 0
	for {
		_, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return 0, err
		}
		rows++
	}

	// Subtract 1 for header row
	if rows > 0 {
		return rows - 1, nil
	}
	return 0, nil
}

// processSequentially processes CSV data sequentially
func (cp *CSVProcessor) processSequentially(data []byte, businessID string) ([]dtos.UploadInvoiceRequestDto, []error) {
	// Read all records
	records, err := cp.readCSVRecords(data)
	if err != nil {
		return nil, []error{err}
	}

	if len(records) < 2 {
		return nil, []error{fmt.Errorf("no data rows found")}
	}

	headers := records[0]
	rows := records[1:]
	headerIndex := cp.createHeaderIndex(headers)

	// Validate required headers
	if err := cp.validateRequiredHeaders(headerIndex); err != nil {
		return nil, []error{err}
	}

	var invoices []dtos.UploadInvoiceRequestDto
	var allErrors []error

	// Process rows sequentially
	for i, row := range rows {
		rowNum := i + 2 // 1-based + header

		if cp.isEmptyRow(row) {
			continue
		}

		invoice, err := cp.parseAndValidateRow(headerIndex, row, rowNum, businessID)
		if err != nil {
			allErrors = append(allErrors, err)
			continue
		}

		invoices = append(invoices, invoice)
	}

	return invoices, allErrors
}

// processConcurrently processes CSV data concurrently
func (cp *CSVProcessor) processConcurrently(data []byte, businessID string, workers int) ([]dtos.UploadInvoiceRequestDto, []error) {
	// Read all records
	records, err := cp.readCSVRecords(data)
	if err != nil {
		return nil, []error{err}
	}

	if len(records) < 2 {
		return nil, []error{fmt.Errorf("no data rows found")}
	}

	headers := records[0]
	rows := records[1:]
	headerIndex := cp.createHeaderIndex(headers)

	// Validate required headers
	if err := cp.validateRequiredHeaders(headerIndex); err != nil {
		return nil, []error{err}
	}

	type job struct {
		row    []string
		rowNum int
		index  int
	}

	type result struct {
		invoice dtos.UploadInvoiceRequestDto
		err     error
		index   int
	}

	jobs := make(chan job, len(rows))
	results := make(chan result, len(rows))

	// Start worker pool
	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for job := range jobs {
				if cp.isEmptyRow(job.row) {
					results <- result{index: job.index}
					continue
				}

				invoice, err := cp.parseAndValidateRow(headerIndex, job.row, job.rowNum, businessID)
				results <- result{
					invoice: invoice,
					err:     err,
					index:   job.index,
				}
			}
		}(w)
	}

	// Send jobs
	go func() {
		for i, row := range rows {
			jobs <- job{
				row:    row,
				rowNum: i + 2,
				index:  i,
			}
		}
		close(jobs)
	}()

	// Collect results
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results
	var invoices []dtos.UploadInvoiceRequestDto
	var errors []error
	invoiceMap := make(map[int]dtos.UploadInvoiceRequestDto, len(rows))

	for result := range results {
		if result.err != nil {
			errors = append(errors, result.err)
		} else if result.invoice.InvoiceNumber != "" { // Check if invoice was created
			invoiceMap[result.index] = result.invoice
		}
	}

	// Reconstruct in correct order
	for i := 0; i < len(rows); i++ {
		if invoice, ok := invoiceMap[i]; ok {
			invoices = append(invoices, invoice)
		}
	}

	return invoices, errors
}

// processChunked processes CSV data in chunks for large files
func (cp *CSVProcessor) processChunked(ctx context.Context, data []byte, businessID string, chunkSize int, workersPerChunk int) ([]dtos.UploadInvoiceRequestDto, []error) {
	// Read header first
	reader := csv.NewReader(bytes.NewReader(data))
	reader.FieldsPerRecord = -1

	header, err := reader.Read()
	if err != nil {
		return nil, []error{fmt.Errorf("failed to read CSV header: %w", err)}
	}

	headerIndex := cp.createHeaderIndex(header)

	// Validate required headers
	if err := cp.validateRequiredHeaders(headerIndex); err != nil {
		return nil, []error{err}
	}

	var allInvoices []dtos.UploadInvoiceRequestDto
	var allErrors []error
	chunkNum := 0

	for {
		// Read chunk of rows
		var chunk [][]string
		for i := 0; i < chunkSize; i++ {
			row, err := reader.Read()
			if err == io.EOF {
				break
			}
			if err != nil {
				return nil, append(allErrors, fmt.Errorf("failed to read CSV row: %w", err))
			}
			chunk = append(chunk, row)
		}

		if len(chunk) == 0 {
			break // No more rows
		}

		chunkNum++
		log.Printf("Processing chunk %d with %d rows", chunkNum, len(chunk))

		// Process this chunk concurrently
		chunkInvoices, chunkErrors := cp.processChunkConcurrently(chunk, headerIndex, businessID, workersPerChunk, chunkNum, chunkSize)

		// Collect results
		allInvoices = append(allInvoices, chunkInvoices...)
		allErrors = append(allErrors, chunkErrors...)

		// Check for context cancellation
		select {
		case <-ctx.Done():
			return allInvoices, append(allErrors, fmt.Errorf("processing cancelled: %w", ctx.Err()))
		default:
			// Continue
		}
	}

	return allInvoices, allErrors
}

// processChunkConcurrently processes a single chunk
func (cp *CSVProcessor) processChunkConcurrently(chunk [][]string, headerIndex map[string]int, businessID string, workers int, chunkNum int, chunkSize int) ([]dtos.UploadInvoiceRequestDto, []error) {
	type job struct {
		row    []string
		rowNum int
		index  int
	}

	type result struct {
		invoice dtos.UploadInvoiceRequestDto
		err     error
	}

	jobs := make(chan job, len(chunk))
	results := make(chan result, len(chunk))

	var wg sync.WaitGroup

	// Start workers
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for job := range jobs {
				if cp.isEmptyRow(job.row) {
					results <- result{}
					continue
				}

				invoice, err := cp.parseAndValidateRow(headerIndex, job.row, job.rowNum, businessID)
				results <- result{
					invoice: invoice,
					err:     err,
				}
			}
		}(w)
	}

	// Send jobs
	go func() {
		for i, row := range chunk {
			// Calculate global row number
			globalRowNum := (chunkNum-1)*chunkSize + i + 2 // +2 for header and 1-based indexing

			jobs <- job{
				row:    row,
				rowNum: globalRowNum,
				index:  i,
			}
		}
		close(jobs)
	}()

	// Wait and collect
	go func() {
		wg.Wait()
		close(results)
	}()

	var chunkInvoices []dtos.UploadInvoiceRequestDto
	var chunkErrors []error

	for result := range results {
		if result.err != nil {
			chunkErrors = append(chunkErrors, result.err)
		} else if result.invoice.InvoiceNumber != "" {
			chunkInvoices = append(chunkInvoices, result.invoice)
		}
	}

	return chunkInvoices, chunkErrors
}

// Helper methods

// readCSVRecords reads all CSV records
func (cp *CSVProcessor) readCSVRecords(data []byte) ([][]string, error) {
	reader := csv.NewReader(bytes.NewReader(data))
	reader.FieldsPerRecord = -1
	reader.LazyQuotes = true
	reader.TrimLeadingSpace = true

	return reader.ReadAll()
}

// createHeaderIndex creates a normalized header index
func (cp *CSVProcessor) createHeaderIndex(headers []string) map[string]int {
	index := make(map[string]int)
	for i, header := range headers {
		normalized := normalizeHeader(header)
		index[normalized] = i
	}
	return index
}

// validateRequiredHeaders checks for required columns
func (cp *CSVProcessor) validateRequiredHeaders(headerIndex map[string]int) error {
	requiredHeaders := []string{
		"invoice_number",
		"issue_date",
		"invoice_type_code",
		"document_currency_code",
		"tax_currency_code",
		"tax_total",
		"legal_monetary_total.line_extension_amount",
		"legal_monetary_total.tax_exclusive_amount",
		"legal_monetary_total.tax_inclusive_amount",
		"legal_monetary_total.payable_amount",
		"invoice_line",
		"supplier_party.party_name",
		"supplier_party.tin",
		"supplier_party.email",
		"supplier_party.street_name",
		"supplier_party.city_name",
		"supplier_party.postal_zone",
		"supplier_party.lga",
		"supplier_party.state",
		"supplier_party.country",
	}

	missingHeaders := make([]string, 0)
	for _, reqHeader := range requiredHeaders {
		if _, exists := headerIndex[reqHeader]; !exists {
			missingHeaders = append(missingHeaders, reqHeader)
		}
	}

	if len(missingHeaders) > 0 {
		return fmt.Errorf("missing required columns: %v", missingHeaders)
	}

	return nil
}

// parseAndValidateRow parses a single row and validates it
func (cp *CSVProcessor) parseAndValidateRow(headerIndex map[string]int, row []string, rowNumber int, businessID string) (dtos.UploadInvoiceRequestDto, error) {
	invoice, parseErrors := cp.parseCSVRow(headerIndex, row, rowNumber, businessID)
	if len(parseErrors) > 0 {
		// Combine all parse errors into one
		var errorStrs []string
		for _, err := range parseErrors {
			errorStrs = append(errorStrs, err.Error())
		}
		return invoice, fmt.Errorf("parse errors: %s", strings.Join(errorStrs, "; "))
	}

	// Validate the struct
	if err := cp.validator.Struct(invoice); err != nil {
		return invoice, fmt.Errorf("validation failed: %w", err)
	}

	return invoice, nil
}

// parseCSVRow parses a single CSV row
func (cp *CSVProcessor) parseCSVRow(headerIndex map[string]int, row []string, rowNumber int, businessID string) (dtos.UploadInvoiceRequestDto, []error) {
	invoice := dtos.UploadInvoiceRequestDto{
		BusinessID:  businessID,
		TaxTotal:    make([]dtos.TaxTotal, 0),
		InvoiceLine: make([]dtos.InvoiceLine, 0),
	}

	var errors []error

	// Helper functions
	getValue := func(fieldName string) (string, bool) {
		if idx, ok := headerIndex[fieldName]; ok && idx < len(row) {
			val := cp.cleanCSVValue(row[idx])
			return val, true
		}
		return "", false
	}

	addError := func(field string, err error) {
		errors = append(errors, fmt.Errorf("row %d, field '%s': %w", rowNumber, field, err))
	}

	// Parse required fields
	for fieldName, required := range cp.getFieldDefinitions() {
		val, exists := getValue(fieldName)

		if required && (!exists || val == "") {
			addError(fieldName, fmt.Errorf("required field not found or empty"))
			continue
		}

		if exists && val != "" {
			if err := cp.parseField(fieldName, val, &invoice); err != nil {
				addError(fieldName, err)
			}
		}
	}

	return invoice, errors
}

// getFieldDefinitions returns field definitions (field name -> required)
func (cp *CSVProcessor) getFieldDefinitions() map[string]bool {
	return map[string]bool{
		"invoice_number":                true,
		"issue_date":                    true,
		"invoice_type_code":             true,
		"document_currency_code":        true,
		"tax_currency_code":             true,
		"accounting_supplier_party":     true,
		"tax_total":                     true,
		"legal_monetary_total":          true,
		"invoice_line":                  true,
		"payment_status":                false,
		"irn":                           false,
		"due_date":                      false,
		"issue_time":                    false,
		"note":                          false,
		"tax_point_date":                false,
		"accounting_cost":               false,
		"buyer_reference":               false,
		"order_reference":               false,
		"actual_delivery_date":          false,
		"payment_terms_note":            false,
		"accounting_customer_party":     false,
		"payee_party":                   false,
		"tax_representative_party":      false,
		"invoice_delivery_period":       false,
		"billing_reference":             false,
		"dispatch_document_reference":   false,
		"receipt_document_reference":    false,
		"originator_document_reference": false,
		"contract_document_reference":   false,
		"additional_document_reference": false,
		"payment_means":                 false,
		"allowance_charge":              false,
	}
}

// parseField parses a single field value
func (cp *CSVProcessor) parseField(fieldName, value string, invoice *dtos.UploadInvoiceRequestDto) error {
	switch fieldName {
	// String fields
	case "invoice_number":
		invoice.InvoiceNumber = value
	case "issue_date":
		if !IsValidDate(value) {
			return fmt.Errorf("invalid date format, expected YYYY-MM-DD")
		}
		invoice.IssueDate = value
	case "invoice_type_code":
		invoice.InvoiceTypeCode = value
	case "document_currency_code":
		if !IsValidCurrencyCode(value) {
			return fmt.Errorf("currency code '%s' is invalid", value)
		}
		invoice.DocumentCurrencyCode = value
	case "tax_currency_code":
		if !IsValidCurrencyCode(value) {
			return fmt.Errorf("currency code '%s' is invalid", value)
		}
		invoice.TaxCurrencyCode = value

	// Optional string pointer fields
	case "payment_status":
		invoice.PaymentStatus = stringPtr(value)
	case "irn":
		invoice.IRN = stringPtr(value)
	case "due_date":
		if value != "" && !IsValidDate(value) {
			return fmt.Errorf("invalid date format, expected YYYY-MM-DD")
		}
		invoice.DueDate = stringPtr(value)
	case "issue_time":
		if value != "" && !IsValidTime(value) {
			return fmt.Errorf("invalid time format, expected HH:MM:SS")
		}
		invoice.IssueTime = stringPtr(value)
	case "note":
		invoice.Note = stringPtr(value)
	case "tax_point_date":
		if value != "" && !IsValidDate(value) {
			return fmt.Errorf("invalid date format, expected YYYY-MM-DD")
		}
		invoice.TaxPointDate = stringPtr(value)
	case "accounting_cost":
		invoice.AccountingCost = stringPtr(value)
	case "buyer_reference":
		invoice.BuyerReference = stringPtr(value)
	case "order_reference":
		invoice.OrderReference = stringPtr(value)
	case "actual_delivery_date":
		if value != "" && !IsValidDate(value) {
			return fmt.Errorf("invalid date format, expected YYYY-MM-DD")
		}
		invoice.ActualDeliveryDate = stringPtr(value)
	case "payment_terms_note":
		invoice.PaymentTermsNote = stringPtr(value)

	// Accounting supplier Json flattened
	case "supplier_party.party_name":
		invoice.AccountingSupplierParty.PartyName = value
	case "supplier_party.tin":
		invoice.AccountingSupplierParty.TIN = value
	case "supplier_party.email":
		invoice.AccountingSupplierParty.Email = value
	case "supplier_party.telephone":
		invoice.AccountingSupplierParty.Telephone = stringPtr(value)
	case "supplier_party.business_description":
		invoice.AccountingSupplierParty.BusinessDescription = stringPtr(value)
	case "supplier_party.street_name":
		invoice.AccountingSupplierParty.PostalAddress.StreetName = value
	case "supplier_party.city_name":
		invoice.AccountingSupplierParty.PostalAddress.CityName = value
	case "supplier_party.postal_zone":
		invoice.AccountingSupplierParty.PostalAddress.PostalZone = value
	case "supplier_party.lga":
		invoice.AccountingSupplierParty.PostalAddress.LGA = value
	case "supplier_party.state":
		invoice.AccountingSupplierParty.PostalAddress.State = value
	case "supplier_party.country":
		invoice.AccountingSupplierParty.PostalAddress.Country = value

	// legal_monetary_total json flattened
	case "legal_monetary_total.line_extension_amount":
		floatValue, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return fmt.Errorf("legal_monetary_total.line_extension_amount: invalid numeric value, expected a valid number")
		}
		invoice.LegalMonetaryTotal.LineExtensionAmount = floatValue
	case "legal_monetary_total.tax_exclusive_amount":
		floatValue, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return fmt.Errorf("legal_monetary_total.tax_exclusive_amount: invalid numeric value, expected a valid number")
		}
		invoice.LegalMonetaryTotal.TaxExclusiveAmount = floatValue
	case "legal_monetary_total.tax_inclusive_amount":
		floatValue, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return fmt.Errorf("legal_monetary_total.tax_inclusive_amount: invalid numeric value, expected a valid number")
		}
		invoice.LegalMonetaryTotal.TaxInclusiveAmount = floatValue
	case "legal_monetary_total.payable_amount":
		floatValue, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return fmt.Errorf("legal_monetary_total.payable_amount: invalid numeric value, expected a valid number")
		}
		invoice.LegalMonetaryTotal.PayableAmount = floatValue

	// JSON fields
	case "tax_total":
		var taxTotals []dtos.TaxTotal
		if err := json.Unmarshal([]byte(value), &taxTotals); err != nil {
			return fmt.Errorf("failed to parse JSON: %w", err)
		}
		invoice.TaxTotal = taxTotals

	case "invoice_line":
		var invoiceLines []dtos.InvoiceLine
		if err := json.Unmarshal([]byte(value), &invoiceLines); err != nil {
			return fmt.Errorf("failed to parse JSON: %w", err)
		}
		invoice.InvoiceLine = invoiceLines

	// Optional JSON pointer fields
	case "accounting_customer_party":
		var party dtos.Party
		if err := json.Unmarshal([]byte(value), &party); err != nil {
			return fmt.Errorf("failed to parse JSON: %w", err)
		}
		invoice.AccountingCustomerParty = &party

	case "payee_party":
		var party dtos.Party
		if err := json.Unmarshal([]byte(value), &party); err != nil {
			return fmt.Errorf("failed to parse JSON: %w", err)
		}
		invoice.PayeeParty = &party

	case "tax_representative_party":
		var party dtos.Party
		if err := json.Unmarshal([]byte(value), &party); err != nil {
			return fmt.Errorf("failed to parse JSON: %w", err)
		}
		invoice.TaxRepresentativeParty = &party

	case "invoice_delivery_period":
		var period dtos.InvoiceDeliveryPeriod
		if err := json.Unmarshal([]byte(value), &period); err != nil {
			return fmt.Errorf("failed to parse JSON: %w", err)
		}
		invoice.InvoiceDeliveryPeriod = &period

	// Optional JSON array fields
	case "billing_reference":
		var refs []dtos.DocumentReference
		if err := json.Unmarshal([]byte(value), &refs); err != nil {
			return fmt.Errorf("failed to parse JSON: %w", err)
		}
		invoice.BillingReference = refs

	case "dispatch_document_reference":
		var ref dtos.DocumentReference
		if err := json.Unmarshal([]byte(value), &ref); err != nil {
			return fmt.Errorf("failed to parse JSON: %w", err)
		}
		invoice.DispatchDocumentReference = &ref

	case "receipt_document_reference":
		var ref dtos.DocumentReference
		if err := json.Unmarshal([]byte(value), &ref); err != nil {
			return fmt.Errorf("failed to parse JSON: %w", err)
		}
		invoice.ReceiptDocumentReference = &ref

	case "originator_document_reference":
		var ref dtos.DocumentReference
		if err := json.Unmarshal([]byte(value), &ref); err != nil {
			return fmt.Errorf("failed to parse JSON: %w", err)
		}
		invoice.OriginatorDocumentReference = &ref

	case "contract_document_reference":
		var ref dtos.DocumentReference
		if err := json.Unmarshal([]byte(value), &ref); err != nil {
			return fmt.Errorf("failed to parse JSON: %w", err)
		}
		invoice.ContractDocumentReference = &ref

	case "additional_document_reference":
		var refs []dtos.DocumentReference
		if err := json.Unmarshal([]byte(value), &refs); err != nil {
			return fmt.Errorf("failed to parse JSON: %w", err)
		}
		invoice.AdditionalDocumentReference = refs

	case "payment_means":
		var means []dtos.PaymentMeans
		if err := json.Unmarshal([]byte(value), &means); err != nil {
			return fmt.Errorf("failed to parse JSON: %w", err)
		}
		invoice.PaymentMeans = means

	case "allowance_charge":
		var charges []dtos.AllowanceCharge
		if err := json.Unmarshal([]byte(value), &charges); err != nil {
			return fmt.Errorf("failed to parse JSON: %w", err)
		}
		invoice.AllowanceCharge = charges
	}

	return nil
}

// cleanCSVValue cleans up CSV cell values
func (cp *CSVProcessor) cleanCSVValue(value string) string {
	value = strings.TrimSpace(value)
	value = strings.Trim(value, `"`)
	value = strings.Trim(value, `'`)
	value = strings.ReplaceAll(value, `\"`, `"`)
	value = strings.ReplaceAll(value, `""`, `"`)
	return value
}

// isEmptyRow checks if a row is empty
func (cp *CSVProcessor) isEmptyRow(row []string) bool {
	for _, cell := range row {
		if strings.TrimSpace(cell) != "" {
			return false
		}
	}
	return true
}
