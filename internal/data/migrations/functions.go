package migrations

import (
	"fmt"

	"gorm.io/gorm"
)

type AlterColumn struct {
	Model     interface{}
	TableName string
	Column    string
	Type      string
}

func (a *AlterColumn) UpdateColumnType(db *gorm.DB) error {
	if err := db.Exec(fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s TYPE %s USING %s::%s", a.TableName, a.Column, a.Type, a.Column, a.Type)).Error; err != nil {
		return err
	}

	if err := db.Migrator().AlterColumn(a.Model, a.Column); err != nil {
		return err
	}

	return nil
}
