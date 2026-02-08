package callback

import (
	"einvoice-access-point/external/zoho"
	"einvoice-access-point/internal/services/token"
	"einvoice-access-point/internal/services/webhooks"
	"einvoice-access-point/pkg/database"
	"einvoice-access-point/pkg/utility"
	"fmt"
	"log"
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
)

type Controller struct {
	Db        *database.Database
	TestDB    *database.Database
	Validator *validator.Validate
	Logger    *utility.Logger
}

func (base *Controller) ZohoAuthCode(c *fiber.Ctx) error {

	state := "testing"
	redirectURI := "http://localhost:8091/api/v1/zoho/callback"
	authURL := zoho.GenerateAuthURL(state, redirectURI)
	fmt.Println(authURL)
	return c.Redirect(authURL)
}

// @Summary      Get Zoho Access Token
// @Description  Exchange an authorization code for a Zoho access token and save it to the database.
// @Tags         Zoho
// @Accept       json
// @Produce      json
// @Param        code            query   string  true  "Authorization Code returned from Zoho OAuth"
// @Param        organisation_id query   string  true  "Zoho Organization ID"
// @Security     BearerAuth
// @Success      200 {object} models.Response "Token generated successfully"
// @Failure      400 {object} models.Response "Bad request (missing code or organisation_id)"
// @Failure      401 {object} models.Response "Unauthorized"
// @Failure      500 {object} models.Response "Internal server error"
// @Router       /zoho/auth/access-token [get]
func (base *Controller) ZohoGetAcessToken(c *fiber.Ctx) error {

	platform := "zoho"

	code := c.Query("code")
	if code == "" {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "no coded provided", nil, nil)
		return c.Status(fiber.StatusBadRequest).JSON(rd)
	}

	orgID := c.Query("organisation_id")
	if orgID == "" {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "no organisation ID provided", nil, nil)
		return c.Status(fiber.StatusBadRequest).JSON(rd)
	}

	_, config, err := webhooks.GetBuinessConfigs(base.Db.Postgresql.DB(), platform, orgID)
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "cant get business wrong org ID", nil, nil)
		return c.Status(fiber.StatusBadRequest).JSON(rd)
	}

	tokens, err := token.GetValidAccessToken(base.Db.Postgresql.DB(), *config, platform, orgID, code)
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", err.Error(), err, nil)
		return c.Status(fiber.StatusBadRequest).JSON(rd)
	}

	log.Printf("Access Token: %s\n", tokens)
	//.Printf("Refresh Token: %s\n", tokens.RefreshToken)

	rd := utility.BuildSuccessResponse(http.StatusOK, "token generated succesfully", nil)
	return c.Status(fiber.StatusOK).JSON(rd)
}

// func (base *Controller) ZohoCallback(c *fiber.Ctx) error {

// 	code := c.Query("code")
// 	errorParam := c.Query("error")

// 	if errorParam != "" {
// 		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", errorParam, nil, nil)
// 		return c.Status(fiber.StatusBadRequest).JSON(rd)
// 	}

// 	if code == "" {
// 		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "no coded provided", nil, nil)
// 		return c.Status(fiber.StatusBadRequest).JSON(rd)
// 	}

// 	tokens, err := zoho.ExchangeCodeForTokens(code)
// 	if err != nil {
// 		return c.Status(fiber.StatusInternalServerError).SendString(err.Error())
// 	}

// 	// Here you can store the tokens, e.g., in a database or file
// 	log.Printf("Access Token: %s\n", tokens.AccessToken)
// 	log.Printf("Refresh Token: %s\n", tokens.RefreshToken)

// 	rd := utility.BuildSuccessResponse(http.StatusOK, "token generated succesfully", nil)
// 	return c.Status(fiber.StatusOK).JSON(rd)
// }
