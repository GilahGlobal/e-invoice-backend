package v1

import (
	"einvoice-access-point/internal/controller/business"
	"einvoice-access-point/pkg/database"
	"einvoice-access-point/pkg/middleware"
	"einvoice-access-point/pkg/utility"
	"fmt"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
)

func BusinessRoute(app *fiber.App, ApiVersion string, validator *validator.Validate, db *database.Database, logger *utility.Logger) *fiber.App {
	businessController := business.Controller{Db: db, Validator: validator, Logger: logger}

	businessUrlSec := app.Group(fmt.Sprintf("%v/business", ApiVersion), middleware.Authorize(db.Postgresql.DB()))
	{
		businessUrlSec.Get("", businessController.GetAllBusiness)
		businessUrlSec.Get("/:id", businessController.GetBusinessByID)
		businessUrlSec.Patch("/business-id/:id", businessController.UpdateBusinessID)
	}

	return app
}
