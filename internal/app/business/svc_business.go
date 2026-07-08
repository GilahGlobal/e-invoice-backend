package business

import (
	"einvoice-access-point/internal/data/dbinit"
	"einvoice-access-point/internal/data/entities"
	"errors"
	"strings"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

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
