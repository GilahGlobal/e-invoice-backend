package routes

import (
	"einvoice-access-point/internal/app/callback"
	"einvoice-access-point/internal/core"
	"fmt"

	"github.com/gofiber/fiber/v2"
)

func CallbackRoute(app *fiber.App, ApiVersion string, c *core.Container) {
	callController := callback.NewHandler(c.Validator, c.Logger, c.DB, c.TestDB)

	callUrlSec := app.Group(fmt.Sprintf("%v", ApiVersion))
	zoho := callUrlSec.Group("/zoho")
	{
		//zoho.Get("/callback", callController.ZohoCallback)
		zoho.Get("/auth", callController.ZohoAuthCode)
		zoho.Get("/auth/access-token", callController.ZohoGetAcessToken)
	}
}
