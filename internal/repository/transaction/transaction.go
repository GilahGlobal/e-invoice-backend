package transaction

import (
	"errors"

	"einvoice-access-point/pkg/database"
	"einvoice-access-point/pkg/models"

	"gorm.io/gorm"
)

func CreateTransaction(record *models.Transaction, db database.DatabaseManager) error {
	return db.DB().Create(record).Error
}

func GetTransactionByReference(reference string, db database.DatabaseManager) (*models.Transaction, error) {
	var transaction models.Transaction
	err := db.DB().Where("reference = ?", reference).First(&transaction).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &transaction, nil
}

func SaveTransaction(record *models.Transaction, db database.DatabaseManager) error {
	return db.DB().Save(record).Error
}
