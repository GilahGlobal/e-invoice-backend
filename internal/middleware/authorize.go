package middleware

import (
	"einvoice-access-point/internal/config"
	"einvoice-access-point/internal/data/dbinit"
	"einvoice-access-point/internal/data/entities"
	"einvoice-access-point/internal/utility"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v4"
	"gorm.io/gorm"
)

func Authorize(db, testDB *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {

		configs := config.GetConfig()

		authHeader := c.Get("Authorization")

		if authHeader == "" {
			rd := utility.BuildErrorResponse(fiber.StatusUnauthorized, "error", "Missing Authorization header", "Unauthorized", nil)
			return c.Status(fiber.StatusUnauthorized).JSON(rd)
		}

		val := strings.Split(authHeader, " ")
		if len(val) < 2 || strings.ToLower(val[0]) != "bearer" {
			rd := utility.BuildErrorResponse(fiber.StatusUnauthorized, "error", "Invalid Authorization format", "Unauthorized", nil)
			return c.Status(fiber.StatusUnauthorized).JSON(rd)
		}

		tokenVal := val[1]

		token, err := jwt.ParseWithClaims(
			tokenVal,
			&UserDataClaims{},
			func(*jwt.Token) (interface{}, error) {
				return []byte(configs.Server.Secret), nil
			})

		if err != nil {
			rd := utility.BuildErrorResponse(fiber.StatusUnauthorized, "error", "Missing token", "Unauthorized", nil)
			return c.Status(fiber.StatusUnauthorized).JSON(rd)
		}

		claims, ok := token.Claims.(*UserDataClaims)
		if claims.Name == "" || !ok {
			rd := utility.BuildErrorResponse(fiber.StatusUnauthorized, "error", invalidUser, "Unauthorized", nil)
			return c.Status(fiber.StatusUnauthorized).JSON(rd)

		}

		selectedDB := dbinit.InitDB(db, false)
		if claims.IsSandbox {
			selectedDB = dbinit.InitDB(testDB, false)
		}

		var accessToken entities.AccessToken
		if err := selectedDB.DB().Where("id = ?", claims.AccessUuid).First(&accessToken).Error; err != nil {
			rd := utility.BuildErrorResponse(fiber.StatusUnauthorized, "error", "Token is invalid", "Unauthorized", nil)
			return c.Status(fiber.StatusUnauthorized).JSON(rd)
		}

		if accessToken.LoginAccessToken != tokenVal || claims.ID != accessToken.OwnerID || !accessToken.IsLive {
			rd := utility.BuildErrorResponse(fiber.StatusUnauthorized, "error", "Session is invalid!", "Unauthorized", nil)
			return c.Status(fiber.StatusUnauthorized).JSON(rd)
		}

		c.Locals("userClaims", claims)
		return c.Next()

	}

}

func AuthorizeAdmin(db, testDB *gorm.DB, requiredRoles ...string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		configs := config.GetConfig()

		authHeader := c.Get("Authorization")
		if authHeader == "" {
			rd := utility.BuildErrorResponse(fiber.StatusUnauthorized, "error", "Missing Authorization header", "Unauthorized", nil)
			return c.Status(fiber.StatusUnauthorized).JSON(rd)
		}

		val := strings.Split(authHeader, " ")
		if len(val) < 2 || strings.ToLower(val[0]) != "bearer" {
			rd := utility.BuildErrorResponse(fiber.StatusUnauthorized, "error", "Invalid Authorization format", "Unauthorized", nil)
			return c.Status(fiber.StatusUnauthorized).JSON(rd)
		}

		tokenVal := val[1]

		token, err := jwt.ParseWithClaims(
			tokenVal,
			&AdminDataClaims{},
			func(*jwt.Token) (interface{}, error) {
				return []byte(configs.Server.Secret), nil
			})

		if err != nil {
			rd := utility.BuildErrorResponse(fiber.StatusUnauthorized, "error", "Missing token or invalid claims", "Unauthorized", nil)
			return c.Status(fiber.StatusUnauthorized).JSON(rd)
		}

		claims, ok := token.Claims.(*AdminDataClaims)
		if claims.Email == "" || !ok {
			rd := utility.BuildErrorResponse(fiber.StatusUnauthorized, "error", "Invalid admin claims", "Unauthorized", nil)
			return c.Status(fiber.StatusUnauthorized).JSON(rd)
		}

		selectedDB := dbinit.InitDB(db, false)
		if claims.IsSandbox {
			selectedDB = dbinit.InitDB(testDB, false)
		}

		// Check roles
		if len(requiredRoles) > 0 {
			var role entities.Role
			if err := selectedDB.DB().Where("id = ?", claims.RoleID).First(&role).Error; err != nil {
				rd := utility.BuildErrorResponse(fiber.StatusForbidden, "error", "Forbidden: Role not found", "Forbidden", nil)
				return c.Status(fiber.StatusForbidden).JSON(rd)
			}

			hasRole := false
			for _, requiredRole := range requiredRoles {
				if string(requiredRole) == role.Name {
					hasRole = true
					break
				}
			}

			if !hasRole {
				rd := utility.BuildErrorResponse(fiber.StatusForbidden, "error", "Forbidden: Insufficient privileges", "Forbidden", nil)
				return c.Status(fiber.StatusForbidden).JSON(rd)
			}
		}

		var accessToken entities.AccessToken
		if err := selectedDB.DB().Where("id = ?", claims.AccessUuid).First(&accessToken).Error; err != nil {
			rd := utility.BuildErrorResponse(fiber.StatusUnauthorized, "error", "Token is invalid", "Unauthorized", nil)
			return c.Status(fiber.StatusUnauthorized).JSON(rd)
		}

		if accessToken.LoginAccessToken != tokenVal || claims.ID != accessToken.OwnerID || !accessToken.IsLive {
			rd := utility.BuildErrorResponse(fiber.StatusUnauthorized, "error", "Session is invalid!", "Unauthorized", nil)
			return c.Status(fiber.StatusUnauthorized).JSON(rd)
		}

		c.Locals("adminClaims", claims)
		return c.Next()
	}
}

// RequireAdminRoles restricts a route to the given admin roles.
// It must run after AuthorizeAdmin and SelectDatabaseFromClaims, as it relies on the
// admin claims and selected database they place in the request context.
func RequireAdminRoles(allowedRoles ...string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		claims, ok := c.Locals("adminClaims").(*AdminDataClaims)
		if !ok || claims == nil {
			rd := utility.BuildErrorResponse(fiber.StatusUnauthorized, "error", "Invalid admin claims", "Unauthorized", nil)
			return c.Status(fiber.StatusUnauthorized).JSON(rd)
		}

		rawDb, err := GetDatabase(c)
		if err != nil {
			rd := utility.BuildErrorResponse(fiber.StatusInternalServerError, "error", err.Error(), "Internal server error", nil)
			return c.Status(fiber.StatusInternalServerError).JSON(rd)
		}

		var role entities.Role
		if err := rawDb.Where("id = ?", claims.RoleID).First(&role).Error; err != nil {
			rd := utility.BuildErrorResponse(fiber.StatusForbidden, "error", "Forbidden: Role not found", "Forbidden", nil)
			return c.Status(fiber.StatusForbidden).JSON(rd)
		}

		for _, allowed := range allowedRoles {
			if role.Name == allowed {
				return c.Next()
			}
		}

		rd := utility.BuildErrorResponse(fiber.StatusForbidden, "error", "Forbidden: Insufficient privileges", "Forbidden", nil)
		return c.Status(fiber.StatusForbidden).JSON(rd)
	}
}
