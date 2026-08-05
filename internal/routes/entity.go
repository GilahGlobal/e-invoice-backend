package routes

import (
	"einvoice-access-point/internal/app/entity"
	"einvoice-access-point/internal/core"
	"einvoice-access-point/internal/middleware"
	"fmt"

	"github.com/gofiber/fiber/v2"
)

func EntityRoute(app *fiber.App, ApiVersion string, c *core.Container) {
	entityController := entity.NewHandler(c.Validator, c.Logger, c.DB, c.TestDB)

	entityUrlSec := app.Group(fmt.Sprintf("%v/entity", ApiVersion), middleware.Authorize(c.DB.Postgresql.DB(), c.TestDB.Postgresql.DB()))
	{
		entityUrlSec.Get("", entityController.GetEntities)
		entityUrlSec.Get("/:entity_id", entityController.GetEntity)
		entityUrlSec.Post("/verify-tin", entityController.VerifyTin)
		entityUrlSec.Post("/vat-payment", entityController.PostVatPayment)
	}
}
