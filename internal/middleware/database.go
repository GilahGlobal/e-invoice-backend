package middleware

import (
	"encoding/json"
	"fmt"
	"strings"

	"einvoice-access-point/internal/data/database"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

const selectedDBKey = "selected_db"

func SelectDatabaseFromClaims(prodDB, sandboxDB *database.Database) fiber.Handler {
	return func(c *fiber.Ctx) error {
		claims, ok := c.Locals("userClaims").(*UserDataClaims)
		if !ok || claims == nil {
			return fiber.NewError(fiber.StatusUnauthorized, "user claims not found")
		}

		return setSelectedDatabase(c, claims.IsSandbox, prodDB, sandboxDB)
	}
}

func SelectDatabaseFromQuery(queryKey string, prodDB, sandboxDB *database.Database) fiber.Handler {
	return func(c *fiber.Ctx) error {
		raw := strings.TrimSpace(c.Query(queryKey))
		if raw == "" {
			return fiber.NewError(fiber.StatusBadRequest, fmt.Sprintf("%s is required", queryKey))
		}

		useSandbox, ok := parseBool(raw)
		if !ok {
			return fiber.NewError(fiber.StatusBadRequest, fmt.Sprintf("invalid %s value", queryKey))
		}

		return setSelectedDatabase(c, useSandbox, prodDB, sandboxDB)
	}
}

func SelectDatabaseFromJSONPath(path string, prodDB, sandboxDB *database.Database) fiber.Handler {
	return func(c *fiber.Ctx) error {
		useSandbox, ok := lookupBoolFromJSONPath(c.Body(), path)
		if !ok {
			return fiber.NewError(fiber.StatusBadRequest, fmt.Sprintf("%s is required", path))
		}

		return setSelectedDatabase(c, useSandbox, prodDB, sandboxDB)
	}
}

func SelectSandboxDatabase(prodDB, sandboxDB *database.Database) fiber.Handler {
	return func(c *fiber.Ctx) error {
		return setSelectedDatabase(c, true, prodDB, sandboxDB)
	}
}

func GetDatabase(c *fiber.Ctx) (*gorm.DB, error) {
	db, ok := c.Locals(selectedDBKey).(*gorm.DB)
	if !ok || db == nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "database was not selected for this request")
	}
	return db, nil
}

func setSelectedDatabase(c *fiber.Ctx, useSandbox bool, prodDB, sandboxDB *database.Database) error {
	if prodDB == nil || sandboxDB == nil {
		return fiber.NewError(fiber.StatusInternalServerError, "database connections are not configured")
	}

	if useSandbox {
		c.Locals(selectedDBKey, sandboxDB.Postgresql.DB())
	} else {
		c.Locals(selectedDBKey, prodDB.Postgresql.DB())
	}

	return c.Next()
}

func parseBool(raw string) (bool, bool) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "true", "1", "yes", "y", "on":
		return true, true
	case "false", "0", "no", "n", "off":
		return false, true
	default:
		return false, false
	}
}

func lookupBoolFromJSONPath(raw []byte, path string) (bool, bool) {
	if len(raw) == 0 {
		return false, false
	}

	var payload any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return false, false
	}

	current := payload
	for _, part := range strings.Split(path, ".") {
		obj, ok := current.(map[string]any)
		if !ok {
			return false, false
		}
		current, ok = obj[part]
		if !ok {
			return false, false
		}
	}

	switch v := current.(type) {
	case bool:
		return v, true
	case string:
		return parseBool(v)
	case float64:
		return v != 0, true
	case json.Number:
		n, err := v.Int64()
		if err != nil {
			return false, false
		}
		return n != 0, true
	default:
		return false, false
	}
}
