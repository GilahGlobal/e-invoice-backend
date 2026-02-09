package middleware

import (
	authRepo "einvoice-access-point/internal/repository/auth"
	"einvoice-access-point/pkg/config"
	"einvoice-access-point/pkg/database/postgresql"
	inst "einvoice-access-point/pkg/dbinit"
	"einvoice-access-point/pkg/utility"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v4"
	"gorm.io/gorm"
)

func Authorize(db, testDB *gorm.DB) fiber.Handler {
	var pgd *postgresql.Postgresql
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

		if claims.IsSandbox {
			pgd = inst.InitDB(testDB, true)
		} else {
			pgd = inst.InitDB(db, true)
		}

		accessToken, err := authRepo.GetByID(claims.AccessUuid, pgd)
		if err != nil {
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
