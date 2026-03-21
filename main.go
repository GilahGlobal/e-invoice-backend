// @title Gention E-invoicing Service API
// @version 1.0
// @description This is the e-invoicing service API documentation.
// @termsOfService http://swagger.io/terms/
// @BasePath /api/v1
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.

package main

import (
	"fmt"
	"log"
	"os"

	"github.com/go-playground/validator/v10"

	v1 "einvoice-access-point/api/v1"
	"einvoice-access-point/pkg/config"
	"einvoice-access-point/pkg/database"
	"einvoice-access-point/pkg/database/postgresql"
	"einvoice-access-point/pkg/migrations"
	"einvoice-access-point/pkg/utility"
)

func main() {

	logger := utility.NewLogger()
	if !logger.IsInitialized() {
		panic("Logger initialization failed: logger is nil")
	}

	configuration := config.Setup(logger, "./app")
	postgresql.ConnectToDatabase(logger, configuration.Database, configuration.TestDatabase)
	validatorRef := validator.New()
	validatorRef.RegisterValidation("nrsdate", utility.IsValidNRSDate)
	db, testDb := database.Connection()

	// Load crypto key from application onstart
	keys, err := utility.LoadCryptoKeys("crypto_keys.txt")
	if err != nil {
		utility.LogAndPrint(logger, fmt.Sprintf("Failed to load crypto keys: %v\n", err))
		os.Exit(1)
	}

	// Run migrations if enabled
	if configuration.Database.Migrate {
		migrations.RunAllMigrations(db)
		// seed.SeedDatabase(db)
	}

	if configuration.TestDatabase.Migrate {
		migrations.RunAllMigrations(testDb)
		// seed.SeedDatabase(testDb)
	}

	app := v1.Setup(logger, validatorRef, db, testDb, keys)

	host := os.Getenv("HOST")
	port := os.Getenv("PORT")

	if port == "" {
		port = configuration.Server.Port
	}
	if host == "" {
		host = "0.0.0.0"
	}

	utility.LogAndPrint(logger, fmt.Sprintf("Server is starting at %s:%s", host, port))
	log.Fatal(app.Listen(fmt.Sprintf("%s:%s", host, port)))
}
