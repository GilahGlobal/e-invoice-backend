package routes

import (
	"einvoice-access-point/internal/app/admin"
	"einvoice-access-point/internal/core"
	"einvoice-access-point/internal/data/entities"
	"einvoice-access-point/internal/middleware"

	"github.com/gofiber/fiber/v2"
)

func AdminRoute(router fiber.Router, version string, container *core.Container) {
	handler := admin.NewHandler(container.Validator, container.Logger, container.Config, container.DB, container.TestDB)

	adminGroup := router.Group(version + "/admin")

	// Public / Unauthenticated
	adminGroup.Post("/auth/login", handler.Login)
	adminGroup.Post("/auth/setup-initial", handler.SetupInitial)

	// Protected endpoints (Requires SuperAdmin)
	adminAuthSuper := adminGroup.Group("/auth", middleware.AuthorizeAdmin(container.DB.Postgresql.DB(), container.TestDB.Postgresql.DB(), entities.RoleSuperAdmin))
	adminAuthSuper.Post("/register", handler.Register)

}
