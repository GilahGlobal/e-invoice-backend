package business

import (
	repository "einvoice-access-point/internal/repository/business"
	inst "einvoice-access-point/pkg/dbinit"
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

	business, err := repository.FindBusinessByID(pdb, id)
	if err != nil {
		return err
	}

	if business == nil {
		return errors.New("business account not found")
	}

	business.BusinessID = businessID

	err = repository.UpdateAUser(business, pdb)

	if err != nil {
		return err
	}
	return nil
}
