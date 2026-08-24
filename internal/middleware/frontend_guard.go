package middleware

import (
	"einvoice-access-point/internal/utility"

	"github.com/gofiber/fiber/v2"
)

// RequireFrontend restricts access to endpoints to only requests originating from the frontend.
// It checks for the presence of the 'client' header, which is expected to be sent by the frontend.
// If the header is missing, the request is assumed to be from an external client and is blocked.
func RequireFrontend() fiber.Handler {
	return func(c *fiber.Ctx) error {
		clientHeader := c.Get("client")

		// If the 'client' header is missing or empty, it's an external client
		if clientHeader == "" {
			rd := utility.BuildErrorResponse(
				fiber.StatusForbidden,
				"error",
				"Access Forbidden",
				"This endpoint is not available to external clients.",
				nil,
			)
			return c.Status(fiber.StatusForbidden).JSON(rd)
		}

		return c.Next()
	}
}
