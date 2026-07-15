package routes

import (
	"einvoice-access-point/internal/app/invoice"
	"einvoice-access-point/internal/core"
	"einvoice-access-point/internal/middleware"
	"fmt"

	"github.com/gofiber/fiber/v2"
)

func ResourcesRoute(app *fiber.App, ApiVersion string, c *core.Container) *fiber.App {
	invoiceController := invoice.NewHandler(c.Validator, c.Logger, c.DB, c.TestDB, c.Keys)

	resources := app.Group(fmt.Sprintf("%v/resources", ApiVersion), middleware.Authorize(c.DB.Postgresql.DB(), c.TestDB.Postgresql.DB()), middleware.SelectDatabaseFromClaims(c.DB, c.TestDB))
	{
		resources.Get("/invoice-types", invoiceController.GetInvoiceTypes)
		resources.Get("/payment-means", invoiceController.GetPaymentMeans)
		resources.Get("/tax-categories", invoiceController.GetTaxCategories)
		resources.Get("/hsn-codes", invoiceController.GetHSNCodes)
		resources.Get("/service-codes", invoiceController.GetServiceCodes)
		resources.Get("/currencies", invoiceController.GetCurrencies)
		resources.Get("/lgas", invoiceController.GetLGA)
		resources.Get("/countries", invoiceController.GetCountries)
		resources.Get("/states", invoiceController.GetStates)
	}

	return app
}
