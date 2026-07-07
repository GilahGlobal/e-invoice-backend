package routes

import (
	"fmt"

	"github.com/gofiber/fiber/v2"

	"github.com/gofiber/adaptor/v2"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func RegisterBaseRoutes(app *fiber.App, ApiVersion string) *fiber.App {

	baseGroup := app.Group(fmt.Sprintf("%v", ApiVersion))
	app.Get("/metrics", adaptor.HTTPHandler(promhttp.Handler()))

	baseGroup.Get("/", func(c *fiber.Ctx) error {
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"status_code": 200,
			"message":     "Home",
			"status":      "false",
		})
	})

	app.Use(func(c *fiber.Ctx) error {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"status_code": 404,
			"message":     "Page not found.",
			"status":      "false",
		})
	})

	return app

}
