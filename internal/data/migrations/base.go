package migrations

import (
	"einvoice-access-point/internal/data/database"
	"fmt"

	"gorm.io/gorm"
)

func RunAllMigrations(db *database.Database) {

	MigrateModels(db.Postgresql.DB(), AuthMigrationModels(), AlterColumnModels())

}

func MigrateModels(db *gorm.DB, entities []interface{}, AlterColums []AlterColumn) {
	_ = db.AutoMigrate(entities...)

	for _, d := range AlterColums {
		err := d.UpdateColumnType(db)
		if err != nil {
			fmt.Println("error migrating ", d.TableName, "for column", d.Column, ": ", err)
		}

	}

}
