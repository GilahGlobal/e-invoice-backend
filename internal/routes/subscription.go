package routes

import (
	"einvoice-access-point/internal/app/subscription"
	"einvoice-access-point/internal/core"
	"einvoice-access-point/internal/middleware"
	"fmt"

	"github.com/gofiber/fiber/v2"
)

func SubscriptionRoute(app *fiber.App, ApiVersion string, c *core.Container) {
	subscriptionController := subscription.NewHandler(c.Validator, c.Logger, c.DB, c.TestDB)

	app.Post(ApiVersion+"/paystack/webhook", middleware.SelectDatabaseFromJSONPath("data.metadata.is_sandbox", c.DB, c.TestDB), subscriptionController.PaystackWebhook)
	subscriptionGroup := app.Group(fmt.Sprintf("%v/subscription", ApiVersion))
	rf := middleware.RequireFrontend()
	{
		subscriptionGroup.Get("/plans", rf, middleware.SelectDatabaseFromQuery("is_sandbox", c.DB, c.TestDB), subscriptionController.GetPlans)
		subscriptionGroup.Post("/plans", rf, middleware.SelectDatabaseFromJSONPath("is_sandbox", c.DB, c.TestDB), subscriptionController.CreatePlan)
	}
}
