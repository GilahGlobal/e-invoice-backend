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
	
	// Endpoints open to external API clients
	invoiceUrlSec.Patch("/update", invoiceController.BulkUpdateInvoice)
	invoiceUrlSec.Patch("/update/:irn", invoiceController.UpdateInvoice)

	// Endpoints restricted to frontend only
	invoiceFrontend := invoiceUrlSec.Group("", middleware.RequireFrontend())
	{
		invoiceFrontend.Get("/stats", invoiceController.GetInvoiceStats)
		invoiceFrontend.Get("", invoiceController.GetAllInvoices)
		invoiceFrontend.Get("/:invoice_id", invoiceController.GetInvoiceDetails)
		invoiceFrontend.Delete("/:invoice_id", invoiceController.DeleteInvoice)
		invoiceFrontend.Post("/upload", invoiceController.UploadInvoice)
		invoiceFrontend.Patch("/upload", invoiceController.ModifyInvoice)
	}
	{
		invoiceFrontend.Post("/validate-irn", invoiceController.ValidateIRN)
		invoiceFrontend.Post("/validate", invoiceController.ValidateInvoice)
		invoiceFrontend.Post("/sign", invoiceController.SignInvoice)
		invoiceFrontend.Post("/sign-irn", invoiceController.SignIRN)
		invoiceFrontend.Post("/generate-irn", invoiceController.GenerateIRN)
		invoiceFrontend.Get("/confirm/:irn", invoiceController.ConfirmInvoice)
		invoiceFrontend.Get("/download/:irn", invoiceController.DownloadInvoice)
	}

	transmit := invoiceFrontend.Group("/transmit")
	{
		transmit.Post("/:irn", invoiceController.TransmitInvoice)
		transmit.Get("/confirm/:irn", invoiceController.TransmitConfirmInvoice)
		transmit.Get("/lookup-irn/:irn", invoiceController.LookUpIRN)
		transmit.Get("/lookup-tin/:tin", invoiceController.LookUpTIN)
		transmit.Get("/lookup-party/:party_id", invoiceController.LookUpPartyID)
		transmit.Get("/pull", invoiceController.TransmitPull)
		transmit.Get("/health-check", invoiceController.DebugHealthCheck)

	}
}
