package v1

import (
	_ "einvoice-access-point/docs"
	"einvoice-access-point/pkg/config"
	"einvoice-access-point/pkg/database"
	"einvoice-access-point/pkg/middleware"
	"einvoice-access-point/pkg/utility"

	"github.com/go-playground/validator/v10"
	"github.com/goccy/go-json"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/swagger"
)

func Setup(logger *utility.Logger, validator *validator.Validate, db, testDb *database.Database, keys *utility.CryptoKeys) *fiber.App {

	/////////////////////////////////////////////
	//Initiate Fiber App
	/////////////////////////////////////////////
	r := fiber.New(fiber.Config{
		Prefork:                 false,
		AppName:                 "eInvoice Firs Backend",
		JSONEncoder:             json.Marshal,
		JSONDecoder:             json.Unmarshal,
		ServerHeader:            "Golang Fiber",
		EnableTrustedProxyCheck: true,
		BodyLimit:               3 << 20,
		ErrorHandler:            middleware.GlobalErrorHandler,
	})

	r.Use(recover.New(recover.Config{
		EnableStackTrace: false,
		StackTraceHandler: func(c *fiber.Ctx, e interface{}) {
			errMsg := "Unknown error"
			if err, ok := e.(error); ok {
				errMsg = err.Error()
			} else if msg, ok := e.(string); ok {
				errMsg = msg
			}

			rd := utility.BuildErrorResponse(fiber.StatusInternalServerError, "Internal Server Error", errMsg, nil, nil)

			_ = c.Status(fiber.StatusInternalServerError).JSON(rd)
		},
	}))

	r.Use(middleware.CORS())
	r.Use(middleware.Security())
	r.Use(middleware.Logger())
	r.Use(middleware.Metrics(config.GetConfig()))

	/////////////////////////////////////////////
	//General api route
	/////////////////////////////////////////////
	r.Get("/swagger/*", swagger.HandlerDefault)
	ApiVersion := "api/v1"
	/////////////////////////////////////////////
	//All Routes Registered
	/////////////////////////////////////////////

	HealthRoute(r, ApiVersion, validator, db, testDb, logger)
	AuthRoute(r, ApiVersion, validator, db, testDb, logger)
	SubscriptionRoute(r, ApiVersion, validator, db, testDb, logger)
	PluginRoute(r, ApiVersion, validator, db, testDb, logger)
	EntityRoute(r, ApiVersion, validator, db, testDb, logger)
	BusinessRoute(r, ApiVersion, validator, db, testDb, logger)
	CallbackRoute(r, ApiVersion, validator, db, testDb, logger)
	InvoiceRoute(r, ApiVersion, validator, db, testDb, logger, keys)
	RegisterBaseRoutes(r, ApiVersion)

	return r
}
