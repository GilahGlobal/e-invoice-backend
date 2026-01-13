package business

import (
	"einvoice-access-point/internal/dtos"
	repository "einvoice-access-point/internal/repository/business"
	inst "einvoice-access-point/pkg/dbinit"
	"einvoice-access-point/pkg/models"
	"errors"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func GetAllBusinesses(db *gorm.DB) ([]fiber.Map, error) {

	pdb := inst.InitDB(db, true)

	businesses, err := repository.FindAllBusinesses(pdb)
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
			"id":               business.ID,
			"email":            business.Email,
			"name":             business.Name,
			"business_id":      business.BusinessID,
			"service_id":       business.ServiceID,
			"platform_configs": cleanConfigs,
			"api_key":          string(business.APIKey),
			"invoices":         business.Invoices,
			"acc_status":       business.AccStatus,
			"created_at":       business.CreatedAt,
			"updated_at":       business.UpdatedAt,
		}
	}

	return response, nil
}

func GetBusinessByID(db *gorm.DB, id string) (fiber.Map, error) {
	pdb := inst.InitDB(db, true)

	business, err := repository.FindBusinessByID(pdb, id)
	if err != nil {
		return nil, err
	}

	cleanConfigs, err := business.PlatformConfigs.Decrypt()
	if err != nil {
		return nil, err
	}

	response := fiber.Map{
		"id":               business.ID,
		"email":            business.Email,
		"name":             business.Name,
		"business_id":      business.BusinessID,
		"service_id":       business.ServiceID,
		"platform_configs": cleanConfigs,
		"api_key":          string(business.APIKey),
		"invoices":         business.Invoices,
		"acc_status":       business.AccStatus,
		"created_at":       business.CreatedAt,
		"updated_at":       business.UpdatedAt,
	}

	return response, nil
}

func UpdateBusinessID(db *gorm.DB, id, businessID string) error {
	pdb := inst.InitDB(db, true)

	err := repository.UpdateNRSBusinessID(pdb, businessID, id)
	if err != nil {
		return err
	}
	return nil
}

func GetBusinessDetails(db *gorm.DB, id string) (*models.Business, error) {
	pdb := inst.InitDB(db, true)

	business := &models.Business{}
	_, err := pdb.SelectOneFromDb(business, "id = ?", id)
	if err != nil {
		return nil, err
	}

	if business.ID == "" {
		return nil, errors.New("business details not found")
	}
	return business, nil
}

func UpdateBusinessDetails(db *gorm.DB, business models.Business, payload dtos.UpdateBusinessDto) error {
	pdb := inst.InitDB(db, true)

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

	_, err := pdb.UpdateFields(business, updates, business.ID)

	if err != nil {
		return err
	}
	return nil
}
