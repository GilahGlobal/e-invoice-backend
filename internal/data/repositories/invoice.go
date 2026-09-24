package repositories

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"sort"
	"strconv"
	"time"

	"einvoice-access-point/internal/data/database"
	"einvoice-access-point/internal/data/entities"
	"einvoice-access-point/internal/utility"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type InvoiceRepository struct {
	db     database.DatabaseManager
	testDB database.DatabaseManager
}

type InvoiceFilter struct {
	IssueDate *string
	StartDate *string
	EndDate   *string
}

type InvoiceListWithMetadata struct {
	ID            string         `gorm:"column:id"`
	InvoiceNumber string         `gorm:"column:invoice_number"`
	IRN           string         `gorm:"column:irn"`
	Platform      string         `gorm:"column:platform"`
	CurrentStatus string         `gorm:"column:current_status"`
	PaymentStatus string         `gorm:"column:payment_status"`
	StatusText    string         `gorm:"column:status_text"`
	TotalAmount   float64        `gorm:"column:total_amount"`
	TaxAmount     float64        `gorm:"column:tax_amount"`
	IssueDate     *time.Time     `gorm:"column:issue_date"`
	StatusHistory datatypes.JSON `gorm:"column:status_history"`
	QrCodeBmpUrl  string         `gorm:"column:qr_code_bmp_url"`
	QrCode        string         `gorm:"column:qr_code"`
	CreatedAt     time.Time      `gorm:"column:created_at"`
}

func NewInvoiceRepository(db, testDB database.DatabaseManager) *InvoiceRepository {
	return &InvoiceRepository{
		db:     db,
		testDB: testDB,
	}
}

func (r *InvoiceRepository) GenerateUniqueInvoiceID(businessID string, db *gorm.DB) string {
	var lastInvoice entities.Invoice
	var newInvoiceNumber string

	err := db.Where("business_id = ?", businessID).
		Order("invoice_number DESC").
		First(&lastInvoice).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			newInvoiceNumber = "INV00001"
		} else {
			log.Println("Error fetching last invoice:", err)
			return ""
		}
	} else {
		lastNumber, _ := strconv.Atoi(lastInvoice.InvoiceNumber[3:])
		newInvoiceNumber = fmt.Sprintf("INV%05d", lastNumber+1)
	}

	return newInvoiceNumber
}

func (r *InvoiceRepository) CreateInvoice(db database.DatabaseManager, invoice *entities.Invoice) error {
	return db.DB().Create(invoice).Error
}

func (r *InvoiceRepository) FindInvoiceByNumber(db database.DatabaseManager, invoiceNumber string) (*entities.Invoice, error) {
	var invoice entities.Invoice
	err := db.DB().Where("invoice_number = ?", invoiceNumber).First(&invoice).Error
	if err != nil {
		return nil, err
	}
	return &invoice, nil
}

func (r *InvoiceRepository) FindInvoiceByNumberAndBusinessID(db database.DatabaseManager, invoiceNumber string, businessID string) (*entities.Invoice, error) {
	var invoice entities.Invoice
	err := db.DB().Where("invoice_number = ? AND business_id = ?", invoiceNumber, businessID).First(&invoice).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &invoice, err
}

func (r *InvoiceRepository) FindInvoiceByIRNAndBusinessID(db database.DatabaseManager, irn string, businessID string) (*entities.Invoice, error) {
	var invoice entities.Invoice
	err := db.DB().Where("irn = ? AND business_id = ?", irn, businessID).First(&invoice).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &invoice, err
}

func (r *InvoiceRepository) UpdateInvoiceStatus(db database.DatabaseManager, invoice *entities.Invoice, step string, status string, message ...string) error {
	var history []entities.StatusHistoryEntry

	if len(invoice.StatusHistory) > 0 {
		_ = json.Unmarshal(invoice.StatusHistory, &history)
	}

	entryMessage := entities.StatusHistoryMessage(step, status)
	if len(message) > 0 {
		entryMessage = message[0]
	}
	entryMessage = utility.ExtractRelevantErrorMessage(errors.New(entryMessage))

	for i := range history {
		if history[i].Step == step {
			history[i].Status = status
			history[i].Message = entryMessage
			history[i].Timestamp = time.Now()
			break
		}
	}

	found := false
	for _, entry := range history {
		if entry.Step == step {
			found = true
			break
		}
	}
	if !found {
		history = append(history, entities.StatusHistoryEntry{
			Step:      step,
			Status:    status,
			Message:   entryMessage,
			Timestamp: time.Now(),
		})
	}

	historyJSON, _ := json.Marshal(history)
	invoice.StatusHistory = historyJSON
	invoice.CurrentStatus = step

	return db.DB().Save(invoice).Error
}

func (r *InvoiceRepository) UpdateInvoiceIRN(db database.DatabaseManager, invoice *entities.Invoice, irn string) error {
	invoice.IRN = irn
	return db.DB().Save(invoice).Error
}

func (r *InvoiceRepository) FindMinimalInvoicesByBusinessID(
	db database.DatabaseManager,
	businessID string,
	pagination database.Pagination,
	filter ...InvoiceFilter,
) ([]entities.MinimalInvoiceDTO, database.PaginationResponse, error) {

	var result []entities.MinimalInvoiceDTO

	// Set pagination defaults
	if pagination.Page <= 0 {
		pagination.Page = 1
	}

	if pagination.Limit <= 0 {
		pagination.Limit = 20
	}

	countDB := db.DB().
		Model(&entities.Invoice{}).
		Where("business_id = ? AND deleted_at IS NULL", businessID)

	whereClause := "invoices.business_id = ? AND invoices.deleted_at IS NULL"
	args := []interface{}{businessID}

	if len(filter) > 0 {
		f := filter[0]
		if f.IssueDate != nil && *f.IssueDate != "" {
			countDB = countDB.Where("issue_date = ?", *f.IssueDate)
			whereClause += " AND invoices.issue_date = ?"
			args = append(args, *f.IssueDate)
		}
		if f.StartDate != nil && *f.StartDate != "" {
			countDB = countDB.Where("issue_date >= ?", *f.StartDate)
			whereClause += " AND invoices.issue_date >= ?"
			args = append(args, *f.StartDate)
		}
		if f.EndDate != nil && *f.EndDate != "" {
			countDB = countDB.Where("issue_date <= ?", *f.EndDate)
			whereClause += " AND invoices.issue_date <= ?"
			args = append(args, *f.EndDate)
		}
	}

	// Get total number of invoices
	var totalCount int64
	if err := countDB.Count(&totalCount).Error; err != nil {
		return nil, database.PaginationResponse{
			CurrentPage:     pagination.Page,
			PageCount:       0,
			TotalPagesCount: 0,
		}, err
	}

	// Calculate total pages
	totalPages := int(math.Ceil(
		float64(totalCount) / float64(pagination.Limit),
	))

	// Calculate offset
	offset := (pagination.Page - 1) * pagination.Limit

	query := fmt.Sprintf(`
	SELECT 
		invoices.id,
		invoices.invoice_number,
		invoices.irn,
		invoices.platform,
		invoices.current_status,
		invoices.payment_status,
		invoices.status_history,
		invoices.total_amount,
		invoices.tax_amount,
		invoices.issue_date,
		invoices.qr_code_bmp_url,
		invoices.qr_code,

		CASE

			WHEN invoices.current_status = 'transmitted_invoice' THEN
				'partial_success'
			
			WHEN invoices.current_status = 'confirmed_invoice' THEN
				CASE
					WHEN current_step_status = 'success' THEN 'success'
					ELSE 'partial_success'
				END

			WHEN invoices.current_status = 'signed_invoice' THEN
				CASE
					WHEN current_step_status = 'success' THEN 'partial_success'
					ELSE 'failed'
				END

			ELSE
				'failed'
		END AS status_text,

		invoices.created_at

	FROM invoices

	LEFT JOIN LATERAL (
		SELECT entry->>'status' AS current_step_status
		FROM jsonb_array_elements(invoices.status_history) AS entry
		WHERE entry->>'step' = invoices.current_status
		ORDER BY (entry->>'timestamp')::timestamptz DESC
		LIMIT 1
	) AS current_step ON TRUE

	WHERE %s

	ORDER BY invoices.created_at DESC
	LIMIT ? OFFSET ?;
	`, whereClause)

	args = append(args, pagination.Limit, offset)

	if err := db.DB().
		Raw(query, args...).
		Scan(&result).Error; err != nil {

		return nil, database.PaginationResponse{
			CurrentPage:     pagination.Page,
			PageCount:       0,
			TotalPagesCount: totalPages,
		}, err
	}

	return result, database.PaginationResponse{
		CurrentPage:     pagination.Page,
		PageCount:       len(result),
		TotalPagesCount: totalPages,
	}, nil
}

func (r *InvoiceRepository) FindInvoicesWithMetadataByBusinessID(
	db database.DatabaseManager,
	businessID string,
	pagination database.Pagination,
	filter ...InvoiceFilter,
) ([]InvoiceListWithMetadata, database.PaginationResponse, error) {

	var result []InvoiceListWithMetadata

	// Set pagination defaults
	if pagination.Page <= 0 {
		pagination.Page = 1
	}

	if pagination.Limit <= 0 {
		pagination.Limit = 20
	}

	countDB := db.DB().
		Model(&entities.Invoice{}).
		Where("business_id = ? AND deleted_at IS NULL", businessID)

	whereClause := "business_id = ? AND deleted_at IS NULL"
	args := []interface{}{businessID}

	if len(filter) > 0 {
		f := filter[0]
		if f.IssueDate != nil && *f.IssueDate != "" {
			countDB = countDB.Where("issue_date = ?", *f.IssueDate)
			whereClause += " AND issue_date = ?"
			args = append(args, *f.IssueDate)
		}
		if f.StartDate != nil && *f.StartDate != "" {
			countDB = countDB.Where("issue_date >= ?", *f.StartDate)
			whereClause += " AND issue_date >= ?"
			args = append(args, *f.StartDate)
		}
		if f.EndDate != nil && *f.EndDate != "" {
			countDB = countDB.Where("issue_date <= ?", *f.EndDate)
			whereClause += " AND issue_date <= ?"
			args = append(args, *f.EndDate)
		}
	}

	// Get total number of invoices
	var totalCount int64
	if err := countDB.Count(&totalCount).Error; err != nil {
		return nil, database.PaginationResponse{
			CurrentPage:     pagination.Page,
			PageCount:       0,
			TotalPagesCount: 0,
		}, err
	}

	// Calculate total pages
	totalPages := int(math.Ceil(
		float64(totalCount) / float64(pagination.Limit),
	))

	// Calculate offset
	offset := (pagination.Page - 1) * pagination.Limit

	query := fmt.Sprintf(`
	SELECT 
		id,
		invoice_number,
		irn,
		platform,
		current_status,
		payment_status,
		status_history,
		total_amount,
		tax_amount,
		issue_date,
		qr_code_bmp_url,
		qr_code,

		CASE

			WHEN current_status = 'transmitted_invoice' THEN
				'partial_success'
			
			WHEN current_status = 'confirmed_invoice' THEN
				CASE
					WHEN current_step_status = 'success' THEN 'success'
					ELSE 'partial_success'
				END

			WHEN current_status = 'signed_invoice' THEN
				CASE
					WHEN current_step_status = 'success' THEN 'partial_success'
					ELSE 'failed'
				END

			ELSE
				'failed'
		END AS status_text,

		created_at

	FROM (
		SELECT 
			invoices.*,

			(
				SELECT entry->>'status'
				FROM jsonb_array_elements(invoices.status_history) AS entry
				WHERE entry->>'step' = invoices.current_status
				ORDER BY (entry->>'timestamp')::timestamptz DESC
				LIMIT 1
			) AS current_step_status

		FROM invoices

		WHERE %s
	) AS invoices

	ORDER BY created_at DESC
	LIMIT ? OFFSET ?;
	`, whereClause)

	args = append(args, pagination.Limit, offset)

	if err := db.DB().
		Raw(query, args...).
		Scan(&result).Error; err != nil {

		return nil, database.PaginationResponse{
			CurrentPage:     pagination.Page,
			PageCount:       0,
			TotalPagesCount: totalPages,
		}, err
	}

	return result, database.PaginationResponse{
		CurrentPage:     pagination.Page,
		PageCount:       len(result),
		TotalPagesCount: totalPages,
	}, nil
}

func (r *InvoiceRepository) FindInvoiceByBusinessAndID(db database.DatabaseManager, businessID, invoiceID string) (*entities.Invoice, error) {
	var invoice entities.Invoice
	if err := db.DB().
		Where("business_id = ? AND id = ?", businessID, invoiceID).
		First(&invoice).Error; err != nil {
		return nil, err
	}
	return &invoice, nil
}

func (r *InvoiceRepository) DeleteInvoiceByBusinessAndID(db database.DatabaseManager, businessID, invoiceID string) error {
	result := db.DB().
		Where("business_id = ? AND id = ?", businessID, invoiceID).
		Delete(&entities.Invoice{})

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("invoice not found")
	}

	return nil
}

func (r *InvoiceRepository) UpdateInvoice(db database.DatabaseManager, invoiceNumber string, invoiceData []byte) error {
	result := db.DB().Model(&entities.Invoice{}).Where("invoice_number = ?", invoiceNumber).Update("invoice_data", invoiceData)
	return result.Error
}

func (r *InvoiceRepository) UpdateInvoiceDataByID(db database.DatabaseManager, invoiceID string, invoiceData []byte) error {
	result := db.DB().Model(&entities.Invoice{}).Where("id = ?", invoiceID).Update("invoice_data", invoiceData)
	return result.Error
}

func (r *InvoiceRepository) UpdateInvoiceDataAndPaymentStatusByID(db database.DatabaseManager, invoiceID string, invoiceData []byte, paymentStatus string) error {
	result := db.DB().Model(&entities.Invoice{}).Where("id = ?", invoiceID).Updates(map[string]interface{}{
		"invoice_data":   invoiceData,
		"payment_status": paymentStatus,
	})
	return result.Error
}

func (r *InvoiceRepository) SaveInvoice(db database.DatabaseManager, invoice *entities.Invoice) error {
	return db.DB().Save(invoice).Error
}

type invoicePeriodCurrencyResult struct {
	Period             string  `gorm:"column:period"`
	Currency           string  `gorm:"column:currency"`
	TotalInvoices      int64   `gorm:"column:total_invoices"`
	SuccessfulInvoices int64   `gorm:"column:successful_invoices"`
	PartialInvoices    int64   `gorm:"column:partial_invoices"`
	FailedInvoices     int64   `gorm:"column:failed_invoices"`
	TotalAmount        float64 `gorm:"column:total_amount"`
	TaxAmount          float64 `gorm:"column:tax_amount"`
}

func (r *InvoiceRepository) GetInvoiceStats(
	db *gorm.DB,
	businessID *string,
	aggregatorID *string,
) (*entities.InvoiceStatsResponseData, error) {

	currencyExpr := `COALESCE(
		NULLIF(TRIM(UPPER(invoices.invoice_data->>'document_currency_code')), ''),
		NULLIF(TRIM(UPPER(invoices.invoice_data->>'currency_code')), ''),
		NULLIF(TRIM(UPPER(invoices.platform_metadata->'zoho'->>'currency_code')), ''),
		'NGN'
	)`

	monthlyQuery := fmt.Sprintf(`
	SELECT 
		TO_CHAR(created_at, 'YYYYMM') AS period,
		%s AS currency,
		COUNT(*) AS total_invoices,

		-- Successful invoices
		SUM(
			CASE
				WHEN current_status = 'confirmed_invoice'
					AND current_step_status = 'success'
				THEN 1
				ELSE 0
			END
		) AS successful_invoices,

		-- Partial invoices
		SUM(
			CASE
				WHEN current_status = 'confirmed_invoice'
					AND current_step_status != 'success'
				THEN 1

				WHEN current_status = 'transmitted_invoice'
					THEN 1

				WHEN current_status = 'signed_invoice'
					AND current_step_status = 'success'
				THEN 1

				ELSE 0
			END
		) AS partial_invoices,

		-- Failed invoices
		SUM(
			CASE
				WHEN current_status = 'signed_invoice'
					AND current_step_status != 'success'
				THEN 1

				WHEN current_status NOT IN (
					'confirmed_invoice',
					'signed_invoice',
					'transmitted_invoice'
				)
				THEN 1

				ELSE 0
			END
		) AS failed_invoices,

		-- Total amount for completed (successful) or partial_success invoices
		COALESCE(
			SUM(
				CASE
					WHEN current_status = 'confirmed_invoice'
						THEN total_amount
					WHEN current_status = 'transmitted_invoice'
						THEN total_amount
					WHEN current_status = 'signed_invoice'
						AND current_step_status = 'success'
						THEN total_amount
					ELSE 0
				END
			),
			0
		) AS total_amount,

		-- Tax amount for completed (successful) or partial_success invoices
		COALESCE(
			SUM(
				CASE
					WHEN current_status = 'confirmed_invoice'
						THEN tax_amount
					WHEN current_status = 'transmitted_invoice'
						THEN tax_amount
					WHEN current_status = 'signed_invoice'
						AND current_step_status = 'success'
						THEN tax_amount
					ELSE 0
				END
			),
			0
		) AS tax_amount

	FROM invoices

	LEFT JOIN LATERAL (
		SELECT entry->>'status' AS current_step_status
		FROM jsonb_array_elements(invoices.status_history) AS entry
		WHERE entry->>'step' = invoices.current_status
		ORDER BY (entry->>'timestamp')::timestamptz DESC
		LIMIT 1
	) AS current_step ON true

	WHERE deleted_at IS NULL
	`, currencyExpr)

	monthlyArgs := []interface{}{}

	if businessID != nil && *businessID != "" {
		monthlyQuery += " AND business_id = ?"
		monthlyArgs = append(monthlyArgs, *businessID)
	}

	if aggregatorID != nil && *aggregatorID != "" {
		monthlyQuery += " AND aggregator_id = ?"
		monthlyArgs = append(monthlyArgs, *aggregatorID)
	}

	monthlyQuery += fmt.Sprintf(`
	GROUP BY TO_CHAR(created_at, 'YYYYMM'), %s
	ORDER BY period DESC, currency ASC;
	`, currencyExpr)

	var monthlyRaw []invoicePeriodCurrencyResult
	if err := db.Raw(monthlyQuery, monthlyArgs...).Scan(&monthlyRaw).Error; err != nil {
		return nil, err
	}

	dailyQuery := fmt.Sprintf(`
	SELECT 
		TO_CHAR(DATE(created_at), 'YYYY-MM-DD') AS period,
		%s AS currency,
		COUNT(*) AS total_invoices,

		-- Successful invoices
		SUM(
			CASE
				WHEN current_status = 'confirmed_invoice'
					AND current_step_status = 'success'
				THEN 1
				ELSE 0
			END
		) AS successful_invoices,

		-- Partial invoices
		SUM(
			CASE
				WHEN current_status = 'confirmed_invoice'
					AND current_step_status != 'success'
				THEN 1

				WHEN current_status = 'transmitted_invoice'
					THEN 1

				WHEN current_status = 'signed_invoice'
					AND current_step_status = 'success'
				THEN 1

				ELSE 0
			END
		) AS partial_invoices,

		-- Failed invoices
		SUM(
			CASE
				WHEN current_status = 'signed_invoice'
					AND current_step_status != 'success'
				THEN 1

				WHEN current_status NOT IN (
					'confirmed_invoice',
					'signed_invoice',
					'transmitted_invoice'
				)
				THEN 1

				ELSE 0
			END
		) AS failed_invoices,

		-- Total amount for completed (successful) or partial_success invoices
		COALESCE(
			SUM(
				CASE
					WHEN current_status = 'confirmed_invoice'
						THEN total_amount
					WHEN current_status = 'transmitted_invoice'
						THEN total_amount
					WHEN current_status = 'signed_invoice'
						AND current_step_status = 'success'
						THEN total_amount
					ELSE 0
				END
			),
			0
		) AS total_amount,

		-- Tax amount for completed (successful) or partial_success invoices
		COALESCE(
			SUM(
				CASE
					WHEN current_status = 'confirmed_invoice'
						THEN tax_amount
					WHEN current_status = 'transmitted_invoice'
						THEN tax_amount
					WHEN current_status = 'signed_invoice'
						AND current_step_status = 'success'
						THEN tax_amount
					ELSE 0
				END
			),
			0
		) AS tax_amount

	FROM invoices

	LEFT JOIN LATERAL (
		SELECT entry->>'status' AS current_step_status
		FROM jsonb_array_elements(invoices.status_history) AS entry
		WHERE entry->>'step' = invoices.current_status
		ORDER BY (entry->>'timestamp')::timestamptz DESC
		LIMIT 1
	) AS current_step ON true

	WHERE deleted_at IS NULL AND created_at >= NOW() - INTERVAL '14 days'
	`, currencyExpr)

	dailyArgs := []interface{}{}

	if businessID != nil && *businessID != "" {
		dailyQuery += " AND business_id = ?"
		dailyArgs = append(dailyArgs, *businessID)
	}

	if aggregatorID != nil && *aggregatorID != "" {
		dailyQuery += " AND aggregator_id = ?"
		dailyArgs = append(dailyArgs, *aggregatorID)
	}

	dailyQuery += fmt.Sprintf(`
	GROUP BY DATE(created_at), %s
	ORDER BY period DESC, currency ASC;
	`, currencyExpr)

	var dailyRaw []invoicePeriodCurrencyResult
	if err := db.Raw(dailyQuery, dailyArgs...).Scan(&dailyRaw).Error; err != nil {
		return nil, err
	}

	monthlyResults := make([]entities.MonthlyInvoiceStatsDto, 0)
	monthMap := make(map[string]int)

	totalCurrenciesMap := make(map[string]*entities.CurrencyStatsDto)
	totalStats := entities.InvoiceStatsDto{
		Currencies:            make([]entities.CurrencyStatsDto, 0),
		TotalAmountByCurrency: make(map[string]float64),
	}

	for _, row := range monthlyRaw {
		idx, exists := monthMap[row.Period]
		if !exists {
			monthlyResults = append(monthlyResults, entities.MonthlyInvoiceStatsDto{
				Month:                 row.Period,
				Currencies:            make([]entities.CurrencyStatsDto, 0),
				TotalAmountByCurrency: make(map[string]float64),
			})
			idx = len(monthlyResults) - 1
			monthMap[row.Period] = idx
		}

		m := &monthlyResults[idx]
		m.TotalInvoices += row.TotalInvoices
		m.SuccessfulInvoices += row.SuccessfulInvoices
		m.PartialInvoices += row.PartialInvoices
		m.FailedInvoices += row.FailedInvoices
		m.TotalAmountByCurrency[row.Currency] += row.TotalAmount
		m.Currencies = append(m.Currencies, entities.CurrencyStatsDto{
			Currency:           row.Currency,
			TotalAmount:        row.TotalAmount,
			TaxAmount:          row.TaxAmount,
			TotalInvoices:      row.TotalInvoices,
			SuccessfulInvoices: row.SuccessfulInvoices,
			PartialInvoices:    row.PartialInvoices,
			FailedInvoices:     row.FailedInvoices,
		})

		// Aggregate overall totals
		totalStats.TotalInvoices += row.TotalInvoices
		totalStats.SuccessfulInvoices += row.SuccessfulInvoices
		totalStats.PartialInvoices += row.PartialInvoices
		totalStats.FailedInvoices += row.FailedInvoices
		totalStats.TotalAmountByCurrency[row.Currency] += row.TotalAmount

		cStat, cExists := totalCurrenciesMap[row.Currency]
		if !cExists {
			cStat = &entities.CurrencyStatsDto{
				Currency: row.Currency,
			}
			totalCurrenciesMap[row.Currency] = cStat
		}
		cStat.TotalAmount += row.TotalAmount
		cStat.TaxAmount += row.TaxAmount
		cStat.TotalInvoices += row.TotalInvoices
		cStat.SuccessfulInvoices += row.SuccessfulInvoices
		cStat.PartialInvoices += row.PartialInvoices
		cStat.FailedInvoices += row.FailedInvoices
	}

	currencyNames := make([]string, 0, len(totalCurrenciesMap))
	for cur := range totalCurrenciesMap {
		currencyNames = append(currencyNames, cur)
	}
	sort.Strings(currencyNames)

	totalCurrencies := make([]entities.CurrencyStatsDto, 0, len(currencyNames))
	for _, cur := range currencyNames {
		totalCurrencies = append(totalCurrencies, *totalCurrenciesMap[cur])
	}
	totalStats.Currencies = totalCurrencies

	dailyResults := make([]entities.DailyInvoiceStatsDto, 0)
	dateMap := make(map[string]int)

	for _, row := range dailyRaw {
		idx, exists := dateMap[row.Period]
		if !exists {
			dailyResults = append(dailyResults, entities.DailyInvoiceStatsDto{
				Date:                  row.Period,
				Currencies:            make([]entities.CurrencyStatsDto, 0),
				TotalAmountByCurrency: make(map[string]float64),
			})
			idx = len(dailyResults) - 1
			dateMap[row.Period] = idx
		}

		d := &dailyResults[idx]
		d.TotalInvoices += row.TotalInvoices
		d.SuccessfulInvoices += row.SuccessfulInvoices
		d.PartialInvoices += row.PartialInvoices
		d.FailedInvoices += row.FailedInvoices
		d.TotalAmountByCurrency[row.Currency] += row.TotalAmount
		d.Currencies = append(d.Currencies, entities.CurrencyStatsDto{
			Currency:           row.Currency,
			TotalAmount:        row.TotalAmount,
			TaxAmount:          row.TaxAmount,
			TotalInvoices:      row.TotalInvoices,
			SuccessfulInvoices: row.SuccessfulInvoices,
			PartialInvoices:    row.PartialInvoices,
			FailedInvoices:     row.FailedInvoices,
		})
	}

	return &entities.InvoiceStatsResponseData{
		Total:                 totalStats,
		TotalAmountByCurrency: totalStats.TotalAmountByCurrency,
		Monthly:               monthlyResults,
		Daily:                 dailyResults,
	}, nil
}
