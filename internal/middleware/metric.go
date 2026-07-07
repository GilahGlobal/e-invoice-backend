package middleware

import (
	"einvoice-access-point/internal/config"
	"log"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	httpRequestDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "http_request_duration_seconds",
		Help:    "Duration of HTTP requests in seconds",
		Buckets: prometheus.DefBuckets,
	}, []string{"method", "route", "status", "name", "interpreter"})
)

func init() {
	prometheus.MustRegister(httpRequestDuration)
}

func Metrics(cfg *config.Configuration) fiber.Handler {
	appName := cfg.App.Name
	if appName == "" {
		log.Println("Warning: APP_NAME is not set in configuration")
		appName = "unknown_app"
	}

	return func(c *fiber.Ctx) error {
		startTime := time.Now()

		method := c.Method()

		err := c.Next() // Process the request

		statusCode := c.Response().StatusCode()
		elapsed := time.Since(startTime).Seconds()

		httpRequestDuration.WithLabelValues(method, c.OriginalURL(), strconv.Itoa(statusCode), appName, "none").Observe(elapsed)

		return err
	}
}
