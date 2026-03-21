package v1

import (
	"einvoice-access-point/internal/controller/invoice"
	"einvoice-access-point/pkg/database"
	"einvoice-access-point/pkg/middleware"
	"einvoice-access-point/pkg/utility"
	"fmt"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
)

func InvoiceRoute(app *fiber.App, ApiVersion string, validator *validator.Validate, db, testDb *database.Database, logger *utility.Logger, keys *utility.CryptoKeys) *fiber.App {
	invoiceController := invoice.Controller{Db: db, TestDB: testDb, Validator: validator, Logger: logger, Keys: keys}

	invoiceUrlSec := app.Group(fmt.Sprintf("%v/invoice", ApiVersion), middleware.Authorize(db.Postgresql.DB(), testDb.Postgresql.DB()))
	// invoiceUrlSec := app.Group(fmt.Sprintf("%v/invoice", ApiVersion), middleware.Authorize(nil, testDb.Postgresql.DB()))

	{
		invoiceUrlUnSec := app.Group(fmt.Sprintf("%v/zoho", ApiVersion))
		invoiceUrlUnSec.Post("/webhook", invoiceController.HandleZohoWebhook)
	}

	{
		webhookUrl := app.Group(fmt.Sprintf("%v/webhook", ApiVersion))
		webhookUrl.Post("/firs", invoiceController.FirsWebhook)
	}
	{
		invoiceUrlSec.Get("", invoiceController.GetAllInvoices)
		invoiceUrlSec.Get("/:invoice_id", invoiceController.GetInvoiceDetails)
		invoiceUrlSec.Post("/create", invoiceController.CreateInvoice)
		invoiceUrlSec.Delete("/:invoice_id", invoiceController.DeleteInvoice)
		invoiceUrlSec.Post("/upload", invoiceController.UploadInvoice)
	}
	{
		invoiceUrlSec.Post("/validate-irn", invoiceController.ValidateIRN)
		invoiceUrlSec.Post("/validate", invoiceController.ValidateInvoice)
		invoiceUrlSec.Post("/sign", invoiceController.SignInvoice)
		invoiceUrlSec.Post("/sign-irn", invoiceController.SignIRN)
		invoiceUrlSec.Post("/generate-irn", invoiceController.GenerateIRN)
		invoiceUrlSec.Patch("/update/:irn", invoiceController.UpdateInvoice)
		invoiceUrlSec.Get("/confirm/:irn", invoiceController.ConfirmInvoice)
		invoiceUrlSec.Get("/download/:irn", invoiceController.DownloadInvoice)

		invoiceUrlSec.Post("/transmit/:irn", invoiceController.TransmitInvoice)
		invoiceUrlSec.Get("/transmit/confirm/:irn", invoiceController.TransmitConfirmInvoice)
		invoiceUrlSec.Get("/transmit/lookup-irn/:irn", invoiceController.LookUpIRN)
		invoiceUrlSec.Get("/transmit/lookup-tin/:tin", invoiceController.LookUpTIN)
		invoiceUrlSec.Get("/transmit/lookup-party/:party_id", invoiceController.LookUpPartyID)
		invoiceUrlSec.Get("/transmit/pull", invoiceController.TransmitPull)
		invoiceUrlSec.Get("/transmit/health-check", invoiceController.DebugHealthCheck)

	}

	return app
}
