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
