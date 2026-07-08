package callback

import (
	"einvoice-access-point/internal/app/business"
	"einvoice-access-point/internal/app/invoice"
	"einvoice-access-point/internal/app/token"
	"einvoice-access-point/internal/apperror"
	"einvoice-access-point/internal/data/database"
	"einvoice-access-point/internal/data/dbinit"
	"einvoice-access-point/internal/pkg/zoho"
	"einvoice-access-point/internal/utility"
	"fmt"
	"log"
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
)

type Handler struct {
	tokenSvc   *token.Service
	invoiceSvc *invoice.Service
	Db         *database.Database
	TestDb     *database.Database
	Validator  *validator.Validate
	Logger     *utility.Logger
}

func NewHandler(validator *validator.Validate, logger *utility.Logger, db, testDb *database.Database) *Handler {
	prodDB := dbinit.InitDB(db.Postgresql.DB(), false)
	testDBConn := dbinit.InitDB(testDb.Postgresql.DB(), false)
	businessSvc := business.NewServiceWithDB(prodDB, testDBConn)
	tokenSvc := token.NewServiceWithDB(prodDB, testDBConn)
	invoiceSvc := invoice.NewServiceWithDB(prodDB, testDBConn, tokenSvc, businessSvc)
	return &Handler{
		tokenSvc:   tokenSvc,
		invoiceSvc: invoiceSvc,
		Validator:  validator,
		Logger:     logger,
		Db:         db,
		TestDb:     testDb,
	}
}

func (h *Handler) ZohoAuthCode(c *fiber.Ctx) error {

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
// @Success      200 {object} entities.Response "Token generated successfully"
// @Failure      400 {object} entities.Response "Bad request (missing code or organisation_id)"
// @Failure      401 {object} entities.Response "Unauthorized"
// @Failure      500 {object} entities.Response "Internal server error"
// @Router       /zoho/auth/access-token [get]
func (h *Handler) ZohoGetAcessToken(c *fiber.Ctx) error {

	platform := "zoho"

	code := c.Query("code")
	if code == "" {
		return apperror.New(http.StatusBadRequest, "error", "no coded provided", nil, nil)
	}

	orgID := c.Query("organisation_id")
	if orgID == "" {
		return apperror.New(http.StatusBadRequest, "error", "no organisation ID provided", nil, nil)
	}

	_, config, err := h.invoiceSvc.GetBusinessConfigs(h.Db.Postgresql.DB(), platform, orgID)
	if err != nil {
		return apperror.New(http.StatusBadRequest, "error", "cant get business wrong org ID", nil, nil)
	}

	tokens, err := h.tokenSvc.GetValidAccessToken(h.TestDb.Postgresql.DB(), *config, platform, orgID, code)
	if err != nil {
		return apperror.New(http.StatusBadRequest, "error", err.Error(), err, nil)
	}

	log.Printf("Access Token: %s\n", tokens)
	//.Printf("Refresh Token: %s\n", tokens.RefreshToken)

	rd := utility.BuildSuccessResponse(http.StatusOK, "token generated succesfully", nil)
	return c.Status(fiber.StatusOK).JSON(rd)
}

// func (h *Handler) ZohoCallback(c *fiber.Ctx) error {

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
