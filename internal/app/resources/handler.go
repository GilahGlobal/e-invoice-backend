package resources

import (
	"einvoice-access-point/internal/apperror"
	"einvoice-access-point/internal/utility"

	"github.com/gofiber/fiber/v2"
)

type Handler struct {
	svc    *Service
	Logger *utility.Logger
}

func NewHandler(logger *utility.Logger) *Handler {
	return &Handler{
		svc:    NewService(),
		Logger: logger,
	}
}

// @Summary Retrieve Invoice Types
// @Description Retrieve a list of all invoice types
// @Tags Resources
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} ResourcesResponseDto "Fetched successfully"
// @Failure 400 {object} apperror.AppError "Bad request"
// @Router /resources/invoice-types [get]
func (h *Handler) GetInvoiceTypes(c *fiber.Ctx) error {
	respData, errDetails, err := h.svc.GetInvoiceTypes()
	if err != nil {
		return apperror.New(fiber.StatusBadRequest, "error", err.Error(), errDetails, nil)
	}

	h.Logger.Info("Fetched invoice types successfully")
	rd := utility.BuildSuccessResponse(fiber.StatusOK, "Fetched successfully", respData.Data)
	return c.Status(fiber.StatusOK).JSON(rd)
}

// @Summary Retrieve Payment Means
// @Description Retrieve a list of all payment means
// @Tags Resources
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} ResourcesResponseDto "Fetched successfully"
// @Failure 400 {object} apperror.AppError "Bad request"
// @Router /resources/payment-means [get]
func (h *Handler) GetPaymentMeans(c *fiber.Ctx) error {
	respData, errDetails, err := h.svc.GetPaymentMeans()
	if err != nil {
		return apperror.New(fiber.StatusBadRequest, "error", err.Error(), errDetails, nil)
	}

	h.Logger.Info("Fetched payment means successfully")
	rd := utility.BuildSuccessResponse(fiber.StatusOK, "Fetched successfully", respData.Data)
	return c.Status(fiber.StatusOK).JSON(rd)
}

// @Summary Retrieve Tax Categories
// @Description Retrieve a list of all tax categories
// @Tags Resources
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} TaxCategoriesResponseDto "Fetched successfully"
// @Failure 400 {object} apperror.AppError "Bad request"
// @Router /resources/tax-categories [get]
func (h *Handler) GetTaxCategories(c *fiber.Ctx) error {
	respData, errDetails, err := h.svc.GetTaxCategories()
	if err != nil {
		return apperror.New(fiber.StatusBadRequest, "error", err.Error(), errDetails, nil)
	}

	h.Logger.Info("Fetched tax categories successfully")
	rd := utility.BuildSuccessResponse(fiber.StatusOK, "Fetched successfully", respData.Data)
	return c.Status(fiber.StatusOK).JSON(rd)
}

// @Summary Retrieve HSN Codes
// @Description Retrieve a list of all HSN codes
// @Tags Resources
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} HSNCodesResponseDto "Fetched successfully"
// @Failure 400 {object} apperror.AppError "Bad request"
// @Router /resources/hsn-codes [get]
func (h *Handler) GetHSNCodes(c *fiber.Ctx) error {
	respData, errDetails, err := h.svc.GetHSNCodes()
	if err != nil {
		return apperror.New(fiber.StatusBadRequest, "error", err.Error(), errDetails, nil)
	}

	h.Logger.Info("Fetched HSN codes successfully")
	rd := utility.BuildSuccessResponse(fiber.StatusOK, "Fetched successfully", respData.Data)
	return c.Status(fiber.StatusOK).JSON(rd)
}

// @Summary Retrieve Service Codes
// @Description Retrieve a list of all service codes
// @Tags Resources
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} ServiceCodesResponseDto "Fetched successfully"
// @Failure 400 {object} apperror.AppError "Bad request"
// @Router /resources/service-codes [get]
func (h *Handler) GetServiceCodes(c *fiber.Ctx) error {
	respData, errDetails, err := h.svc.GetServiceCodes()
	if err != nil {
		return apperror.New(fiber.StatusBadRequest, "error", err.Error(), errDetails, nil)
	}

	h.Logger.Info("Fetched service codes successfully")
	rd := utility.BuildSuccessResponse(fiber.StatusOK, "Fetched successfully", respData.Data)
	return c.Status(fiber.StatusOK).JSON(rd)
}

// @Summary Retrieve Currencies
// @Description Retrieve a list of all currencies
// @Tags Resources
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} CurrenciesResponseDto "Fetched successfully"
// @Failure 400 {object} apperror.AppError "Bad request"
// @Router /resources/currencies [get]
func (h *Handler) GetCurrencies(c *fiber.Ctx) error {
	respData, errDetails, err := h.svc.GetCurrencies()
	if err != nil {
		return apperror.New(fiber.StatusBadRequest, "error", err.Error(), errDetails, nil)
	}

	h.Logger.Info("Fetched currencies successfully")
	rd := utility.BuildSuccessResponse(fiber.StatusOK, "Fetched successfully", respData.Data)
	return c.Status(fiber.StatusOK).JSON(rd)
}

// @Summary Retrieve LGAs
// @Description Retrieve a list of all Local Government Areas
// @Tags Resources
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} LGAsResponseDto "Fetched successfully"
// @Failure 400 {object} apperror.AppError "Bad request"
// @Router /resources/lgas [get]
func (h *Handler) GetLGA(c *fiber.Ctx) error {
	respData, errDetails, err := h.svc.GetLGA()
	if err != nil {
		return apperror.New(fiber.StatusBadRequest, "error", err.Error(), errDetails, nil)
	}

	h.Logger.Info("Fetched LGAs successfully")
	rd := utility.BuildSuccessResponse(fiber.StatusOK, "Fetched successfully", respData.Data)
	return c.Status(fiber.StatusOK).JSON(rd)
}

// @Summary Retrieve Countries
// @Description Retrieve a list of all countries
// @Tags Resources
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} CountriesResponseDto "Fetched successfully"
// @Failure 400 {object} apperror.AppError "Bad request"
// @Router /resources/countries [get]
func (h *Handler) GetCountries(c *fiber.Ctx) error {
	respData, errDetails, err := h.svc.GetCountries()
	if err != nil {
		return apperror.New(fiber.StatusBadRequest, "error", err.Error(), errDetails, nil)
	}

	h.Logger.Info("Fetched countries successfully")
	rd := utility.BuildSuccessResponse(fiber.StatusOK, "Fetched successfully", respData.Data)
	return c.Status(fiber.StatusOK).JSON(rd)
}

// @Summary Retrieve States
// @Description Retrieve a list of all states
// @Tags Resources
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} StatesResponseDto "Fetched successfully"
// @Failure 400 {object} apperror.AppError "Bad request"
// @Router /resources/states [get]
func (h *Handler) GetStates(c *fiber.Ctx) error {
	respData, errDetails, err := h.svc.GetStates()
	if err != nil {
		return apperror.New(fiber.StatusBadRequest, "error", err.Error(), errDetails, nil)
	}

	h.Logger.Info("Fetched states successfully")
	rd := utility.BuildSuccessResponse(fiber.StatusOK, "Fetched successfully", respData.Data)
	return c.Status(fiber.StatusOK).JSON(rd)
}
