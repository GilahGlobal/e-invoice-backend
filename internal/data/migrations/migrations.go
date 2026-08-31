package migrations

import "einvoice-access-point/internal/data/entities"

func AuthMigrationModels() []interface{} {
	return []interface{}{
		&entities.AggregatorInvitation{},
		&entities.AggregatorActivityLog{},
		&entities.Business{},
		&entities.Invoice{},
		&entities.SubscriptionPlan{},
		&entities.Subscription{},
		&entities.Transaction{},
		&entities.AccessToken{},
		&entities.TokenManager{},
		&entities.BulkUpload{},
		&entities.Admin{},
		&entities.ApiLog{},
		&entities.BusinessAggregatorHistory{},
	}

}

func AlterColumnModels() []AlterColumn {
	return []AlterColumn{}
}
