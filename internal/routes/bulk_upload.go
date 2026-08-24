package routes

import (
	"einvoice-access-point/internal/app/bulk_upload"
	"einvoice-access-point/internal/core"
	"einvoice-access-point/internal/middleware"
	"fmt"

	"github.com/gofiber/fiber/v2"
)

func BulkUploadRoute(app *fiber.App, ApiVersion string, c *core.Container) {
	handler := bulk_upload.NewHandler(c.Validator, c.Logger, c.DB, c.TestDB)

	bulkUrlSec := app.Group(fmt.Sprintf("%v/invoice", ApiVersion), middleware.Authorize(c.DB.Postgresql.DB(), c.TestDB.Postgresql.DB()), middleware.SelectDatabaseFromClaims(c.DB, c.TestDB))
	bulkUpload := bulkUrlSec.Group("/bulk-upload", middleware.RequireFrontend())
	{
		bulkUpload.Get("", handler.GetBulkUploadLogs)
		bulkUpload.Get("/failures/:id", handler.GetBulkUploadFailedInvoices)
		bulkUpload.Get("/:bulk_id/failed/download", handler.DownloadBulkUploadFailedInvoices)
	}
	bulkUrlSec.Post("/create", handler.CreateInvoice)
}
