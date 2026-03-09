package sme

import (
	"einvoice-access-point/pkg/database"
	"einvoice-access-point/pkg/models"
	"errors"

	"gorm.io/gorm"
)

func FindSMEByAggregatorId(db database.DatabaseManager, aggregatorId, email string) (*models.SME, error) {
	var sme models.SME
	err := db.DB().
		Model(&models.SME{}).
		Where("aggregator_id = ? AND email = ?", aggregatorId, email).
		First(&sme).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}

	return &sme, nil
}

func FindSmeByID(db database.DatabaseManager, id string) (*models.SME, error) {
	var sme models.SME
	err := db.DB().Where("id = ?", id).First(&sme).Error
	if err != nil {
		return nil, err
	}

	return &sme, nil
}

func CreateSme(b *models.SME, db database.DatabaseManager) error {

	err := db.CreateOneRecord(&b)
	if err != nil {
		return err
	}
	return nil
}
