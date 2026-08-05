package routes

import (
	"einvoice-access-point/internal/app/webhooks"
	"einvoice-access-point/internal/core"
	"einvoice-access-point/internal/middleware"
	"fmt"

	"github.com/gofiber/fiber/v2"
)

func WebhooksRoute(app *fiber.App, ApiVersion string, c *core.Container) {
	webhookController := webhooks.NewHandler(c.Validator, c.Logger, c.DB, c.TestDB, c.Keys)

	webhookUrl := app.Group(fmt.Sprintf("%v/webhook", ApiVersion))
	{
		webhookUrl.Post("/firs", webhookController.FirsWebhook)
	}

	invoiceUrlUnSec := app.Group(fmt.Sprintf("%v/zoho", ApiVersion), middleware.SelectSandboxDatabase(c.DB, c.TestDB))
	{
		invoiceUrlUnSec.Post("/webhook", webhookController.HandleZohoWebhook)
	}
}
