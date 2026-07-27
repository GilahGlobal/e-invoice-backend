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

	"einvoice-access-point/internal/config"
	"einvoice-access-point/internal/data/database"
	"einvoice-access-point/internal/data/database/postgresql"
	"einvoice-access-point/internal/data/dbinit"
	"einvoice-access-point/internal/data/migrations"
	"einvoice-access-point/internal/data/seed"
	"einvoice-access-point/internal/routes"
	"einvoice-access-point/internal/utility"
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
	validatorRef.RegisterValidation("hsncode", utility.ValidateHSNCode)
	db, testDb := database.Connection()

	// Load crypto key from application onstart
	keys, err := utility.LoadCryptoKeys("crypto_keys.txt")
	if err != nil {
		utility.LogAndPrint(logger, fmt.Sprintf("Failed to load crypto keys: %v\n", err))
		os.Exit(1)
	}

	// Run migrations if enabled

	if configuration.TestDatabase.Migrate {
		migrations.RunAllMigrations(testDb)
		if err := seed.SeedSuperAdmin(dbinit.InitDB(testDb.Postgresql.DB(), false)); err != nil {
			utility.LogAndPrint(logger, fmt.Sprintf("Failed to seed sandbox super admin: %v\n", err))
		}
	}

	if configuration.Database.Migrate {
		migrations.RunAllMigrations(db)
		if err := seed.SeedSuperAdmin(dbinit.InitDB(db.Postgresql.DB(), false)); err != nil {
			utility.LogAndPrint(logger, fmt.Sprintf("Failed to seed production super admin: %v\n", err))
		}
	}

	app := routes.Setup(logger, validatorRef, db, testDb, keys)

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
