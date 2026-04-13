package v1

import (
	"einvoice-access-point/internal/controller/aggregator"
	"einvoice-access-point/pkg/database"
	"einvoice-access-point/pkg/middleware"
	"einvoice-access-point/pkg/utility"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
)

func AggregatorRoute(r fiber.Router, ApiVersion string, db, testDB *database.Database, logger *utility.Logger, validator *validator.Validate) {
	controller := &aggregator.Controller{
		Db:        db,
		TestDB:    testDB,
		Logger:    logger,
		Validator: validator,
	}

	aggregatorRoute := r.Group(ApiVersion + "/aggregator")

	// Protected Routes (Must be Authenticated and be an Aggregator)
	protected := aggregatorRoute.Group("/")
	protected.Use(middleware.Authorize(db.Postgresql.DB(), testDB.Postgresql.DB()))
	protected.Use(middleware.AggregatorGuard())

	// Portal - Dashboard & Invitations
	protected.Get("/dashboard", controller.Dashboard)
	protected.Get("/invitations", controller.ListInvitations)
	protected.Post("/invitations/respond", controller.RespondToInvitation)

	// Portal - Business Management
	protected.Get("/businesses", controller.ListBusinesses)
	protected.Get("/businesses/:id", controller.GetBusinessDetail)
	protected.Delete("/businesses/:id", controller.RemoveBusiness)

	// Portal - Log Views
	protected.Get("/invoices", controller.ListAllInvoices)
	protected.Get("/invoices/:id", controller.ListBusinessInvoices)
	protected.Get("/bulk-uploads", controller.ListAllBulkUploads)
	protected.Get("/bulk-uploads/:id", controller.ListBulkUploadLogs)
	protected.Get("/activity-log", controller.ActivityLog)

	// Portal - Invoice Uploading
	protected.Post("/invoices/:id", controller.UploadInvoice)
	protected.Post("/bulk-uploads/:id", controller.BulkUpload)
}
