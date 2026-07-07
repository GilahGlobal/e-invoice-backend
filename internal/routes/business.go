package routes

import (
	"einvoice-access-point/internal/app/aggregator"
	"einvoice-access-point/internal/app/business"
	"einvoice-access-point/internal/core"
	"einvoice-access-point/internal/middleware"
	"fmt"

	"github.com/gofiber/fiber/v2"
)

func BusinessRoute(app *fiber.App, ApiVersion string, c *core.Container) *fiber.App {
	businessController := business.NewHandler(c.Validator, c.Logger, c.DB, c.TestDB)

	businessUrlSec := app.Group(fmt.Sprintf("%v/business", ApiVersion), middleware.Authorize(c.DB.Postgresql.DB(), c.TestDB.Postgresql.DB()), middleware.SelectDatabaseFromClaims(c.DB, c.TestDB))
	// businessUrlSec := app.Group(fmt.Sprintf("%v/business", ApiVersion), middleware.Authorize(nil, testDb.Postgresql.DB()))
	{
		// businessUrlSec.Get("", businessController.GetAllBusiness)
		businessUrlSec.Get("", businessController.GetBusiness)
		businessUrlSec.Patch("", businessController.UpdateBusinessProfile)

		businessUrlSec.Post("/crypto-keys", businessController.UploadIRNSigningKeys)
		aggregatorController := aggregator.NewHandler(c.Validator, c.Logger, c.DB, c.TestDB)
		aggregators := businessUrlSec.Group("/aggregators")
		{
			aggregators.Get("/", aggregatorController.ListAvailableAggregators)
			aggregators.Post("/invite", aggregatorController.SendAggregatorInvitation)
			aggregators.Post("/invite-by-email", aggregatorController.SendAggregatorInvitationByEmail)
			aggregators.Get("/invitations", aggregatorController.ListSentInvitations)
			aggregators.Delete("/invitations/:id", aggregatorController.RevokeAggregatorInvitation)
		}
	}

	return app
}
