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

	rf := middleware.RequireFrontend()
	adminAuthSuper.Post("/register", rf, handler.Register)

	// Protected endpoints (Requires Admin or SuperAdmin)
	adminAuthAll := adminGroup.Group("",
		middleware.AuthorizeAdmin(container.DB.Postgresql.DB(), container.TestDB.Postgresql.DB(), entities.RoleAdmin, entities.RoleSuperAdmin),
		middleware.SelectDatabaseFromClaims(container.DB, container.TestDB),
	)

	adminAuthAll.Get("/roles", rf, handler.GetRoles)
	adminAuthAll.Get("/businesses", rf, handler.GetBusinesses)
	adminAuthAll.Post("/businesses", rf, handler.CreateBusiness)
	adminAuthAll.Put("/businesses/:id", rf, handler.UpdateBusiness)
	adminAuthAll.Get("/businesses/stats/:id", rf, handler.GetBusinessDailyInvoiceStats)
	adminAuthAll.Get("/businesses/aggregator/:id", rf, handler.GetBusinessAggregatorInfo)

	adminAuthAll.Get("/aggregators", rf, handler.GetAggregators)
	adminAuthAll.Post("/aggregators", rf, handler.CreateAggregator)
	adminAuthAll.Put("/aggregators/:id", rf, handler.UpdateAggregator)
	adminAuthAll.Get("/aggregators/:id", rf, handler.GetAggregatorInfo)
	adminAuthAll.Get("/aggregators/invitations/:id", rf, handler.GetAggregatorInvitations)

	adminAuthAll.Get("/businesses/invoices/:id", rf, handler.GetInvoicesByBusiness)
	adminAuthAll.Get("/aggregators/invoices/:id", rf, handler.GetInvoicesByAggregator)
	adminAuthAll.Get("/transactions", rf, handler.GetTransactions)
	adminAuthAll.Get("/stats/invoices", rf, handler.GetInvoiceStats)
	adminAuthAll.Get("/stats/businesses", rf, handler.GetBusinessStats)
	adminAuthAll.Get("/stats/overview", rf, handler.GetOverviewStats)
}
