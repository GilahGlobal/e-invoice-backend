package repositories

import (
	"einvoice-access-point/internal/data/database"
	"einvoice-access-point/internal/data/entities"
	"errors"

	"gorm.io/gorm"
)

type AdminRepository interface {
	CreateAdmin(admin *entities.Admin, db database.DatabaseManager) error
	GetAdminByEmail(db database.DatabaseManager, email string) (*entities.Admin, error)
	GetAdminByID(db database.DatabaseManager, id string) (*entities.Admin, error)
	CountAdmins(db database.DatabaseManager) (int64, error)
}

type adminRepository struct{}

func NewAdminRepository() AdminRepository {
	return &adminRepository{}
}

func (r *adminRepository) CreateAdmin(admin *entities.Admin, db database.DatabaseManager) error {
	return db.CreateOneRecord(admin)
}

func (r *adminRepository) GetAdminByEmail(db database.DatabaseManager, email string) (*entities.Admin, error) {
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

func (r *adminRepository) GetAdminByID(db database.DatabaseManager, id string) (*entities.Admin, error) {
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

func (r *adminRepository) CountAdmins(db database.DatabaseManager) (int64, error) {
	var count int64
	err := db.DB().Model(&entities.Admin{}).Count(&count).Error
	if err != nil {
		return 0, err
	}
	return count, nil
}
