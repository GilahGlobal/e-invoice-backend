package business

import (
	"crypto/sha256"
	"einvoice-access-point/pkg/database"
	"einvoice-access-point/pkg/models"
	"encoding/hex"
	"fmt"
)

func FindUserByID(db database.DatabaseManager, id string) (*models.Business, error) {
	var user models.Business
	err := db.DB().Where("id = ? AND acc_status = ?", id, 0).First(&user).Error
	if err != nil {
		return nil, err
	}

	user.APIKey.AfterFind(db.DB())

	return &user, nil
}

func GetUserByEmail(db database.DatabaseManager, userEmail string) (models.Business, error) {
	var user models.Business

	query := db.DB().Where("email = ?", userEmail)
	query = db.PreloadEntities(query, &user, "Invoices")

	if err := query.First(&user).Error; err != nil {
		return user, err
	}

	user.APIKey.AfterFind(db.DB())

	return user, nil
}

func FindUserByKey(db database.DatabaseManager, apiKey string) (*models.Business, error) {
	apiKeyHash := sha256.Sum256([]byte(apiKey))
	apiKeyHashStr := hex.EncodeToString(apiKeyHash[:])

	var user models.Business
	if err := db.DB().Where("api_key_hash = ? AND acc_status = ?", apiKeyHashStr, 0).First(&user).Error; err != nil {
		return nil, err
	}

	user.APIKey.AfterFind(db.DB())

	return &user, nil
}

func FindByEmailAndAPIKey(db database.DatabaseManager, username, apiKey string) (*models.Business, error) {
	apiKeyHash := sha256.Sum256([]byte(apiKey))
	apiKeyHashStr := hex.EncodeToString(apiKeyHash[:])

	var user models.Business
	err := db.DB().Where("email = ? AND api_key_hash = ? AND acc_status = ?", username, apiKeyHashStr, 0).First(&user).Error
	if err != nil {
		return nil, err
	}

	user.APIKey.AfterFind(db.DB())

	return &user, nil
}

func FindBusinessByPlatformOrgID(db database.DatabaseManager, platform, orgID string) (*models.Business, error) {
	var business models.Business
	fmt.Printf("plat: %s, org: %s\n", platform, orgID)
	err := db.DB().Debug().Raw(
		`SELECT * FROM "businesses" WHERE (platform_configs->?->>'org_id' = ? AND acc_status = ?) AND "businesses"."deleted_at" IS NULL ORDER BY "businesses"."id" LIMIT 1`,
		platform, orgID, 0,
	).Scan(&business).Error
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return nil, err
	}
	business.APIKey.AfterFind(db.DB())
	return &business, nil
}

func FindAllBusinesses(db database.DatabaseManager) ([]models.Business, error) {
	var businesses []models.Business
	query := db.DB().Where("acc_status = ?", 0)
	query = db.PreloadEntities(query, &models.Business{}, "Invoices")

	if err := query.Find(&businesses).Error; err != nil {
		return nil, err
	}

	for i := range businesses {
		if err := businesses[i].APIKey.AfterFind(db.DB()); err != nil {
			return nil, fmt.Errorf("failed to decrypt API key for business %s: %w", businesses[i].ID, err)
		}
	}

	return businesses, nil
}

func FindBusinessByID(db database.DatabaseManager, id string) (*models.Business, error) {
	var business models.Business
	query := db.DB().Where("id = ? AND acc_status = ?", id, 0)
	query = db.PreloadEntities(query, &models.Business{}, "Invoices")

	if err := query.First(&business).Error; err != nil {
		return nil, err
	}

	if err := business.APIKey.AfterFind(db.DB()); err != nil {
		return nil, fmt.Errorf("failed to decrypt API key for business %s: %w", business.ID, err)
	}

	return &business, nil
}
