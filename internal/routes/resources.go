package routes

import (
	"einvoice-access-point/internal/app/resources"
	"einvoice-access-point/internal/core"
	"einvoice-access-point/internal/middleware"
	"fmt"

	"github.com/gofiber/fiber/v2"
)

func ResourcesRoute(app *fiber.App, ApiVersion string, c *core.Container) {
	resourcesController := resources.NewHandler(c.Logger)

	resourcesGroup := app.Group(fmt.Sprintf("%v/resources", ApiVersion), middleware.Authorize(c.DB.Postgresql.DB(), c.TestDB.Postgresql.DB()), middleware.SelectDatabaseFromClaims(c.DB, c.TestDB))
	{
		resourcesGroup.Get("/invoice-types", resourcesController.GetInvoiceTypes)
		resourcesGroup.Get("/payment-means", resourcesController.GetPaymentMeans)
		resourcesGroup.Get("/tax-categories", resourcesController.GetTaxCategories)
		resourcesGroup.Get("/hsn-codes", resourcesController.GetHSNCodes)
		resourcesGroup.Get("/service-codes", resourcesController.GetServiceCodes)
		resourcesGroup.Get("/currencies", resourcesController.GetCurrencies)
		resourcesGroup.Get("/lgas", resourcesController.GetLGA)
		resourcesGroup.Get("/countries", resourcesController.GetCountries)
		resourcesGroup.Get("/states", resourcesController.GetStates)
	}
}
