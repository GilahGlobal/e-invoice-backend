package v1

import (
	"einvoice-access-point/internal/controller/subscription"
	"einvoice-access-point/pkg/database"
	"einvoice-access-point/pkg/utility"
	"fmt"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
)

func SubscriptionRoute(app *fiber.App, ApiVersion string, validator *validator.Validate, db, testDb *database.Database, logger *utility.Logger) *fiber.App {
	subscriptionController := subscription.Controller{Db: db, TestDb: testDb, Validator: validator, Logger: logger}

	app.Post(ApiVersion+"/paystack/webhook", subscriptionController.PaystackWebhook)
	subscriptionGroup := app.Group(fmt.Sprintf("%v/subscription", ApiVersion))
	{
		subscriptionGroup.Get("/plans", subscriptionController.GetPlans)
		subscriptionGroup.Post("/plans", subscriptionController.CreatePlan)
	}

	return app
}
