package middleware

import (
	"einvoice-access-point/internal/config"
	"einvoice-access-point/internal/utility"
	"errors"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/limiter"
)

func Throttle() fiber.Handler {
	serverConfig := config.GetConfig().Server

	requestPerSecond := serverConfig.RequestPerSecond
	if requestPerSecond == 0 {
		requestPerSecond = 7
	}

	return limiter.New(limiter.Config{
		Max:        int(requestPerSecond),
		Expiration: time.Second,
		KeyGenerator: func(c *fiber.Ctx) string {
			return c.IP()
		},
		LimitReached: func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).JSON(utility.BuildErrorResponse(
				fiber.StatusTooManyRequests, "error", "Rate limit exceeded", "You have exceeded the allowed request limit. Try again later.", nil,
			))
		},
	})
}

func Security() fiber.Handler {
	return func(c *fiber.Ctx) error {
		c.Set("X-XSS-Protection", "1; mode=block")
		c.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")
		c.Set("X-Frame-Options", "SAMEORIGIN")
		c.Set("X-Content-Type-Options", "nosniff")
		//c.Set("Content-Security-Policy", "default-src 'self';")
		c.Set("Content-Security-Policy", "default-src 'self'; style-src 'self' 'unsafe-inline' https://fonts.googleapis.com; font-src 'self' https://fonts.gstatic.com; script-src 'self' 'unsafe-inline';")
		c.Set("X-Permitted-Cross-Domain-Policies", "none")
		c.Set("Referrer-Policy", "no-referrer")
		c.Set("Feature-Policy", "microphone 'none'; camera 'none'")

		return c.Next()
	}
}

func isExemptIP(ip string, exemptIPs []string) bool {
	for _, exemptIP := range exemptIPs {
		if ip == exemptIP {
			return true
		}
	}
	return false
}

func CheckAPIKey(c *fiber.Ctx) (string, error) {
	apiKey := c.Get("X-API-KEY")
	if apiKey == "" {
		return "", errors.New("api key not found")
	}
	return apiKey, nil
}
