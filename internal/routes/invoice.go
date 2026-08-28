package routes

import (
	"einvoice-access-point/internal/app/invoice"
	"einvoice-access-point/internal/core"
	"einvoice-access-point/internal/middleware"
	"fmt"

	"github.com/gofiber/fiber/v2"
)

func InvoiceRoute(app *fiber.App, ApiVersion string, c *core.Container) {
	invoiceController := invoice.NewHandler(c.Validator, c.Logger, c.DB, c.TestDB, c.Keys)

	invoiceUrlSec := app.Group(fmt.Sprintf("%v/invoice", ApiVersion), middleware.Authorize(c.DB.Postgresql.DB(), c.TestDB.Postgresql.DB()), middleware.SelectDatabaseFromClaims(c.DB, c.TestDB))

	rf := middleware.RequireFrontend()

	// Endpoints open to external API clients
	invoiceUrlSec.Patch("/update", invoiceController.BulkUpdateInvoice)
	invoiceUrlSec.Patch("/update/:irn", invoiceController.UpdateInvoice)
	invoiceUrlSec.Post("/upload", invoiceController.UploadInvoice)
	invoiceUrlSec.Patch("/upload", invoiceController.ModifyInvoice)
	invoiceUrlSec.Get("/download/:irn", invoiceController.DownloadInvoice)

	// Endpoints restricted to frontend only
	invoiceUrlSec.Get("/stats", rf, invoiceController.GetInvoiceStats)
	invoiceUrlSec.Get("", invoiceController.GetAllInvoices)
	invoiceUrlSec.Post("/validate-irn", rf, invoiceController.ValidateIRN)
	invoiceUrlSec.Post("/validate", rf, invoiceController.ValidateInvoice)
	invoiceUrlSec.Post("/sign", rf, invoiceController.SignInvoice)
	invoiceUrlSec.Post("/sign-irn", rf, invoiceController.SignIRN)
	invoiceUrlSec.Post("/generate-irn", rf, invoiceController.GenerateIRN)
	invoiceUrlSec.Get("/confirm/:irn", rf, invoiceController.ConfirmInvoice)

	transmit := invoiceUrlSec.Group("/transmit", rf)
	{
		transmit.Post("/:irn", invoiceController.TransmitInvoice)
		transmit.Get("/confirm/:irn", invoiceController.TransmitConfirmInvoice)
		transmit.Get("/lookup-irn/:irn", invoiceController.LookUpIRN)
		transmit.Get("/lookup-tin/:tin", invoiceController.LookUpTIN)
		transmit.Get("/lookup-party/:party_id", invoiceController.LookUpPartyID)
		transmit.Get("/pull", invoiceController.TransmitPull)
		transmit.Get("/health-check", invoiceController.DebugHealthCheck)
	}

	// Parameterized routes MUST be registered last
	invoiceUrlSec.Get("/:invoice_id", invoiceController.GetInvoiceDetails)
	invoiceUrlSec.Delete("/:invoice_id", rf, invoiceController.DeleteInvoice)
}
