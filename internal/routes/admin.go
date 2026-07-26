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

	// Protected endpoints (Requires SuperAdmin)
	adminAuthSuper := adminGroup.Group("/auth",
		middleware.AuthorizeAdmin(container.DB.Postgresql.DB(), container.TestDB.Postgresql.DB(), entities.RoleSuperAdmin),
		middleware.SelectDatabaseFromClaims(container.DB, container.TestDB),
	)
	adminAuthSuper.Post("/register", handler.Register)

	// Protected endpoints (Requires Admin or SuperAdmin)
	adminAuthAll := adminGroup.Group("",
		middleware.AuthorizeAdmin(container.DB.Postgresql.DB(), container.TestDB.Postgresql.DB(), entities.RoleAdmin, entities.RoleSuperAdmin),
		middleware.SelectDatabaseFromClaims(container.DB, container.TestDB),
	)

	adminAuthAll.Get("/businesses", handler.GetBusinesses)
	adminAuthAll.Get("/aggregators", handler.GetAggregators)
	adminAuthAll.Get("/businesses/invoices/:id", handler.GetInvoicesByBusiness)
	adminAuthAll.Get("/aggregators/invoices/:id", handler.GetInvoicesByAggregator)
	adminAuthAll.Get("/transactions", handler.GetTransactions)
	adminAuthAll.Get("/stats/invoices", handler.GetInvoiceStats)
	adminAuthAll.Get("/stats/businesses", handler.GetBusinessStats)
}
