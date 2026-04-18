package migrations

import (
	"einvoice-access-point/pkg/models"
)

func AuthMigrationModels() []interface{} {
	return []interface{}{
		&models.AggregatorInvitation{},
		&models.AggregatorActivityLog{},
		&models.Business{},
		&models.Invoice{},
		&models.SubscriptionPlan{},
		&models.Subscription{},
		&models.Transaction{},
		&models.AccessToken{},
		&models.TokenManager{},
		&models.BulkUpload{},
	}

}

func AlterColumnModels() []AlterColumn {
	return []AlterColumn{}
}
