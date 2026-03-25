package bulkupload

import (
	"bytes"
	"context"
	"einvoice-access-point/internal/dtos"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/xuri/excelize/v2"
)

// ExcelProcessor handles Excel file parsing and processing
type ExcelProcessor struct {
	validator *validator.Validate
}

// NewExcelProcessor creates a new Excel processor
func NewExcelProcessor(validator *validator.Validate) *ExcelProcessor {
	return &ExcelProcessor{
		validator: validator,
	}
}

// ProcessExcel processes Excel data with automatic strategy selection
func (ep *ExcelProcessor) ProcessExcel(data []byte, businessID string) ([]dtos.UploadInvoiceRequestDto, *ProcessingStats, []error) {
	stats := &ProcessingStats{
		StartTime: time.Now(),
	}

	// Open Excel file
	f, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		stats.EndTime = time.Now()
		stats.Duration = stats.EndTime.Sub(stats.StartTime)
		return nil, stats, []error{fmt.Errorf("failed to open Excel file: %w", err)}
	}
	defer f.Close()

	// Read all rows
	sheetName := f.GetSheetName(0)
	rows, err := f.GetRows(sheetName)
	if err != nil {
		stats.EndTime = time.Now()
		stats.Duration = stats.EndTime.Sub(stats.StartTime)
		return nil, stats, []error{fmt.Errorf("failed to read rows: %w", err)}
	}

	if len(rows) == 0 {
		stats.EndTime = time.Now()
		stats.Duration = stats.EndTime.Sub(stats.StartTime)
		return nil, stats, []error{fmt.Errorf("no data found in Excel file")}
	}

	// Process headers
	headers := rows[0]
	headerIndex := ep.createHeaderIndex(headers)

	// Validate required headers
	if err := ep.validateRequiredHeaders(headerIndex); err != nil {
		stats.EndTime = time.Now()
		stats.Duration = stats.EndTime.Sub(stats.StartTime)
		return nil, stats, []error{err}
	}

	dataRows := rows[1:]
	stats.TotalRows = len(dataRows)

	// Choose processing strategy based on row count
	var invoices []dtos.UploadInvoiceRequestDto
	var errors []error

	if stats.TotalRows == 0 {
		stats.EndTime = time.Now()
		stats.Duration = stats.EndTime.Sub(stats.StartTime)
		return nil, stats, []error{fmt.Errorf("no data rows found")}
	} else if stats.TotalRows <= 100 {
		// Small files: sequential processing
		invoices, errors = ep.processSequentially(dataRows, headerIndex, businessID)
	} else if stats.TotalRows <= 5000 {
		// Medium files: concurrent processing
		workers := calculateWorkers(stats.TotalRows)
		log.Printf("Processing %d Excel rows with %d concurrent workers", stats.TotalRows, workers)
		invoices, errors = ep.processConcurrently(dataRows, headerIndex, businessID, workers)
	} else {
		// Large files: chunked processing (Excel files are loaded in memory anyway)
		workers := calculateWorkers(stats.TotalRows)
		log.Printf("Processing %d Excel rows with %d workers (chunked in memory)", stats.TotalRows, workers)
		invoices, errors = ep.processConcurrently(dataRows, headerIndex, businessID, workers)
	}

	// Update statistics
	stats.EndTime = time.Now()
	stats.Duration = stats.EndTime.Sub(stats.StartTime)
	stats.ValidRows = len(invoices)
	stats.InvalidRows = stats.TotalRows - stats.ValidRows
	stats.TotalErrors = len(errors)

	log.Printf("Excel processing complete: %d invoices, %d errors, duration: %v",
		stats.ValidRows, stats.TotalErrors, stats.Duration)
	return invoices, stats, errors
}

// createHeaderIndex creates a normalized header index
func (ep *ExcelProcessor) createHeaderIndex(headers []string) map[string]int {
	index := make(map[string]int)
	for i, header := range headers {
		normalized := normalizeHeader(header)
		index[normalized] = i
	}
	return index
}

// validateRequiredHeaders checks for required columns
func (ep *ExcelProcessor) validateRequiredHeaders(headerIndex map[string]int) error {
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

// processSequentially processes Excel data sequentially
func (ep *ExcelProcessor) processSequentially(rows [][]string, headerIndex map[string]int, businessID string) ([]dtos.UploadInvoiceRequestDto, []error) {
	var invoices []dtos.UploadInvoiceRequestDto
	var allErrors []error

	for i, row := range rows {
		rowNum := i + 2 // 1-based + header

		if ep.isEmptyRow(row) {
			continue
		}

		invoice, err := ep.parseAndValidateRow(headerIndex, row, rowNum, businessID)
		if err != nil {
			allErrors = append(allErrors, err)
			continue
		}

		invoices = append(invoices, invoice)
	}

	return invoices, allErrors
}

// processConcurrently processes Excel data concurrently
func (ep *ExcelProcessor) processConcurrently(rows [][]string, headerIndex map[string]int, businessID string, workers int) ([]dtos.UploadInvoiceRequestDto, []error) {
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
				if ep.isEmptyRow(job.row) {
					results <- result{index: job.index}
					continue
				}

				invoice, err := ep.parseAndValidateRow(headerIndex, job.row, job.rowNum, businessID)
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

// parseAndValidateRow parses a single row and validates it
func (ep *ExcelProcessor) parseAndValidateRow(headerIndex map[string]int, row []string, rowNumber int, businessID string) (dtos.UploadInvoiceRequestDto, error) {
	invoice, parseErrors := ep.parseExcelRow(headerIndex, row, rowNumber, businessID)
	if len(parseErrors) > 0 {
		// Combine all parse errors into one
		var errorStrs []string
		for _, err := range parseErrors {
			errorStrs = append(errorStrs, err.Error())
		}
		return invoice, fmt.Errorf("parse errors: %s", strings.Join(errorStrs, "; "))
	}

	// Validate the struct
	if err := ep.validator.Struct(invoice); err != nil {
		return invoice, fmt.Errorf("validation failed: %w", err)
	}

	return invoice, nil
}

// parseExcelRow parses a single Excel row
func (ep *ExcelProcessor) parseExcelRow(headerIndex map[string]int, row []string, rowNumber int, businessID string) (dtos.UploadInvoiceRequestDto, []error) {
	invoice := dtos.UploadInvoiceRequestDto{
		BusinessID:  businessID,
		TaxTotal:    make([]dtos.TaxTotal, 0),
		InvoiceLine: make([]dtos.InvoiceLine, 0),
	}

	var errors []error

	// Helper functions
	getValue := func(fieldName string) (string, bool) {
		if idx, ok := headerIndex[fieldName]; ok && idx < len(row) {
			val := strings.TrimSpace(row[idx])
			// Clean Excel escaping
			val = strings.ReplaceAll(val, `\"`, `"`)
			val = strings.ReplaceAll(val, `'`, `"`)
			val = strings.Trim(val, `"`)
			return val, true
		}
		return "", false
	}

	addError := func(field string, err error) {
		errors = append(errors, fmt.Errorf("row %d, field '%s': %w", rowNumber, field, err))
	}

	// Parse required fields
	for fieldName, required := range ep.getFieldDefinitions() {
		val, exists := getValue(fieldName)

		if required && (!exists || val == "") {
			addError(fieldName, fmt.Errorf("required field not found or empty"))
			continue
		}

		if exists && val != "" {
			if err := ep.parseField(fieldName, val, &invoice); err != nil {
				addError(fieldName, err)
			}
		}
	}

	return invoice, errors
}

// getFieldDefinitions returns field definitions (field name -> required)
func (ep *ExcelProcessor) getFieldDefinitions() map[string]bool {
	return map[string]bool{
		"invoice_number":         true,
		"issue_date":             true,
		"invoice_type_code":      true,
		"document_currency_code": true,
		"tax_currency_code":      true,
		"tax_total":              true,
		"legal_monetary_total.line_extension_amount": true,
		"legal_monetary_total.tax_exclusive_amount":  true,
		"legal_monetary_total.tax_inclusive_amount":  true,
		"legal_monetary_total.payable_amount":        true,
		"invoice_line":                               true,
		"supplier_party.party_name":                  true,
		"supplier_party.tin":                         true,
		"supplier_party.email":                       true,
		"supplier_party.street_name":                 true,
		"supplier_party.city_name":                   true,
		"supplier_party.postal_zone":                 true,
		"supplier_party.lga":                         true,
		"supplier_party.state":                       true,
		"supplier_party.country":                     true,
		"payment_status":                             false,
		"irn":                                        false,
		"due_date":                                   false,
		"issue_time":                                 false,
		"note":                                       false,
		"tax_point_date":                             false,
		"accounting_cost":                            false,
		"buyer_reference":                            false,
		"order_reference":                            false,
		"actual_delivery_date":                       false,
		"payment_terms_note":                         false,
		"payee_party":                                false,
		"tax_representative_party":                   false,
		"invoice_delivery_period":                    false,
		"billing_reference":                          false,
		"dispatch_document_reference":                false,
		"receipt_document_reference":                 false,
		"originator_document_reference":              false,
		"contract_document_reference":                false,
		"additional_document_reference":              false,
		"payment_means":                              false,
		"allowance_charge":                           false,
		"accounting_customer_party":                  false,
	}
}

// parseField parses a single field value
func (ep *ExcelProcessor) parseField(fieldName, value string, invoice *dtos.UploadInvoiceRequestDto) error {
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

// isEmptyRow checks if a row is empty
func (ep *ExcelProcessor) isEmptyRow(row []string) bool {
	for _, cell := range row {
		if strings.TrimSpace(cell) != "" {
			return false
		}
	}
	return true
}

// ProcessExcelWithRetry processes Excel with retry logic
func (ep *ExcelProcessor) ProcessExcelWithRetry(ctx context.Context, data []byte, maxRetries int, businessID string) ([]dtos.UploadInvoiceRequestDto, []error, error) {
	var allErrors []error

	for attempt := 1; attempt <= maxRetries; attempt++ {
		invoices, _, errors := ep.ProcessExcel(data, businessID)

		if len(errors) == 0 {
			return invoices, errors, nil
		}

		allErrors = append(allErrors, errors...)

		if attempt < maxRetries {
			log.Printf("Excel Attempt %d failed with %d errors, attempting to fix common issues",
				attempt, len(errors))

			fixedData, err := ep.fixCommonExcelIssues(data, errors)
			if err != nil {
				log.Printf("Failed to fix Excel issues: %v", err)
				break
			}

			data = fixedData
		}
	}

	// Try one last time to get any valid invoices
	invoices, _, _ := ep.ProcessExcel(data, businessID)

	if len(invoices) == 0 {
		return nil, allErrors, fmt.Errorf("failed to process Excel file after %d attempts", maxRetries)
	}

	return invoices, allErrors, nil
}

// fixCommonExcelIssues attempts to fix common Excel formatting issues
func (ep *ExcelProcessor) fixCommonExcelIssues(data []byte, errors []error) ([]byte, error) {
	f, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer f.Close()

	sheetName := f.GetSheetName(0)
	rows, err := f.GetRows(sheetName)
	if err != nil {
		return nil, err
	}

	// Fix headers
	for i, header := range rows[0] {
		rows[0][i] = strings.TrimSpace(header)
	}

	// Fix JSON fields
	for rowIdx := 1; rowIdx < len(rows); rowIdx++ {
		for colIdx := 0; colIdx < len(rows[rowIdx]); colIdx++ {
			cell := rows[rowIdx][colIdx]
			cell = strings.ReplaceAll(cell, `\"`, `"`)
			cell = strings.ReplaceAll(cell, `'`, `"`)
			cell = strings.Trim(cell, `"`)
			rows[rowIdx][colIdx] = cell
		}
	}

	// Create new workbook with fixes
	newFile := excelize.NewFile()
	newSheet := "Sheet1"
	newFile.NewSheet(newSheet)

	for rowIdx, row := range rows {
		for colIdx, cell := range row {
			cellRef, _ := excelize.CoordinatesToCellName(colIdx+1, rowIdx+1)
			newFile.SetCellValue(newSheet, cellRef, cell)
		}
	}

	// Save to buffer
	buf := new(bytes.Buffer)
	if err := newFile.Write(buf); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// GenerateExcelTemplate creates an Excel template for users
func GenerateExcelTemplate() (*bytes.Buffer, error) {
	f := excelize.NewFile()

	// Create template sheet
	f.NewSheet("Template")
	headers := []string{
		"invoice_number", "business_id", "issue_date", "invoice_type_code",
		"document_currency_code", "tax_currency_code", "payment_status",
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
		"supplier_party.country", "tax_total", "invoice_line",
		"irn", "due_date", "issue_time", "note", "tax_point_date", "accounting_cost",
		"buyer_reference", "order_reference", "actual_delivery_date", "payment_terms_note",
		"accounting_customer_party", "payee_party", "tax_representative_party",
		"invoice_delivery_period", "billing_reference", "dispatch_document_reference",
		"receipt_document_reference", "originator_document_reference",
		"contract_document_reference", "additional_document_reference",
		"payment_means", "allowance_charge",
	}

	for i, header := range headers {
		cellRef, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue("Template", cellRef, header)
	}

	// Add example data
	exampleRow := []string{
		"INV-001-2024",
		"123e4567-e89b-12d3-a456-426614174000",
		"2024-01-16",
		"381",
		"NGN",
		"NGN",
		"PENDING",
		`{"party_name":"ABC Ltd","tin":"123456789","email":"abc@test.com","telephone":"+2348012345678","postal_address":{"street_name":"123 Street","city_name":"City","postal_zone":"100001","lga":"NG-LA-IKE","state":"NG-LA","country":"NG"}}`,
		`[{"tax_amount":1500.75,"tax_subtotal":[{"taxable_amount":10000,"tax_amount":1500.75,"tax_category":{"id":"VAT","percent":15}}]}]`,
		`{"line_extension_amount":10000,"tax_exclusive_amount":10000,"tax_inclusive_amount":11500.75,"payable_amount":11500.75}`,
		`[{"hsn_code":"1282.10","product_category":"Electronics","invoiced_quantity":2,"line_extension_amount":20000,"item":{"name":"Laptop","description":"Gaming laptop"},"price":{"price_amount":10000,"base_quantity":1,"price_unit":"NGN per 1"}}]`,
		"",
		"2024-02-16",
		"10:00:00",
		"Invoice note",
		"",
		"",
		"",
		"",
		"",
		"",
		"",
		"",
		"",
		"",
		"",
		"",
		"",
		"",
		"",
		"",
		"",
		"",
		"",
	}

	for i, cell := range exampleRow {
		cellRef, _ := excelize.CoordinatesToCellName(i+1, 2)
		f.SetCellValue("Template", cellRef, cell)
	}

	// Set column widths
	for i := 1; i <= len(headers); i++ {
		width := 20.0
		if i >= 8 && i <= 11 { // JSON columns
			width = 40.0
		}
		colName, _ := excelize.ColumnNumberToName(i)
		f.SetColWidth("Template", colName, colName, width)
	}

	// Save to buffer
	buf := new(bytes.Buffer)
	if err := f.Write(buf); err != nil {
		return nil, err
	}

	return buf, nil
}

func IsDateAfterOrEqual(dateStr1, dateStr2 string) bool {
	date1, err1 := time.Parse("2006-01-02", dateStr1)
	date2, err2 := time.Parse("2006-01-02", dateStr2)

	if err1 != nil || err2 != nil {
		return false
	}

	return !date1.After(date2)
}
