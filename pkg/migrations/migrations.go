package migrations

import (
	"einvoice-access-point/pkg/models"
)

func AuthMigrationModels() []interface{} {
	return []interface{}{
		&models.Business{},
		&models.Invoice{},
		&models.AccessToken{},
		&models.TokenManager{},
		&models.BulkUpload{},
		&models.SubscriptionPlan{},
		&models.Subscription{},
		&models.Transaction{},
		&models.SME{},
	}

}

func AlterColumnModels() []AlterColumn {
	return []AlterColumn{}
}
