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

type BusinessRepository struct {
	prodDB database.DatabaseManager
	testDB database.DatabaseManager
}

func NewBusinessRepository(prodDB, testDB database.DatabaseManager) *BusinessRepository {
	return &BusinessRepository{
		prodDB: prodDB,
		testDB: testDB,
	}
}

func (r *BusinessRepository) CreateBusiness(b *entities.Business, db database.DatabaseManager) error {
	err := db.CreateOneRecord(&b)
	if err != nil {
		return err
	}
	return nil
}

func (r *BusinessRepository) UpdateAUser(b *entities.Business, db database.DatabaseManager) error {
	_, err := db.SaveAllFields(&b)
	return err
}

func (r *BusinessRepository) DeleteAUser(b *entities.Business, db database.DatabaseManager) error {
	err := db.DeleteRecordFromDb(&b)

	if err != nil {
		return err
	}

	return nil
}

func (r *BusinessRepository) GenerateUniqueServiceID(db *gorm.DB) string {
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

func (r *BusinessRepository) FindUserByID(db database.DatabaseManager, id string) (*entities.Business, error) {
	var user entities.Business
	err := db.DB().Where("id = ? AND acc_status = ?", id, 0).First(&user).Error
	if err != nil {
		return nil, err
	}

	user.APIKey.AfterFind(db.DB())

	return &user, nil
}

func (r *BusinessRepository) GetUserByEmail(db database.DatabaseManager, userEmail string) (entities.Business, error) {
	var user entities.Business

	query := db.DB().Where("email = ?", userEmail)
	query = db.PreloadEntities(query, &user, "Invoices")

	if err := query.First(&user).Error; err != nil {
		return user, err
	}

	user.APIKey.AfterFind(db.DB())

	return user, nil
}

func (r *BusinessRepository) CheckBusinessExistsByEmailOrCompanyName(db database.DatabaseManager, email, companyName string) (bool, error) {
	var count int64
	err := db.DB().Model(&entities.Business{}).
		Where("LOWER(email) = ? OR LOWER(company_name) = ?", strings.ToLower(email), strings.ToLower(companyName)).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *BusinessRepository) FindUserByKey(db database.DatabaseManager, apiKey string) (*entities.Business, error) {
	apiKeyHash := sha256.Sum256([]byte(apiKey))
	apiKeyHashStr := hex.EncodeToString(apiKeyHash[:])

	var user entities.Business
	if err := db.DB().Where("api_key_hash = ? AND acc_status = ?", apiKeyHashStr, 0).First(&user).Error; err != nil {
		return nil, err
	}

	user.APIKey.AfterFind(db.DB())

	return &user, nil
}

func (r *BusinessRepository) FindByEmailAndAPIKey(db database.DatabaseManager, username, apiKey string) (*entities.Business, error) {
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

func (r *BusinessRepository) FindBusinessByPlatformOrgID(db database.DatabaseManager, platform, orgID string) (*entities.Business, error) {
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

func (r *BusinessRepository) FindAllBusinesses(db database.DatabaseManager) ([]entities.Business, error) {
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

type AdminBusinessQueryResult struct {
	entities.Business
	TotalInvoicesUploaded int64
	LastInvoiceUploadedAt *string
	SubscribedPlan        string
}

func (r *BusinessRepository) ListAllBusinesses(db database.DatabaseManager, search string, page, size int) ([]AdminBusinessQueryResult, int64, error) {
	var businesses []AdminBusinessQueryResult
	var total int64

	// Base query for businesses
	baseQuery := db.DB().Model(&entities.Business{}).Where("is_aggregator = ?", false)

	if search != "" {
		searchPattern := "%" + strings.ToLower(search) + "%"
		baseQuery = baseQuery.Where("LOWER(name) LIKE ? OR LOWER(company_name) LIKE ? OR LOWER(email) LIKE ?", searchPattern, searchPattern, searchPattern)
	}

	if err := baseQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * size

	// Join with invoices for counts and last upload, and transactions for subscribed plan
	query := db.DB().Table("businesses").
		Select(`businesses.*, 
			(SELECT COUNT(*) FROM invoices WHERE invoices.business_id = businesses.id) as total_invoices_uploaded,
			(SELECT MAX(created_at) FROM invoices WHERE invoices.business_id = businesses.id) as last_invoice_uploaded_at,
			(SELECT plan FROM subscriptions WHERE subscriptions.business_id = businesses.id AND subscriptions.is_active = true ORDER BY created_at DESC LIMIT 1) as subscribed_plan`).
		Where("is_aggregator = ?", false)

	if search != "" {
		searchPattern := "%" + strings.ToLower(search) + "%"
		query = query.Where("LOWER(businesses.name) LIKE ? OR LOWER(businesses.company_name) LIKE ? OR LOWER(businesses.email) LIKE ?", searchPattern, searchPattern, searchPattern)
	}

	if err := query.Offset(offset).Limit(size).Order("businesses.created_at DESC").Scan(&businesses).Error; err != nil {
		return nil, 0, err
	}

	return businesses, total, nil
}

func (r *BusinessRepository) FindBusinessByID(db database.DatabaseManager, id string) (*entities.Business, error) {
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

func (r *BusinessRepository) GetBusinessByIDForAdmin(db database.DatabaseManager, id string) (*entities.Business, error) {
	var business entities.Business
	query := db.DB().Where("id = ? AND acc_status = ?", id, 0)
	query = db.PreloadEntities(query, &entities.Business{}, "Invoices")

	if err := query.First(&business).Error; err != nil {
		return nil, err
	}

	return &business, nil
}

func (r *BusinessRepository) GetBusinessByIDForAggregator(db database.DatabaseManager, aggregatorID, businessID string) (*entities.Business, error) {
	var business entities.Business
	err := db.DB().Where("id = ? AND aggregator_id = ?", businessID, aggregatorID).First(&business).Error
	if err != nil {
		return nil, err
	}
	return &business, nil
}

func (r *BusinessRepository) GetSystemBusinessStats(db database.DatabaseManager) (int64, int64, error) {
	var totalBusinesses, totalAggregators int64

	if err := db.DB().Model(&entities.Business{}).Where("is_aggregator = ?", false).Count(&totalBusinesses).Error; err != nil {
		return 0, 0, err
	}

	if err := db.DB().Model(&entities.Business{}).Where("is_aggregator = ?", true).Count(&totalAggregators).Error; err != nil {
		return 0, 0, err
	}

	return totalBusinesses, totalAggregators, nil
}

func (r *BusinessRepository) CreateBusinessAggregatorHistory(db database.DatabaseManager, history *entities.BusinessAggregatorHistory) error {
	return db.CreateOneRecord(history)
}

func (r *BusinessRepository) GetBusinessAggregatorHistory(db database.DatabaseManager, businessID string) ([]entities.BusinessAggregatorHistory, error) {
	var histories []entities.BusinessAggregatorHistory
	err := db.DB().Where("business_id = ?", businessID).Order("created_at DESC").Find(&histories).Error
	return histories, err
}
