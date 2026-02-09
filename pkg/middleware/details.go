package middleware

import (
	"einvoice-access-point/pkg/database"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func GetUserDetails(c *fiber.Ctx) (*UserDataClaims, error) {
	claims, ok := c.Locals("userClaims").(*UserDataClaims)
	if !ok || claims == nil {
		return nil, fiber.NewError(fiber.StatusUnauthorized, "user claims not found")
	}

	if claims.ID == "" {
		return nil, fiber.NewError(fiber.StatusUnauthorized, "invalid user data in token")
	}

	return claims, nil
}

func GetDatabaseInstance(IsSandbox bool, prodDB, sandboxDB *database.Database) *gorm.DB {
	if IsSandbox {
		return sandboxDB.Postgresql.DB()
	}
	return prodDB.Postgresql.DB()
}
