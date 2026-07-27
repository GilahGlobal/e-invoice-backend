package seed

import (
	"einvoice-access-point/internal/data/database"
	"einvoice-access-point/internal/data/entities"
	"einvoice-access-point/internal/utility"
	"fmt"
	"strings"
)

func SeedSuperAdmin(db database.DatabaseManager) error {
	email := strings.ToLower("joel@gention.tech")
	
	// Check if any admin exists
	var count int64
	err := db.DB().Model(&entities.Admin{}).Where("email = ?", email).Count(&count).Error
	if err != nil {
		return fmt.Errorf("failed to check existing admin: %w", err)
	}
	
	if count > 0 {
		return nil // Already seeded
	}

	passwordHash, err := utility.HashPassword("password123")
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	admin := entities.Admin{
		ID:       utility.GenerateUUID(),
		Name:     "Joel",
		Email:    email,
		Password: passwordHash,
		Role:     entities.RoleSuperAdmin,
	}

	err = db.CreateOneRecord(&admin)
	if err != nil {
		return fmt.Errorf("failed to create super admin: %w", err)
	}

	return nil
}
