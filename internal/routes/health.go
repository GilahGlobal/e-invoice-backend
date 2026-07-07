package routes

import (
	"einvoice-access-point/internal/app/health"
	"einvoice-access-point/internal/core"
	"fmt"

	"github.com/gofiber/fiber/v2"
)

func HealthRoute(app *fiber.App, ApiVersion string, c *core.Container) *fiber.App {
	svc := health.NewService()
	healthController := health.NewHandler(svc, c.Validator, c.Logger, c.DB, c.TestDB)

	healthGroup := app.Group(fmt.Sprintf("%v", ApiVersion))
	healthGroup.Post("/health", healthController.Post)
	healthGroup.Get("/health", healthController.Get)
	healthGroup.Get("/health/firs", healthController.FirsHealthCheck)

	return app
}
