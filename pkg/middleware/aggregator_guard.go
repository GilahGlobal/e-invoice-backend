package middleware

import (
	"einvoice-access-point/pkg/utility"

	"github.com/gofiber/fiber/v2"
)

// AggregatorGuard is a Fiber middleware that ensures the authenticated user
// is an aggregator (UserType == "aggregator"). Must be used AFTER Authorize.
func AggregatorGuard() fiber.Handler {
	return func(c *fiber.Ctx) error {
		claims, err := GetUserDetails(c)
		if err != nil {
			rd := utility.BuildErrorResponse(fiber.StatusUnauthorized, "error", "unable to get user claims", nil, nil)
			return c.Status(fiber.StatusUnauthorized).JSON(rd)
		}

		if !claims.IsAggregator {
			rd := utility.BuildErrorResponse(fiber.StatusForbidden, "error", "Access denied: aggregator account required", nil, nil)
			return c.Status(fiber.StatusForbidden).JSON(rd)
		}

		return c.Next()
	}
}
