package repositories

import (
	"einvoice-access-point/internal/data/database"
	"einvoice-access-point/internal/data/entities"
	"errors"

	"gorm.io/gorm"
)

type AdminRepository struct{}

func NewAdminRepository() *AdminRepository {
	return &AdminRepository{}
}

func (r *AdminRepository) CreateAdmin(admin *entities.Admin, db database.DatabaseManager) error {
	return db.CreateOneRecord(admin)
}

func (r *AdminRepository) GetAdminByEmail(db database.DatabaseManager, email string) (*entities.Admin, error) {
	var admin entities.Admin
	err, nilErr := db.SelectOneFromDb(&admin, "email = ?", email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		return nil, err
	}
	if nilErr != nil {
		return nil, nilErr
	}
	return &admin, nil
}

func (r *AdminRepository) GetAdminByID(db database.DatabaseManager, id string) (*entities.Admin, error) {
	var admin entities.Admin
	err, nilErr := db.SelectOneFromDb(&admin, "id = ?", id)
	if err != nil {
		return nil, err
	}
	if nilErr != nil {
		return nil, nilErr
	}
	return &admin, nil
}

func (r *AdminRepository) CountAdmins(db database.DatabaseManager) (int64, error) {
	var count int64
	err := db.DB().Model(&entities.Admin{}).Count(&count).Error
	if err != nil {
		return 0, err
	}
	return count, nil
}
