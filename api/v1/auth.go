package v1

import (
	"einvoice-access-point/internal/controller/auth"
	"einvoice-access-point/pkg/database"
	"einvoice-access-point/pkg/middleware"
	"einvoice-access-point/pkg/utility"
	"fmt"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
)

func AuthRoute(app *fiber.App, ApiVersion string, validator *validator.Validate, db *database.Database, logger *utility.Logger) *fiber.App {
	authController := auth.Controller{Db: db, Validator: validator, Logger: logger}

	authGroup := app.Group(fmt.Sprintf("%v/auth", ApiVersion))
	authGroup.Post("/login", authController.Login)
	authGroup.Post("/register", authController.Register)
	authGroup.Post("/initiate-forgot-password", authController.InitiateForgotPassword)
	authGroup.Post("/complete-forgot-password", authController.CompleteForgotPassword)

	authUrlSec := app.Group(fmt.Sprintf("%v/auth", ApiVersion), middleware.Authorize(db.Postgresql.DB()))
	{
		authUrlSec.Get("/logout", authController.Logout)
		authUrlSec.Post("/register1", authController.Register)
	}

	return app
}
