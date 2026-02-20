package database

import (
	"einvoice-access-point/pkg/config"
	"einvoice-access-point/pkg/utility"

	"gorm.io/gorm"
)

type DbConnection interface {
	NewDatabaseConnection(db *gorm.DB, logger *utility.Logger, config *config.Database) *Database
}

type Database struct {
	Postgresql DatabaseManager
	Redis      CacheManager
}

var DB *Database = &Database{}
var TestDB *Database = &Database{}

func Connection() (*Database, *Database) {
	return DB, TestDB
}
