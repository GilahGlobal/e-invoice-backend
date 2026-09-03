package seed

import (
	"einvoice-access-point/internal/data/database"
	"einvoice-access-point/internal/data/entities"
	"einvoice-access-point/internal/utility"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

func SeedRoles(db database.DatabaseManager) error {
	roles := []entities.Role{
		{
			ID:          uuid.New(),
			Name:        entities.RoleSuperAdmin,
			Description: "Super Administrator with full access",
		},
		{
			ID:          uuid.New(),
			Name:        entities.RoleAdmin,
			Description: "Administrator with limited access",
		},
	}

	for _, role := range roles {
		var count int64
		if err := db.DB().Model(&entities.Role{}).Where("name = ?", role.Name).Count(&count).Error; err != nil {
			return fmt.Errorf("failed to check existing role: %w", err)
		}
		if count == 0 {
			if err := db.CreateOneRecord(&role); err != nil {
				return fmt.Errorf("failed to create role %s: %w", role.Name, err)
			}
		}
	}

	return nil
}

func SeedSuperAdmin(db database.DatabaseManager) error {
	if err := SeedRoles(db); err != nil {
		return err
	}

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

	var superAdminRole entities.Role
	if err := db.DB().Where("name = ?", entities.RoleSuperAdmin).First(&superAdminRole).Error; err != nil {
		return fmt.Errorf("failed to find superadmin role: %w", err)
	}

	admin := entities.Admin{
		ID:       utility.GenerateUUID(),
		Name:     "Joel",
		Email:    email,
		Password: passwordHash,
		RoleID:   superAdminRole.ID,
	}

	err = db.CreateOneRecord(&admin)
	if err != nil {
		return fmt.Errorf("failed to create super admin: %w", err)
	}

	return nil
}
