package middleware

import (
	"einvoice-access-point/internal/data/database"
	"einvoice-access-point/internal/data/entities"

	"github.com/gofiber/fiber/v2"
)

func ApiLogger(dbManager *database.Database) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Proceed with the request
		err := c.Next()

		// Skip logging for health checks or metrics if desired
		path := c.Path()
		if path == "/metrics" || path == "/health" {
			return err
		}

		// Save the log asynchronously
		go func() {
			db := dbManager.Postgresql.DB() // Default to production DB for logs
			
			logEntry := entities.ApiLog{
				Path:       path,
				Method:     c.Method(),
				StatusCode: c.Response().StatusCode(),
				ClientIP:   c.IP(),
				UserAgent:  string(c.Request().Header.UserAgent()),
			}

			// Attempt to insert the log silently
			_ = db.Create(&logEntry)
		}()

		return err
	}
}
