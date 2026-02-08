package postgresql

import (
	"einvoice-access-point/pkg/config"
	"einvoice-access-point/pkg/database"
	"fmt"
	"log"
	"os"

	"einvoice-access-point/pkg/utility"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	lg "gorm.io/gorm/logger"
)

func ConnectToDatabase(logger *utility.Logger, configDatabases, testDatabase config.Database) (*gorm.DB, *gorm.DB) {
	// dbsCV := configDatabases

	// utility.LogAndPrint(logger, "connecting to database")
	// connectedDB := connectToDb(dbsCV.DB_HOST, dbsCV.USERNAME, dbsCV.PASSWORD, dbsCV.DB_NAME, dbsCV.DB_PORT, dbsCV.SSLMODE, dbsCV.TIMEZONE, logger)

	// utility.LogAndPrint(logger, "connected to database")
	// database.DB.Postgresql = NewPostgresqlConnection(connectedDB)

	testDB := testDatabase
	utility.LogAndPrint(logger, "connecting to test database")
	testConnectedDB := connectToDb(testDB.DB_HOST, testDB.USERNAME, testDB.PASSWORD, testDB.DB_NAME, testDB.DB_PORT, testDB.SSLMODE, testDB.TIMEZONE, logger)
	utility.LogAndPrint(logger, "connected to test database")

	database.TestDB.Postgresql = NewPostgresqlConnection(testConnectedDB)
	// return connectedDB, testConnectedDB
	return nil, testConnectedDB
}

func connectToDb(host, user, password, dbname, port, sslmode, timezone string, logger *utility.Logger) *gorm.DB {
	port = database.ResolvePortParsing(port, logger)
	sslRootCertPath := "ca.crt"
	dsn := fmt.Sprintf("host=%v user=%v password=%v dbname=%v port=%v sslmode=%v sslrootcert=%v TimeZone=%v", host, user, password, dbname, port, sslmode, sslRootCertPath, timezone)
	newLogger := lg.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags),
		lg.Config{
			LogLevel:                  lg.Error,
			IgnoreRecordNotFoundError: true,
			Colorful:                  true,
		},
	)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: newLogger,
	})
	if err != nil {
		utility.LogAndPrint(logger, fmt.Sprintf("connection to %v db failed with: %v", dbname, err))
		panic(err)

	}
	utility.LogAndPrint(logger, fmt.Sprintf("connected to %v db", dbname))
	// db = db.Debug() //database debug mode
	return db
}
