package v1

import (
	"einvoice-access-point/internal/controller/callback"
	"einvoice-access-point/pkg/database"
	"einvoice-access-point/pkg/utility"
	"fmt"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
)

func CallbackRoute(app *fiber.App, ApiVersion string, validator *validator.Validate, db, testDb *database.Database, logger *utility.Logger) *fiber.App {
	callController := callback.Controller{Db: db, TestDB: testDb, Validator: validator, Logger: logger}

	callUrlSec := app.Group(fmt.Sprintf("%v", ApiVersion))
	{
		//callUrlSec.Get("/zoho/callback", callController.ZohoCallback)
		callUrlSec.Get("/zoho/auth", callController.ZohoAuthCode)
		callUrlSec.Get("/zoho/auth/access-token", callController.ZohoGetAcessToken)
	}

	return app
}
