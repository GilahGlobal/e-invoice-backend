package business

import (
	"einvoice-access-point/internal/common"
	"einvoice-access-point/internal/data/database"
	"einvoice-access-point/internal/data/dbinit"
	"einvoice-access-point/internal/data/entities"
	"einvoice-access-point/internal/data/repositories"
	"einvoice-access-point/internal/utility"
	"errors"
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type Service struct {
	repo *repositories.BusinessRepository
}

func NewService(repo *repositories.BusinessRepository) *Service {
	return &Service{
		repo: repo,
	}
}

func NewServiceWithDB(prodDB, testDB database.DatabaseManager) *Service {
	repo := repositories.NewBusinessRepository(prodDB, testDB)
	return NewService(repo)
}

func (s *Service) GetAllBusinesses(db *gorm.DB) ([]fiber.Map, error) {
	pdb := dbinit.InitDB(db, false)

	businesses, err := s.repo.FindAllBusinesses(pdb)
	if err != nil {
		return nil, err
	}

	response := make([]fiber.Map, len(businesses))
	for i, business := range businesses {
		cleanConfigs, err := business.PlatformConfigs.Decrypt()
		if err != nil {
			return nil, err
		}

		response[i] = fiber.Map{
			"id":                     business.ID,
			"email":                  business.Email,
			"name":                   business.Name,
			"business_id":            business.BusinessID,
			"service_id":             business.ServiceID,
			"platform_configs":       cleanConfigs,
			"api_key":                string(business.APIKey),
			"invoices":               business.Invoices,
			"acc_status":             business.AccStatus,
			"irn_signing_configured": strings.TrimSpace(string(business.IRNPublicKey)) != "" && strings.TrimSpace(string(business.IRNCertificate)) != "",
			"created_at":             business.CreatedAt,
			"updated_at":             business.UpdatedAt,
		}
	}

	return response, nil
}

func (s *Service) GetBusinessByID(db *gorm.DB, id string) (fiber.Map, error) {
	pdb := dbinit.InitDB(db, false)

	business, err := s.repo.FindBusinessByID(pdb, id)
	if err != nil {
		return nil, err
	}

	cleanConfigs, err := business.PlatformConfigs.Decrypt()
	if err != nil {
		return nil, err
	}

	response := fiber.Map{
		"id":                     business.ID,
		"email":                  business.Email,
		"name":                   business.Name,
		"tin":                    business.TIN,
		"phone_number":           business.PhoneNumber,
		"company_name":           business.CompanyName,
		"business_id":            business.BusinessID,
		"service_id":             business.ServiceID,
		"platform_configs":       cleanConfigs,
		"api_key":                string(business.APIKey),
		"invoices":               business.Invoices,
		"acc_status":             business.AccStatus,
		"irn_signing_configured": strings.TrimSpace(string(business.IRNPublicKey)) != "" && strings.TrimSpace(string(business.IRNCertificate)) != "",
		"created_at":             business.CreatedAt,
		"updated_at":             business.UpdatedAt,
	}

	return response, nil
}

func (s *Service) GetBusinessDetails(db *gorm.DB, id string) (*entities.Business, error) {
	pdb := dbinit.InitDB(db, false)

	business := &entities.Business{}
	_, err := pdb.SelectOneFromDb(business, "id = ?", id)
	if err != nil {
		return nil, err
	}

	if business.ID == "" {
		return nil, errors.New("business details not found")
	}
	return business, nil
}

func (s *Service) UpdateBusinessDetails(db *gorm.DB, business entities.Business, payload UpdateBusinessDto) error {
	pdb := dbinit.InitDB(db, false)

	updates := make(map[string]interface{})

	if payload.Name != nil {
		updates["name"] = *payload.Name
	}
	if payload.Email != nil {
		updates["email"] = *payload.Email
	}
	if payload.PhoneNumber != nil {
		updates["phone_number"] = *payload.PhoneNumber
	}
	if payload.CompanyName != nil {
		updates["company_name"] = *payload.CompanyName
	}
	if payload.BusinessID != nil {
		updates["business_id"] = *payload.BusinessID
	}
	if payload.ServiceID != nil {
		updates["service_id"] = *payload.ServiceID
	}

	_, err := pdb.UpdateFields(business, updates, business.ID)
	if err != nil {
		return err
	}
	return nil
}

func (s *Service) GetBusinessByPlatformOrgID(db *gorm.DB, platform, orgID string) (*entities.Business, error) {
	pdb := dbinit.InitDB(db, false)
	return s.repo.FindBusinessByPlatformOrgID(pdb, platform, orgID)
}

type InvoiceUploadSetup struct {
	BusinessID        string
	ServiceID         string
	BmpUploadSelected bool
}

func (s *Service) SaveBusinessIRNSigningKeys(db *gorm.DB, id string, fileContent []byte) error {
	document, err := utility.ParseCryptoKeyDocument(fileContent)
	if err != nil {
		return err
	}

	if _, err := utility.NewCryptoKeys(document.PublicKey, document.Certificate); err != nil {
		return fmt.Errorf("invalid crypto keys document: %w", err)
	}

	business, err := s.GetBusinessDetails(db, id)
	if err != nil {
		return err
	}

	business.IRNPublicKey = common.EncryptedString(document.PublicKey)
	business.IRNCertificate = common.EncryptedString(document.Certificate)
	business.KeysSet = true

	pdb := dbinit.InitDB(db, false)
	if _, err := pdb.SaveAllFields(business); err != nil {
		return fmt.Errorf("failed to save business IRN signing keys: %w", err)
	}

	return nil
}

func (s *Service) ResolveBusinessIRNSigningKeys(db *gorm.DB, id string, isSandbox bool, fallbackKeys *utility.CryptoKeys) (*utility.CryptoKeys, error) {
	if isSandbox {
		if fallbackKeys != nil {
			return fallbackKeys, nil
		}

		return utility.LoadCryptoKeys("crypto_keys.txt")
	}

	business, err := s.GetBusinessDetails(db, id)
	if err != nil {
		return nil, err
	}

	publicKey := strings.TrimSpace(string(business.IRNPublicKey))
	certificate := strings.TrimSpace(string(business.IRNCertificate))
	if publicKey == "" || certificate == "" {
		return nil, errors.New("business IRN signing keys have not been configured")
	}

	keys, err := utility.NewCryptoKeys(publicKey, certificate)
	if err != nil {
		return nil, fmt.Errorf("failed to parse saved business IRN signing keys: %w", err)
	}

	return keys, nil
}

func (s *Service) ValidateInvoiceUploadSetup(db *gorm.DB, ownerID string) (*InvoiceUploadSetup, error) {
	business, err := s.GetBusinessDetails(db, ownerID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve business details: %w", err)
	}

	missing := make([]string, 0, 3)

	businessID := ""
	if business.BusinessID != nil {
		businessID = strings.TrimSpace(*business.BusinessID)
	}
	if businessID == "" {
		missing = append(missing, "business id")
	}

	serviceID := ""
	if business.ServiceID != nil {
		serviceID = strings.TrimSpace(*business.ServiceID)
	}
	if serviceID == "" {
		missing = append(missing, "service id")
	}

	irnPublicKey := strings.TrimSpace(string(business.IRNPublicKey))
	irnCertificate := strings.TrimSpace(string(business.IRNCertificate))
	if !business.KeysSet || irnPublicKey == "" || irnCertificate == "" {
		missing = append(missing, "crypto keys")
	}

	if len(missing) > 0 {
		return nil, fmt.Errorf("cannot upload invoice: missing required setup: %s", strings.Join(missing, ", "))
	}

	return &InvoiceUploadSetup{
		BusinessID:        businessID,
		ServiceID:         serviceID,
		BmpUploadSelected: business.BmpUploadSelected,
	}, nil
}
