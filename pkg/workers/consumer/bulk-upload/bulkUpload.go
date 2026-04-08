package bulkupload

import (
	"context"
	"einvoice-access-point/internal/dtos"
	"einvoice-access-point/internal/services/invoice"
	"einvoice-access-point/pkg/database"
	"einvoice-access-point/pkg/middleware"
	"einvoice-access-point/pkg/models"
	"einvoice-access-point/pkg/s3"
	"einvoice-access-point/pkg/utility"
	"einvoice-access-point/pkg/workers"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/hibiken/asynq"
)

const TypeBulkUpload = "bulk:upload"

type BulkUploadConsumer struct {
	db             *database.Database
	testDb         *database.Database
	logger         *utility.Logger
	validator      *validator.Validate
	excelProcessor *ExcelProcessor
	csvProcessor   *CSVProcessor
}

func NewBulkUploadConsumer(db, testDB *database.Database, logger *utility.Logger) *BulkUploadConsumer {
	v := validator.New()

	// Register custom validations
	v.RegisterValidation("nrsdate", func(fl validator.FieldLevel) bool {
		dateStr := fl.Field().String()
		_, err := time.Parse("2006-01-02", dateStr)
		return err == nil
	})

	return &BulkUploadConsumer{
		db:             db,
		testDb:         testDB,
		logger:         logger,
		validator:      v,
		excelProcessor: NewExcelProcessor(v),
		csvProcessor:   NewCSVProcessor(v),
	}
}

func (qc *BulkUploadConsumer) HandleBulkUploadTask(ctx context.Context, t *asynq.Task) error {
	// return nil
	const (
		maxWorkers = 10
		maxRetries = 3
	)

	// 1. Validate payload
	var payload workers.BulkUploadInput
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	if payload.FileKey == "" {
		return fmt.Errorf("empty file key in payload")
	}

	var fileBytes []byte
	var downloadErr error
	fileBytes, downloadErr = s3.DownloadFileFromS3(ctx, payload.FileKey)

	if downloadErr != nil {
		log.Println(downloadErr)
		return fmt.Errorf("failed to download file from S3 after %d attempts: %w", maxRetries, downloadErr)
	}

	// 3. Parse file content
	fileType := DetermineFileType(fileBytes, payload.FileKey)
	log.Println("Detected file type:", fileType)

	var invoices []dtos.UploadInvoiceRequestDto
	var stats *ProcessingStats
	var validationErrors []error

	switch fileType {
	case "excel":
		invoices, stats, validationErrors = qc.excelProcessor.ProcessExcel(fileBytes, payload.BusinessID)
	case "csv":
		invoices, stats, validationErrors = qc.csvProcessor.ProcessCSV(fileBytes, payload.BusinessID)
	default:
		return fmt.Errorf("unsupported file type: %s", fileType)
	}

	if len(validationErrors) > 0 {
		log.Println(validationErrors)
		qc.storeValidationErrors(payload.BulkID, payload.FileKey, payload.BusinessID, validationErrors, payload.IsSandbox)
	}

	log.Println("Processing bulk upload",
		"file_key", payload.FileKey,
		"total_rows", stats.TotalRows,
		"valid_rows", stats.ValidRows,
		"invalid_rows", stats.InvalidRows,
	)

	if stats.ValidRows == 0 {
		log.Println("No valid invoices found after validation",
			"total_invoices", stats.TotalRows,
			"validation_errors", stats.TotalErrors)
		// Validation errors (if any) already persisted above.
		return nil
	}

	// 5. Process validated invoices with controlled concurrency
	processedResults := qc.processValidatedInvoices(
		ctx,
		invoices,
		maxWorkers,
		payload.ID,
		payload.BusinessID,
		payload.ServiceID,
		payload.IsSandbox,
	)

	stats.SuccessfulInvoices = processedResults.SuccessCount
	stats.PartiallySuccessfulInvoices = processedResults.PartialCount
	stats.UnsuccessfulInvoices = processedResults.ErrorCount
	stats.TotalErrors += processedResults.ErrorCount

	ststsBytes, _ := json.MarshalIndent(stats, "", "  ")
	log.Println("Bulk upload stats:", string(ststsBytes))

	var errorsToStore []error
	for _, processErr := range processedResults.Errors {
		if strings.HasPrefix(processErr.Error, "FIRS validation failed:") {
			errMsg := strings.TrimPrefix(processErr.Error, "FIRS validation failed: ")
			errorsToStore = append(errorsToStore, fmt.Errorf("invoice %s: %s", processErr.InvoiceNumber, errMsg))
		} else if strings.HasPrefix(processErr.Error, "invoice cannot be overwritten:") {
			log.Printf("Duplicate invoice detected: %s (same invoice number)\n", processErr.InvoiceNumber)
			errorsToStore = append(errorsToStore, fmt.Errorf("invoice %s: duplicate invoice sent (same invoice number)", processErr.InvoiceNumber))
		} else if strings.Contains(strings.ToLower(processErr.Error), "duplicate") {
			log.Printf("Duplicate invoice detected: %s - %s\n", processErr.InvoiceNumber, processErr.Error)
			errorsToStore = append(errorsToStore, fmt.Errorf("invoice %s: duplicate invoice sent - %s", processErr.InvoiceNumber, processErr.Error))
		}
	}

	if len(errorsToStore) > 0 {
		validationErrors = append(validationErrors, errorsToStore...)
		qc.storeValidationErrors(payload.BulkID, payload.FileKey, payload.BusinessID, validationErrors, payload.IsSandbox)
	}

	// 6. Summarize results
	qc.logResults(payload.BulkID, payload.FileKey, payload.BusinessID, stats, processedResults, payload.IsSandbox)

	return nil
}
func (qc *BulkUploadConsumer) processValidatedInvoices(ctx context.Context, invoices []dtos.UploadInvoiceRequestDto, maxWorkers int, id, businessId, serviceID string, isSandbox bool) *ProcessResults {
	results := &ProcessResults{
		SuccessCount: 0,
		PartialCount: 0,
		ErrorCount:   0,
		Errors:       make([]ProcessError, 0),
	}
	var mu sync.Mutex

	// Create worker pool
	workChan := make(chan *dtos.UploadInvoiceRequestDto, len(invoices))
	resultsChan := make(chan ProcessResult, len(invoices))

	// Start workers
	var wg sync.WaitGroup
	for i := 0; i < min(maxWorkers, len(invoices)); i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			qc.invoiceWorker(ctx, workerID, workChan, resultsChan, id, businessId, serviceID, isSandbox)
		}(i)
	}

	// Send work
	for _, invoice := range invoices {
		workChan <- &invoice
	}
	close(workChan)

	// Wait for workers and collect results
	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	// Aggregate results
	for result := range resultsChan {
		mu.Lock()
		if result.Status == models.StatusConfirmed || result.Posted {
			results.SuccessCount++
		} else if result.Status == models.StatusSignedInvoice || result.Status == models.StatusTransmitted {
			results.PartialCount++
		} else if result.Error != nil {
			results.ErrorCount++
			results.Errors = append(results.Errors, ProcessError{
				InvoiceNumber: result.Invoice.InvoiceNumber,
				Error:         result.Error.Error(),
			})
		} else {
			results.ErrorCount++
			results.Errors = append(results.Errors, ProcessError{
				InvoiceNumber: result.Invoice.InvoiceNumber,
				Error:         "invoice did not reach signed/transmitted/confirmed status",
			})
		}
		mu.Unlock()
	}

	return results
}

func (qc *BulkUploadConsumer) invoiceWorker(ctx context.Context, workerID int, workChan <-chan *dtos.UploadInvoiceRequestDto, resultsChan chan<- ProcessResult, id, businessId, serviceID string, isSandbox bool) {
	for invoice := range workChan {
		select {
		case <-ctx.Done():
			log.Println("Worker stopped by context", "worker_id", workerID)
			return
		default:
			result := ProcessResult{Invoice: invoice}

			// processing logic here
			posted, status, err := qc.processSingleInvoice(ctx, invoice, id, businessId, serviceID, isSandbox)
			result.Posted = posted
			result.Status = status
			if err != nil {
				result.Error = err
				log.Println("Failed to process invoice",
					"worker_id", workerID,
					"invoice_id", invoice.InvoiceNumber,
					"error", err)
			}

			resultsChan <- result
		}
	}
}

func (qc *BulkUploadConsumer) processSingleInvoice(ctx context.Context, invoicePayload *dtos.UploadInvoiceRequestDto, id, businessId, serviceID string, isSandbox bool) (bool, string, error) {

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	db := middleware.GetDatabaseInstance(isSandbox, qc.db, qc.testDb)
	invoiceExists, err := invoice.GetInvoiceByInvoiceNumber(db, invoicePayload.InvoiceNumber, id)

	if err != nil {
		return false, "", fmt.Errorf("database error: %w", err)
	}

	if invoiceExists != nil {
		blockedStatuses := map[string]bool{
			models.StatusSignedInvoice: true,
			models.StatusTransmitted:   true,
			models.StatusConfirmed:     true,
		}
		if blockedStatuses[invoiceExists.CurrentStatus] {
			return false, "", fmt.Errorf("invoice cannot be overwritten: %s", invoicePayload.InvoiceNumber)
		}
	}

	var irnPayload dtos.InvoiceData
	if invoicePayload.IRN == nil {
		IRNData, err := invoice.IRNGeneration(db, id, invoicePayload.InvoiceNumber, serviceID, businessId, isSandbox)
		if err != nil {
			rd := *err
			return false, "", fmt.Errorf("IRN generation failed: %s", rd.Message)
		}
		irnPayload = *IRNData
		invoicePayload.IRN = &irnPayload.IRN
	} else {
		irnPayload = dtos.InvoiceData{
			QRCode:  invoiceExists.QrCode,
			QRCode2: invoiceExists.EncryptedIRN,
		}
		invoicePayload.IRN = &invoiceExists.IRN
	}
	createdInvoice, _, err, invoiceSigned := invoice.CreateInvoice(db, *invoicePayload, invoicePayload.InvoiceNumber, id, irnPayload.QRCode, irnPayload.QRCode2, invoiceExists, isSandbox)
	currentStatus := ""
	if createdInvoice != nil {
		currentStatus = createdInvoice.CurrentStatus
	}
	if err != nil && !invoiceSigned {
		errStr := err.Error()
		if strings.Contains(errStr, "failed to validate invoice:") {
			errStr = strings.Replace(errStr, "failed to process invoice through all steps: ", "", 1)
			return false, currentStatus, fmt.Errorf("FIRS validation failed: %s", errStr)
		}
		errorArray := strings.Split(errStr, "-")
		return false, currentStatus, fmt.Errorf("invoice creation failed: %s", strings.TrimSpace(errorArray[len(errorArray)-1]))
	}

	if err != nil && invoiceSigned {
		log.Println("invoice processing completed with non-blocking error",
			"invoice_id", invoicePayload.InvoiceNumber,
			"error", err)
	}

	return err == nil && invoiceSigned, currentStatus, nil
}

func (qc *BulkUploadConsumer) logResults(bulkID, fileKey, businessID string, stats *ProcessingStats, results *ProcessResults, isSanbox bool) {
	db := middleware.GetDatabaseInstance(isSanbox, qc.db, qc.testDb)
	payload := map[string]interface{}{
		"TotalRows":                   stats.TotalRows,
		"ValidRows":                   stats.ValidRows,
		"SuccessfulInvoices":          stats.SuccessfulInvoices,
		"PartiallySuccessfulInvoices": stats.PartiallySuccessfulInvoices,
		"UnsuccessfulInvoices":        stats.UnsuccessfulInvoices,
		"Duration":                    stats.Duration,
		"StartTime":                   &stats.StartTime,
		"EndTime":                     &stats.EndTime,
	}
	err := invoice.UpdateBulkUploadLog(db, bulkID, fileKey, businessID, payload)
	if err != nil {
		qc.logger.Error("Failed to update bulk upload log", "error", err)
	}
	qc.logger.Info("Bulk upload processing completed",
		"file_key", fileKey,
		"successful", results.SuccessCount,
		"partial", results.PartialCount,
		"failed", results.ErrorCount,
		"total", results.SuccessCount+results.PartialCount+results.ErrorCount)

	if results.ErrorCount > 0 {
		qc.logger.Warning("Processing errors encountered",
			"error_count", results.ErrorCount,
			"first_errors", results.Errors[:min(5, len(results.Errors))])
	}
}

func (qc *BulkUploadConsumer) storeValidationErrors(bulkID, fileKey, businessID string, errs []error, isSandbox bool) {
	if len(errs) == 0 {
		return
	}

	validationErrors := make([]ValidationError, 0, len(errs))
	for i, err := range errs {
		msg := err.Error()
		invoiceNumber := ""
		if idx := strings.Index(msg, "invoice "); idx != -1 {
			start := idx + len("invoice ")
			end := start
			for end < len(msg) {
				c := msg[end]
				if c == ':' || c == ' ' || c == ',' || c == ';' {
					break
				}
				end++
			}
			if end > start {
				invoiceNumber = msg[start:end]
			}
		}
		rowNumber := 0
		if idx := strings.Index(msg, "row "); idx != -1 {
			start := idx + len("row ")
			for start < len(msg) && msg[start] == ' ' {
				start++
			}
			end := start
			for end < len(msg) && msg[end] >= '0' && msg[end] <= '9' {
				end++
			}
			if end > start {
				if n, parseErr := strconv.Atoi(msg[start:end]); parseErr == nil {
					rowNumber = n
				}
			}
		}
		if rowNumber == 0 {
			rowNumber = i + 1
		}

		var parsedErr any = msg
		if jsonStart := strings.Index(msg, "{"); jsonStart != -1 {
			jsonStr := strings.TrimSpace(msg[jsonStart:])
			if strings.HasSuffix(jsonStr, "}") {
				var jsonObj map[string]interface{}
				if err := json.Unmarshal([]byte(jsonStr), &jsonObj); err == nil {
					parsedErr = jsonObj
				}
			}
		}

		validationErrors = append(validationErrors, ValidationError{
			InvoiceIndex:  rowNumber,
			InvoiceNumber: invoiceNumber,
			Error:         parsedErr,
		})
	}

	payload := map[string]interface{}{
		"file_key":    fileKey,
		"error_count": len(validationErrors),
		"errors":      validationErrors,
		"stored_at":   time.Now().UTC(),
	}

	errorsJSON, err := json.Marshal(validationErrors)
	if err != nil {
		qc.logger.Error("Failed to marshal validation errors for db", "error", err)
		return
	}

	db := middleware.GetDatabaseInstance(isSandbox, qc.db, qc.testDb)
	if err := invoice.StoreBulkUploadValidationErrors(db, bulkID, fileKey, businessID, errorsJSON, len(validationErrors)); err != nil {
		qc.logger.Error("Failed to store validation errors in db", "error", err)
	}

	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		qc.logger.Error("Failed to marshal validation errors", "error", err)
		return
	}

	dir := filepath.Join("tmp", "bulk_upload_errors")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		qc.logger.Error("Failed to create validation error directory", "error", err)
		return
	}

	sanitizedKey := strings.NewReplacer("/", "_", "\\", "_", " ", "_").Replace(fileKey)
	if sanitizedKey == "" {
		sanitizedKey = fmt.Sprintf("bulk_upload_%s", time.Now().UTC().Format("20060102T150405Z"))
	}
	filename := fmt.Sprintf("%s_validation_errors.json", sanitizedKey)
	path := filepath.Join(dir, filename)

	if err := os.WriteFile(path, data, 0o644); err != nil {
		qc.logger.Error("Failed to store validation errors", "error", err)
		return
	}

	qc.logger.Info("Stored validation errors", "file_key", fileKey, "path", path, "error_count", len(validationErrors))
}
