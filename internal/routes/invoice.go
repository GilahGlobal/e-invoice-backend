package routes

import (
	"einvoice-access-point/internal/app/invoice"
	"einvoice-access-point/internal/core"
	"einvoice-access-point/internal/middleware"
	"fmt"

	"github.com/gofiber/fiber/v2"
)

func InvoiceRoute(app *fiber.App, ApiVersion string, c *core.Container) *fiber.App {
	invoiceController := invoice.NewHandler(c.Validator, c.Logger, c.DB, c.TestDB, c.Keys)

	invoiceUrlSec := app.Group(fmt.Sprintf("%v/invoice", ApiVersion), middleware.Authorize(c.DB.Postgresql.DB(), c.TestDB.Postgresql.DB()), middleware.SelectDatabaseFromClaims(c.DB, c.TestDB))
	// invoiceUrlSec := app.Group(fmt.Sprintf("%v/invoice", ApiVersion), middleware.Authorize(nil, testDb.Postgresql.DB()))

	webhookUrl := app.Group(fmt.Sprintf("%v/webhook", ApiVersion))
	{
		webhookUrl.Post("/firs", invoiceController.FirsWebhook)
	}

	invoiceUrlUnSec := app.Group(fmt.Sprintf("%v/zoho", ApiVersion), middleware.SelectSandboxDatabase(c.DB, c.TestDB))
	{
		invoiceUrlUnSec.Post("/webhook", invoiceController.HandleZohoWebhook)
	}
	{
		invoiceUrlSec.Get("/stats", invoiceController.GetInvoiceStats)
		invoiceUrlSec.Get("", invoiceController.GetAllInvoices)
		invoiceUrlSec.Get("/:invoice_id", invoiceController.GetInvoiceDetails)
		invoiceUrlSec.Delete("/:invoice_id", invoiceController.DeleteInvoice)
		invoiceUrlSec.Post("/upload", invoiceController.UploadInvoice)
		invoiceUrlSec.Patch("/upload", invoiceController.ModifyInvoice)
	}
	{
		invoiceUrlSec.Post("/validate-irn", invoiceController.ValidateIRN)
		invoiceUrlSec.Post("/validate", invoiceController.ValidateInvoice)
		invoiceUrlSec.Post("/sign", invoiceController.SignInvoice)
		invoiceUrlSec.Post("/sign-irn", invoiceController.SignIRN)
		invoiceUrlSec.Post("/generate-irn", invoiceController.GenerateIRN)
		invoiceUrlSec.Patch("/update", invoiceController.BulkUpdateInvoice)
		invoiceUrlSec.Patch("/update/:irn", invoiceController.UpdateInvoice)
		invoiceUrlSec.Get("/confirm/:irn", invoiceController.ConfirmInvoice)
		invoiceUrlSec.Get("/download/:irn", invoiceController.DownloadInvoice)
	}

	resources := invoiceUrlSec.Group("/resources")
	{
		resources.Get("/invoice-types", invoiceController.GetInvoiceTypes)
		resources.Get("/payment-means", invoiceController.GetPaymentMeans)
		resources.Get("/tax-categories", invoiceController.GetTaxCategories)
		resources.Get("/product-codes", invoiceController.GetProductCodes)
		resources.Get("/service-codes", invoiceController.GetServiceCodes)
		resources.Get("/currencies", invoiceController.GetCurrencies)
		resources.Get("/lgas", invoiceController.GetLGA)
		resources.Get("/countries", invoiceController.GetCountries)
		resources.Get("/states", invoiceController.GetStates)
	}

	transmit := invoiceUrlSec.Group("/transmit")
	{
		transmit.Post("/:irn", invoiceController.TransmitInvoice)
		transmit.Get("/confirm/:irn", invoiceController.TransmitConfirmInvoice)
		transmit.Get("/lookup-irn/:irn", invoiceController.LookUpIRN)
		transmit.Get("/lookup-tin/:tin", invoiceController.LookUpTIN)
		transmit.Get("/lookup-party/:party_id", invoiceController.LookUpPartyID)
		transmit.Get("/pull", invoiceController.TransmitPull)
		transmit.Get("/health-check", invoiceController.DebugHealthCheck)

	}

	return app
}
