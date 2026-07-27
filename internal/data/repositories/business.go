package repositories

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"einvoice-access-point/internal/data/database"
	"einvoice-access-point/internal/data/entities"
	"einvoice-access-point/internal/utility"

	"gorm.io/gorm"
)

type BusinessRepository interface {
	FindUserByID(db database.DatabaseManager, id string) (*entities.Business, error)
	GetBusinessByIDForAggregator(db database.DatabaseManager, aggregatorID, businessID string) (*entities.Business, error)
	GetUserByEmail(db database.DatabaseManager, userEmail string) (entities.Business, error)
	FindUserByKey(db database.DatabaseManager, apiKey string) (*entities.Business, error)
	FindByEmailAndAPIKey(db database.DatabaseManager, username, apiKey string) (*entities.Business, error)
	FindBusinessByPlatformOrgID(db database.DatabaseManager, platform, orgID string) (*entities.Business, error)
	FindAllBusinesses(db database.DatabaseManager) ([]entities.Business, error)
	ListAllBusinesses(db database.DatabaseManager, search string, page, size int) ([]entities.Business, int64, error)
	GetSystemBusinessStats(db database.DatabaseManager) (int64, int64, error)
	FindBusinessByID(db database.DatabaseManager, id string) (*entities.Business, error)
	CreateBusiness(b *entities.Business, db database.DatabaseManager) error
	UpdateAUser(b *entities.Business, db database.DatabaseManager) error
	DeleteAUser(b *entities.Business, db database.DatabaseManager) error
	GenerateUniqueServiceID(db *gorm.DB) string
}

type businessRepository struct {
	prodDB database.DatabaseManager
	testDB database.DatabaseManager
}

func NewBusinessRepository(prodDB, testDB database.DatabaseManager) BusinessRepository {
	return &businessRepository{
		prodDB: prodDB,
		testDB: testDB,
	}
}

func (r *businessRepository) CreateBusiness(b *entities.Business, db database.DatabaseManager) error {
	err := db.CreateOneRecord(&b)
	if err != nil {
		return err
	}
	return nil
}

func (r *businessRepository) UpdateAUser(b *entities.Business, db database.DatabaseManager) error {
	_, err := db.SaveAllFields(&b)
	return err
}

func (r *businessRepository) DeleteAUser(b *entities.Business, db database.DatabaseManager) error {
	err := db.DeleteRecordFromDb(&b)

	if err != nil {
		return err
	}

	return nil
}

func (r *businessRepository) GenerateUniqueServiceID(db *gorm.DB) string {
	var existingCount int64
	serviceID := utility.GenerateRandomServiceID()

	for {
		db.Table("businesses").Where("service_id = ?", serviceID).Count(&existingCount)
		if existingCount == 0 {
			break
		}
		serviceID = utility.GenerateRandomServiceID()
	}

	return serviceID
}

func (r *businessRepository) FindUserByID(db database.DatabaseManager, id string) (*entities.Business, error) {
	var user entities.Business
	err := db.DB().Where("id = ? AND acc_status = ?", id, 0).First(&user).Error
	if err != nil {
		return nil, err
	}

	user.APIKey.AfterFind(db.DB())

	return &user, nil
}

func (r *businessRepository) GetUserByEmail(db database.DatabaseManager, userEmail string) (entities.Business, error) {
	var user entities.Business

	query := db.DB().Where("email = ?", userEmail)
	query = db.PreloadEntities(query, &user, "Invoices")

	if err := query.First(&user).Error; err != nil {
		return user, err
	}

	user.APIKey.AfterFind(db.DB())

	return user, nil
}

func (r *businessRepository) FindUserByKey(db database.DatabaseManager, apiKey string) (*entities.Business, error) {
	apiKeyHash := sha256.Sum256([]byte(apiKey))
	apiKeyHashStr := hex.EncodeToString(apiKeyHash[:])

	var user entities.Business
	if err := db.DB().Where("api_key_hash = ? AND acc_status = ?", apiKeyHashStr, 0).First(&user).Error; err != nil {
		return nil, err
	}

	user.APIKey.AfterFind(db.DB())

	return &user, nil
}

func (r *businessRepository) FindByEmailAndAPIKey(db database.DatabaseManager, username, apiKey string) (*entities.Business, error) {
	apiKeyHash := sha256.Sum256([]byte(apiKey))
	apiKeyHashStr := hex.EncodeToString(apiKeyHash[:])

	var user entities.Business
	err := db.DB().Where("email = ? AND api_key_hash = ? AND acc_status = ?", username, apiKeyHashStr, 0).First(&user).Error
	if err != nil {
		return nil, err
	}

	user.APIKey.AfterFind(db.DB())

	return &user, nil
}

func (r *businessRepository) FindBusinessByPlatformOrgID(db database.DatabaseManager, platform, orgID string) (*entities.Business, error) {
	var business entities.Business
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

func (r *businessRepository) FindAllBusinesses(db database.DatabaseManager) ([]entities.Business, error) {
	var businesses []entities.Business
	query := db.DB().Where("acc_status = ?", 0)
	query = db.PreloadEntities(query, &entities.Business{}, "Invoices")

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

func (r *businessRepository) ListAllBusinesses(db database.DatabaseManager, search string, page, size int) ([]entities.Business, int64, error) {
	var businesses []entities.Business
	var total int64

	query := db.DB().Where("acc_status = ? AND is_aggregator = ?", 0, false)

	if search != "" {
		searchPattern := "%" + strings.ToLower(search) + "%"
		query = query.Where("LOWER(name) LIKE ? OR LOWER(company_name) LIKE ? OR LOWER(email) LIKE ?", searchPattern, searchPattern, searchPattern)
	}

	if err := query.Model(&entities.Business{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * size
	if err := query.Offset(offset).Limit(size).Order("created_at DESC").Find(&businesses).Error; err != nil {
		return nil, 0, err
	}

	for i := range businesses {
		if err := businesses[i].APIKey.AfterFind(db.DB()); err != nil {
			return nil, 0, fmt.Errorf("failed to decrypt API key for business %s: %w", businesses[i].ID, err)
		}
	}

	return businesses, total, nil
}

func (r *businessRepository) FindBusinessByID(db database.DatabaseManager, id string) (*entities.Business, error) {
	var business entities.Business
	query := db.DB().Where("id = ? AND acc_status = ?", id, 0)
	query = db.PreloadEntities(query, &entities.Business{}, "Invoices")

	if err := query.First(&business).Error; err != nil {
		return nil, err
	}

	if err := business.APIKey.AfterFind(db.DB()); err != nil {
		return nil, fmt.Errorf("failed to decrypt API key for business %s: %w", business.ID, err)
	}

	return &business, nil
}

func (r *businessRepository) GetBusinessByIDForAggregator(db database.DatabaseManager, aggregatorID, businessID string) (*entities.Business, error) {
	var business entities.Business
	err := db.DB().Where("id = ? AND aggregator_id = ?", businessID, aggregatorID).First(&business).Error
	if err != nil {
		return nil, err
	}
	return &business, nil
}

func (r *businessRepository) GetSystemBusinessStats(db database.DatabaseManager) (int64, int64, error) {
	var totalBusinesses, totalAggregators int64

	if err := db.DB().Model(&entities.Business{}).Where("is_aggregator = ?", false).Count(&totalBusinesses).Error; err != nil {
		return 0, 0, err
	}

	if err := db.DB().Model(&entities.Business{}).Where("is_aggregator = ?", true).Count(&totalAggregators).Error; err != nil {
		return 0, 0, err
	}

	return totalBusinesses, totalAggregators, nil
}
