package v1

import (
	"einvoice-access-point/internal/controller/entity"
	"einvoice-access-point/pkg/database"
	"einvoice-access-point/pkg/middleware"
	"einvoice-access-point/pkg/utility"
	"fmt"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
)

func EntityRoute(app *fiber.App, ApiVersion string, validator *validator.Validate, db, testDb *database.Database, logger *utility.Logger) *fiber.App {
	entityController := entity.Controller{Db: db, TestDB: testDb, Validator: validator, Logger: logger}

	// entityUrlSec := app.Group(fmt.Sprintf("%v/entity", ApiVersion), middleware.Authorize(db.Postgresql.DB(), testDb.Postgresql.DB()))
	entityUrlSec := app.Group(fmt.Sprintf("%v/entity", ApiVersion), middleware.Authorize(nil, testDb.Postgresql.DB()))
	{
		entityUrlSec.Get("", entityController.GetEntities)
		entityUrlSec.Get("/:entity_id", entityController.GetEntity)
		entityUrlSec.Post("/verify-tin", entityController.VerifyTin)
		entityUrlSec.Post("/vat-payment", entityController.PostVatPayment)
	}

	return app
}
