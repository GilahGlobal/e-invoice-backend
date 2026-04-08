package business

import (
	"einvoice-access-point/internal/dtos"
	"einvoice-access-point/internal/services/business"
	"einvoice-access-point/pkg/database"
	"einvoice-access-point/pkg/middleware"
	"einvoice-access-point/pkg/utility"
	"io"
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
)

type Controller struct {
	Db        *database.Database
	TestDb    *database.Database
	Validator *validator.Validate
	Logger    *utility.Logger
}

// @Summary      Get All Businesses
// @Description  Retrieve a list of all businesses in the system
// @Tags         Business
// @Accept       json
// @Produce      json
// @Security BearerAuth
// @Success      200 {object} models.Response "Businesses retrieved successfully"
// @Failure      400 {object} models.Response "Bad request"
// @Failure      401 {object} models.Response "Unauthorized"
// @Failure      500 {object} models.Response "Internal server error"
// @Router       /business [get]
func (base *Controller) GetAllBusiness(c *fiber.Ctx) error {
	businesses, err := business.GetAllBusinesses(base.Db.Postgresql.DB())
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", err.Error(), err, nil)
		return c.Status(fiber.StatusBadRequest).JSON(rd)
	}

	rd := utility.BuildSuccessResponse(http.StatusOK, "businesses gotten successfully", businesses)
	return c.Status(fiber.StatusOK).JSON(rd)
}

// @Summary      Get Business Details
// @Description  Retrieve details of a specific business
// @Tags         Business
// @Accept       json
// @Produce      json
// @Security BearerAuth
// @Success      200 {object} dtos.GetBusinessResponseDto "Business retrieved successfully"
// @Failure      400 {object} models.Response "Bad request"
// @Failure      401 {object} models.Response "Unauthorized"
// @Failure      404 {object} models.Response "Business not found"
// @Failure      500 {object} models.Response "Internal server error"
// @Router       /business [get]
func (base *Controller) GetBusiness(c *fiber.Ctx) error {
	userDetails, err := middleware.GetUserDetails(c)
	if err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusUnauthorized, "error", "Unauthorized", err, nil)
		return c.Status(fiber.StatusUnauthorized).JSON(rd)
	}

	db := middleware.GetDatabaseInstance(userDetails.IsSandbox, base.Db, base.TestDb)
	business, err := business.GetBusinessByID(db, userDetails.ID)
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusNotFound, "error", err.Error(), err, nil)
		return c.Status(http.StatusNotFound).JSON(rd)
	}

	rd := utility.BuildSuccessResponse(http.StatusOK, "business gotten successfully", business)
	return c.Status(http.StatusOK).JSON(rd)
}

// @Summary      Update Business Details
// @Description Update Business Details
// @Tags         Business
// @Accept       json
// @Produce      json
// @Security BearerAuth
// @Param data body dtos.UpdateBusinessDto true "Update business details request payload"
// @Success      200 {object} dtos.BaseResponseDto "Business updated successfully"
// @Failure      400 {object} models.Response "Bad request"
// @Failure      401 {object} models.Response "Unauthorized"
// @Failure      404 {object} models.Response "Business not found"
// @Failure      500 {object} models.Response "Internal server error"
// @Router       /business [patch]
func (base *Controller) UpdateBusinessProfile(c *fiber.Ctx) error {
	userDetails, err := middleware.GetUserDetails(c)
	if err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusUnauthorized, "error", "Unauthorized", err, nil)
		return c.Status(fiber.StatusUnauthorized).JSON(rd)
	}
	db := middleware.GetDatabaseInstance(userDetails.IsSandbox, base.Db, base.TestDb)

	var req dtos.UpdateBusinessDto
	err = c.BodyParser(&req)
	if err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusBadRequest, "error", "Failed to parse request body", err, nil)
		return c.Status(fiber.StatusBadRequest).JSON(rd)
	}
	err = base.Validator.Struct(&req)
	if err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusUnprocessableEntity, "error", "Validation failed", utility.ValidationResponse(err, validator.New()), nil)
		return c.Status(fiber.StatusUnprocessableEntity).JSON(rd)
	}

	businessData, err := business.GetBusinessDetails(db, userDetails.ID)

	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", err.Error(), err, nil)
		return c.Status(http.StatusBadRequest).JSON(rd)
	}
	business.UpdateBusinessDetails(db, *businessData, req)

	rd := utility.BuildSuccessResponse(http.StatusOK, "business profile updated successfully", nil)
	return c.Status(http.StatusOK).JSON(rd)
}

// UploadIRNSigningKeys godoc
// @Summary Upload Business IRN Signing Keys
// @Description Uploads the crypto keys document for a business and stores the public_key and certificate values
// @Tags Business
// @Accept multipart/form-data
// @Produce json
// @Security BearerAuth
// @Param file formData file true "Crypto keys document"
// @Success 200 {object} dtos.UploadBusinessIRNSigningKeysResponseDto "Business IRN signing keys uploaded successfully"
// @Failure 400 {object} models.Response "Bad request"
// @Failure 401 {object} models.Response "Unauthorized"
// @Failure 500 {object} models.Response "Internal server error"
// @Router /business/crypto-keys [post]
func (base *Controller) UploadIRNSigningKeys(c *fiber.Ctx) error {
	userDetails, err := middleware.GetUserDetails(c)
	if err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusUnauthorized, "error", "Unauthorized", err, nil)
		return c.Status(fiber.StatusUnauthorized).JSON(rd)
	}

	db := middleware.GetDatabaseInstance(userDetails.IsSandbox, base.Db, base.TestDb)

	file, err := c.FormFile("file")
	if err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusBadRequest, "error", "crypto keys file is required", nil, nil)
		return c.Status(fiber.StatusBadRequest).JSON(rd)
	}

	openedFile, err := file.Open()
	if err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusBadRequest, "error", "failed to open crypto keys file", err, nil)
		return c.Status(fiber.StatusBadRequest).JSON(rd)
	}
	defer openedFile.Close()

	fileContent, err := io.ReadAll(openedFile)
	if err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusBadRequest, "error", "failed to read crypto keys file", err, nil)
		return c.Status(fiber.StatusBadRequest).JSON(rd)
	}

	if err := business.SaveBusinessIRNSigningKeys(db, userDetails.ID, fileContent); err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusBadRequest, "error", err.Error(), nil, nil)
		return c.Status(fiber.StatusBadRequest).JSON(rd)
	}

	environment := "production"
	if userDetails.IsSandbox {
		environment = "sandbox"
	}

	rd := utility.BuildSuccessResponse(http.StatusOK, "business IRN signing keys uploaded successfully", fiber.Map{
		"file_name":              file.Filename,
		"environment":            environment,
		"irn_signing_configured": true,
	})
	return c.Status(http.StatusOK).JSON(rd)
}
