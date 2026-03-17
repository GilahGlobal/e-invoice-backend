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
		maxWorkers       = 10
		maxRetries       = 3
		validationBuffer = 100
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
	var err []error

	switch fileType {
	case "excel":
		invoices, stats, err = qc.excelProcessor.ProcessExcel(fileBytes, payload.BusinessID)
	case "csv":
		invoices, stats, err = qc.csvProcessor.ProcessCSV(fileBytes, payload.BusinessID)
	default:
		return fmt.Errorf("unsupported file type: %s", fileType)
	}

	if err != nil {
		log.Println(err)
		return nil
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
		// Optionally store validation errors for reporting
		// qc.storeValidationErrors(payload.FileKey, validationResults.Errors)
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
	stats.UnsuccessfulInvoices = processedResults.ErrorCount
	stats.TotalErrors += processedResults.ErrorCount

	ststsBytes, _ := json.MarshalIndent(stats, "", "  ")
	log.Println("Bulk upload stats:", string(ststsBytes))
	// 6. Summarize results
	qc.logResults(payload.FileKey, processedResults)

	return nil
}
func (qc *BulkUploadConsumer) processValidatedInvoices(ctx context.Context, invoices []dtos.UploadInvoiceRequestDto, maxWorkers int, id, businessId, serviceID string, isSandbox bool) *ProcessResults {
	results := &ProcessResults{
		SuccessCount: 0,
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
		if result.Error != nil {
			results.ErrorCount++
			results.Errors = append(results.Errors, ProcessError{
				InvoiceNumber: result.Invoice.InvoiceNumber,
				Error:         result.Error.Error(),
			})
		} else {
			results.SuccessCount++
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
			err := qc.processSingleInvoice(ctx, invoice, id, businessId, serviceID, isSandbox)
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

func (qc *BulkUploadConsumer) processSingleInvoice(ctx context.Context, invoicePayload *dtos.UploadInvoiceRequestDto, id, businessId, serviceID string, isSandbox bool) error {

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	db := middleware.GetDatabaseInstance(isSandbox, qc.db, qc.testDb)
	invoiceExists, err := invoice.GetInvoiceByInvoiceNumber(db, invoicePayload.InvoiceNumber, id)

	if err != nil {
		return fmt.Errorf("database error: %w", err)
	}

	if invoiceExists != nil {
		blockedStatuses := map[string]bool{
			models.StatusSignedInvoice: true,
			models.StatusTransmitted:   true,
			models.StatusConfirmed:     true,
		}
		if blockedStatuses[invoiceExists.CurrentStatus] {
			log.Println("invoice cannot be overwritten", invoicePayload.InvoiceNumber)
			return nil
		}
	}

	var irnPayload dtos.InvoiceData
	if invoicePayload.IRN == nil {
		IRNData, err := invoice.IRNGeneration(invoicePayload.InvoiceNumber, serviceID, businessId, isSandbox)
		if err != nil {
			rd := *err
			return fmt.Errorf("IRN generation failed: %s", rd.Message)
		}
		irnPayload = *IRNData
		invoicePayload.IRN = &irnPayload.IRN
	} else {
		irnPayload = dtos.InvoiceData{
			QRCode:       invoiceExists.QrCode,
			EncryptedIRN: invoiceExists.EncryptedIRN,
		}
		invoicePayload.IRN = &invoiceExists.IRN
	}
	_, _, err, invoiceSigned := invoice.CreateInvoice(db, *invoicePayload, invoicePayload.InvoiceNumber, id, irnPayload.QRCode, irnPayload.EncryptedIRN, invoiceExists, isSandbox)
	if err != nil && !invoiceSigned {
		errorArray := strings.Split(err.Error(), "-")
		return fmt.Errorf("invoice creation failed: %s", strings.TrimSpace(errorArray[len(errorArray)-1]))
	}

	return nil
}

func (qc *BulkUploadConsumer) logResults(fileKey string, results *ProcessResults) {
	// err := invoice.UpdateBulkUploadLog(qc.db.Postgresql.DB(), fileKey, results)
	// if err != nil {
	// 	qc.logger.Error("Failed to update bulk upload log", "error", err)
	// }
	qc.logger.Info("Bulk upload processing completed",
		"file_key", fileKey,
		"successful", results.SuccessCount,
		"failed", results.ErrorCount,
		"total", results.SuccessCount+results.ErrorCount)

	if results.ErrorCount > 0 {
		qc.logger.Warning("Processing errors encountered",
			"error_count", results.ErrorCount,
			"first_errors", results.Errors[:min(5, len(results.Errors))])
	}
}
