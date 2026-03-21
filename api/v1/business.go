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

func BusinessRoute(app *fiber.App, ApiVersion string, validator *validator.Validate, db, testDb *database.Database, logger *utility.Logger) *fiber.App {
	businessController := business.Controller{Db: db, TestDb: testDb, Validator: validator, Logger: logger}

	businessUrlSec := app.Group(fmt.Sprintf("%v/business", ApiVersion), middleware.Authorize(db.Postgresql.DB(), testDb.Postgresql.DB()))
	// businessUrlSec := app.Group(fmt.Sprintf("%v/business", ApiVersion), middleware.Authorize(nil, testDb.Postgresql.DB()))
	{
		// businessUrlSec.Get("", businessController.GetAllBusiness)
		businessUrlSec.Get("", businessController.GetBusiness)
		businessUrlSec.Patch("", businessController.UpdateBusinessProfile)
	}

	return app
}
