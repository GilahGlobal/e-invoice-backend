package v1

import (
	"einvoice-access-point/internal/controller/health"
	"einvoice-access-point/pkg/database"
	"einvoice-access-point/pkg/utility"
	"fmt"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
)

func HealthRoute(app *fiber.App, ApiVersion string, validator *validator.Validate, db, testDB *database.Database, logger *utility.Logger) *fiber.App {
	healthController := health.Controller{Db: db, Validator: validator, Logger: logger, TestDb: testDB}

	healthGroup := app.Group(fmt.Sprintf("%v", ApiVersion))
	healthGroup.Post("/health", healthController.Post)
	healthGroup.Get("/health", healthController.Get)
	healthGroup.Get("/health/firs", healthController.FirsHealthCheck)

	return app
}
