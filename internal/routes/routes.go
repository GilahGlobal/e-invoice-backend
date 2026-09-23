package routes

import (
	"einvoice-access-point/internal/config"
	"einvoice-access-point/internal/core"
	"einvoice-access-point/internal/data/database"
	"einvoice-access-point/internal/utility"

	_ "einvoice-access-point/docs/external"
	_ "einvoice-access-point/docs/frontend"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/swagger"
)

func Setup(app *fiber.App, logger *utility.Logger, validatorRef *validator.Validate, db, testDb *database.Database, keys *utility.CryptoKeys) {
	apiVersion := "api/v1"

	container := core.NewContainer(config.GetConfig(), db, testDb, logger, validatorRef, keys)

	// Swagger UI for external integrations
	app.Get("/swagger/external/*", swagger.New(swagger.Config{
		InstanceName: "external",
		Title:        "External Integration API",
	}))

	// Swagger UI for frontend integration (shows all endpoints)
	app.Get("/swagger/frontend/*", swagger.New(swagger.Config{
		InstanceName: "frontend",
		Title:        "Frontend API",
	}))

	// Redirect base swagger path to a landing page
	app.Get("/swagger", func(c *fiber.Ctx) error {
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"message": "Swagger Documentation",
			"docs": fiber.Map{
				"external": "/swagger/external/index.html",
				"frontend": "/swagger/frontend/index.html",
			},
		})
	})

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

