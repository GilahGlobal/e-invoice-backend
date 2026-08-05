package routes

import (
	_ "einvoice-access-point/docs"
	"einvoice-access-point/internal/config"
	"einvoice-access-point/internal/core"
	"einvoice-access-point/internal/data/database"
	"einvoice-access-point/internal/utility"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/swagger"
)

func Setup(app *fiber.App, logger *utility.Logger, validatorRef *validator.Validate, db, testDb *database.Database, keys *utility.CryptoKeys) {
	apiVersion := "api/v1"

	container := core.NewContainer(config.GetConfig(), db, testDb, logger, validatorRef, keys)

	// General API route
	app.Get("/swagger/*", swagger.HandlerDefault)

	// All routes registered
	HealthRoute(app, apiVersion, container)
	AuthRoute(app, apiVersion, container)
	SubscriptionRoute(app, apiVersion, container)
	AggregatorRoute(app, apiVersion, container)
	EntityRoute(app, apiVersion, container)
	BusinessRoute(app, apiVersion, container)
	CallbackRoute(app, apiVersion, container)
	BulkUploadRoute(app, apiVersion, container)
	InvoiceRoute(app, apiVersion, container)
	WebhooksRoute(app, apiVersion, container)
	ResourcesRoute(app, apiVersion, container)
	AdminRoute(app, apiVersion, container)
	RegisterBaseRoutes(app, apiVersion)
}
