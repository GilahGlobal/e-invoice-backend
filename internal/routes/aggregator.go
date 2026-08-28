package routes

import (
	"einvoice-access-point/internal/app/aggregator"
	"einvoice-access-point/internal/app/subscription"
	"einvoice-access-point/internal/core"
	"einvoice-access-point/internal/middleware"

	"github.com/gofiber/fiber/v2"
)

func AggregatorRoute(r fiber.Router, ApiVersion string, c *core.Container) {
	controller := aggregator.NewHandler(c.Validator, c.Logger, c.DB, c.TestDB, c.Config)
	subHandler := subscription.NewAggregatorHandler(c.Validator, c.Logger, c.DB, c.TestDB)

	aggregatorRoute := r.Group(ApiVersion + "/aggregator")

	// Protected Routes (Must be Authenticated and be an Aggregator)
	protected := aggregatorRoute.Group("/")
	protected.Use(middleware.Authorize(c.DB.Postgresql.DB(), c.TestDB.Postgresql.DB()))
	protected.Use(middleware.SelectDatabaseFromClaims(c.DB, c.TestDB))
	protected.Use(middleware.AggregatorGuard())

	rf := middleware.RequireFrontend()

	// Dashboard & General
	{
		protected.Get("/dashboard", rf, controller.Dashboard)
		protected.Get("/stats", rf, controller.GetInvoiceStats)
		protected.Get("/activity-log", rf, controller.ActivityLog)
	}

	// Invitations
	invitations := protected.Group("/invitations", rf)
	{
		invitations.Get("/", controller.ListInvitations)
		invitations.Post("/respond", controller.RespondToInvitation)
	}

	// Business Management
	businesses := protected.Group("/businesses", rf)
	{
		businesses.Get("/", controller.ListBusinesses)
		businesses.Post("/", controller.CreateBusiness)
		businesses.Get("/:id", controller.GetBusinessDetail)
		businesses.Get("/:id/stats", controller.GetBusinessInvoiceStats)
		businesses.Delete("/:id", controller.RemoveBusiness)
		businesses.Patch("/:id", controller.UpdateBusinessSetup)
	}

	// Invoices
	invoices := protected.Group("/invoices", rf)
	{
		invoices.Get("/", controller.ListAllInvoices)
		invoices.Get("/single/:invoice_id", controller.GetInvoiceDetail)
		invoices.Get("/:id", controller.ListBusinessInvoices)
		invoices.Post("/:id", controller.UploadInvoice)
	}

	// Bulk Uploads
	bulkUploads := protected.Group("/bulk-uploads", rf)
	{
		bulkUploads.Get("/", controller.ListAllBulkUploads)
		bulkUploads.Get("/:bulk_id/failed", controller.GetBulkUploadFailedInvoices)
		bulkUploads.Get("/:bulk_id/failed/download", controller.DownloadBulkUploadFailedInvoices)
		bulkUploads.Get("/:business_id", controller.ListBulkUploadLogs)
		bulkUploads.Post("/:id", controller.BulkUpload)
	}

	// Transactions
	{
		protected.Get("/transactions", rf, controller.ListAllTransactions)
	}

	// Subscription
	subscription := protected.Group("/subscription", rf)
	{
		subscription.Get("/plans", subHandler.GetPlans)
		subscription.Post("/subscribe", subHandler.Subscribe)
	}
}
