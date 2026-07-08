package business

import (
	"einvoice-access-point/internal/apperror"
	"einvoice-access-point/internal/data/database"
	"einvoice-access-point/internal/middleware"
	"einvoice-access-point/internal/utility"
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
// @Success      200 {object} entities.Response "Businesses retrieved successfully"
// @Failure      400 {object} entities.Response "Bad request"
// @Failure      401 {object} entities.Response "Unauthorized"
// @Failure      500 {object} entities.Response "Internal server error"
// @Router       /business [get]
func (h *Handler) GetAllBusiness(c *fiber.Ctx) error {
	businesses, err := h.svc.GetAllBusinesses(h.Db.Postgresql.DB())
	if err != nil {
		return apperror.New(http.StatusBadRequest, "error", err.Error(), err, nil)
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
// @Success      200 {object} GetBusinessResponseDto "Business retrieved successfully"
// @Failure      400 {object} entities.Response "Bad request"
// @Failure      401 {object} entities.Response "Unauthorized"
// @Failure      404 {object} entities.Response "Business not found"
// @Failure      500 {object} entities.Response "Internal server error"
// @Router       /business [get]
func (h *Handler) GetBusiness(c *fiber.Ctx) error {
	userDetails, err := middleware.GetUserDetails(c)
	if err != nil {
		return apperror.New(fiber.StatusUnauthorized, "error", "Unauthorized", err, nil)
	}

	db, err := middleware.GetDatabase(c)
	if err != nil {
		return apperror.New(fiber.StatusInternalServerError, "error", err.Error(), err, nil)
	}
	business, err := h.svc.GetBusinessByID(db, userDetails.ID)
	if err != nil {
		return apperror.New(http.StatusNotFound, "error", err.Error(), err, nil)
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
// @Param data body UpdateBusinessDto true "Update business details request payload"
// @Success      200 {object} entities.Response "Business updated successfully"
// @Failure      400 {object} entities.Response "Bad request"
// @Failure      401 {object} entities.Response "Unauthorized"
// @Failure      404 {object} entities.Response "Business not found"
// @Failure      500 {object} entities.Response "Internal server error"
// @Router       /business [patch]
func (h *Handler) UpdateBusinessProfile(c *fiber.Ctx) error {
	userDetails, err := middleware.GetUserDetails(c)
	if err != nil {
		return apperror.New(fiber.StatusUnauthorized, "error", "Unauthorized", err, nil)
	}
	db, err := middleware.GetDatabase(c)
	if err != nil {
		return apperror.New(fiber.StatusInternalServerError, "error", err.Error(), err, nil)
	}

	var req UpdateBusinessDto
	err = c.BodyParser(&req)
	if err != nil {
		return apperror.New(fiber.StatusBadRequest, "error", "Failed to parse request body", err, nil)
	}
	err = h.Validator.Struct(&req)
	if err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusUnprocessableEntity, "error", "Validation failed", utility.ValidationResponse(err, validator.New()), nil)
		return c.Status(fiber.StatusUnprocessableEntity).JSON(rd)
	}

	businessData, err := h.svc.GetBusinessDetails(db, userDetails.ID)

	if err != nil {
		return apperror.New(http.StatusBadRequest, "error", err.Error(), err, nil)
	}
	h.svc.UpdateBusinessDetails(db, *businessData, req)

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
// @Success 200 {object} UploadBusinessIRNSigningKeysResponseDto "Business IRN signing keys uploaded successfully"
// @Failure 400 {object} entities.Response "Bad request"
// @Failure 401 {object} entities.Response "Unauthorized"
// @Failure 500 {object} entities.Response "Internal server error"
// @Router /business/crypto-keys [post]
func (h *Handler) UploadIRNSigningKeys(c *fiber.Ctx) error {
	userDetails, err := middleware.GetUserDetails(c)
	if err != nil {
		return apperror.New(fiber.StatusUnauthorized, "error", "Unauthorized", err, nil)
	}

	db, err := middleware.GetDatabase(c)
	if err != nil {
		return apperror.New(fiber.StatusInternalServerError, "error", err.Error(), err, nil)
	}

	file, err := c.FormFile("file")
	if err != nil {
		return apperror.New(fiber.StatusBadRequest, "error", "crypto keys file is required", nil, nil)
	}

	openedFile, err := file.Open()
	if err != nil {
		return apperror.New(fiber.StatusBadRequest, "error", "failed to open crypto keys file", err, nil)
	}
	defer openedFile.Close()

	fileContent, err := io.ReadAll(openedFile)
	if err != nil {
		return apperror.New(fiber.StatusBadRequest, "error", "failed to read crypto keys file", err, nil)
	}

	if err := h.svc.SaveBusinessIRNSigningKeys(db, userDetails.ID, fileContent); err != nil {
		return apperror.New(fiber.StatusBadRequest, "error", err.Error(), nil, nil)
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
