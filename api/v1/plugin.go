package v1

import (
	"einvoice-access-point/internal/controller/plugin"
	"einvoice-access-point/pkg/database"
	"einvoice-access-point/pkg/utility"
	"fmt"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
)

func PluginRoute(app *fiber.App, ApiVersion string, validator *validator.Validate, db, testDb *database.Database, logger *utility.Logger) *fiber.App {
	pluginController := plugin.Controller{Db: db, TestDB: testDb, Validator: validator, Logger: logger}

	pluginGroup := app.Group(fmt.Sprintf("%v/plugin", ApiVersion))
	{
		pluginGroup.Get("/plans", pluginController.GetPlans)
		pluginGroup.Get("/business", pluginController.CheckBusiness)
		pluginGroup.Post("/register", pluginController.Register)
		pluginGroup.Post("/subscribe", pluginController.Subscribe)
		pluginGroup.Post("/paystack/webhook", pluginController.PaystackWebhook)
		pluginGroup.Post("/invoice-upload", pluginController.UploadInvoice)
	}

	return app
}
