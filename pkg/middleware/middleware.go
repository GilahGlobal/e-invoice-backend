package middleware

import (
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/sirupsen/logrus"
)

func CORS() fiber.Handler {
	return func(c *fiber.Ctx) error {
		origin := c.Get("Origin")

		if origin == "" {
			referrer := strings.TrimRight(c.Get("Referer"), "/")

			if referrer != "" {
				if u, err := url.Parse(referrer); err == nil {
					origin = u.Scheme + "://" + u.Host
				}
			}
		}
		if origin != "" {
			c.Set("Access-Control-Allow-Origin", origin)
			c.Set("Access-Control-Allow-Credentials", "true")
			c.Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, Accept, Origin, Cache-Control, X-Requested-With, Referer")
			c.Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE, PATCH")
		}

		if c.Method() == fiber.MethodOptions {
			return c.SendStatus(fiber.StatusNoContent)
		}

		return c.Next()
	}
}

func Logger() fiber.Handler {

	logger := logrus.New()
	logger.Out = os.Stdout
	logger.SetFormatter(&logrus.TextFormatter{})
	logger.SetLevel(logrus.DebugLevel)

	return func(c *fiber.Ctx) error {
		baseRoute := "/"
		if c.OriginalURL() == baseRoute {
			return c.Next()
		}

		startTime := time.Now()
		err := c.Next()
		endTime := time.Now()
		latencyTime := endTime.Sub(startTime)
		reqMethod := c.Method()
		reqURI := c.OriginalURL()
		statusCode := c.Response().StatusCode()
		clientIP := c.IP()
		userIdentifier := "-"
		userID := "-"
		currentTime := time.Now().Format("02/Jan/2006:15:04:05 -0700")

		responseSize := len(c.Response().Body())

		logEntry := fmt.Sprintf("%s %s %s [%s] \"%s %s HTTP/1.0\" %d %d %v",
			clientIP, userIdentifier, userID, currentTime, reqMethod, reqURI, statusCode, responseSize, latencyTime)

		logger.Log(logrus.DebugLevel, logEntry)

		return err
	}
}
