package invoice

import (
	"einvoice-access-point/internal/apperror"
	"einvoice-access-point/internal/utility"

	"github.com/gofiber/fiber/v2"
)

// GetInvoiceTypes godoc
// @Summary Get Invoice Types
// @Description Fetches invoice types resource from FIRS.
// @Tags Resources
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} entities.Response "Success"
// @Failure 400 {object} entities.Response "Bad request"
// @Router /invoice/resources/invoice-types [get]
func (h *Handler) GetInvoiceTypes(c *fiber.Ctx) error {
	respData, errDetails, err := h.svc.GetInvoiceTypes()
	if err != nil {
		return apperror.New(fiber.StatusBadRequest, "error", err.Error(), errDetails, nil)
	}

	h.Logger.Info("Fetched invoice types successfully")
	rd := utility.BuildSuccessResponse(fiber.StatusOK, "Fetched successfully", respData)
	return c.Status(fiber.StatusOK).JSON(rd)
}

// GetPaymentMeans godoc
// @Summary Get Payment Means
// @Description Fetches payment means resource from FIRS.
// @Tags Resources
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} entities.Response "Success"
// @Failure 400 {object} entities.Response "Bad request"
// @Router /invoice/resources/payment-means [get]
func (h *Handler) GetPaymentMeans(c *fiber.Ctx) error {
	respData, errDetails, err := h.svc.GetPaymentMeans()
	if err != nil {
		return apperror.New(fiber.StatusBadRequest, "error", err.Error(), errDetails, nil)
	}

	h.Logger.Info("Fetched payment means successfully")
	rd := utility.BuildSuccessResponse(fiber.StatusOK, "Fetched successfully", respData)
	return c.Status(fiber.StatusOK).JSON(rd)
}

// GetTaxCategories godoc
// @Summary Get Tax Categories
// @Description Fetches tax categories resource from FIRS.
// @Tags Resources
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} entities.Response "Success"
// @Failure 400 {object} entities.Response "Bad request"
// @Router /invoice/resources/tax-categories [get]
func (h *Handler) GetTaxCategories(c *fiber.Ctx) error {
	respData, errDetails, err := h.svc.GetTaxCategories()
	if err != nil {
		return apperror.New(fiber.StatusBadRequest, "error", err.Error(), errDetails, nil)
	}

	h.Logger.Info("Fetched tax categories successfully")
	rd := utility.BuildSuccessResponse(fiber.StatusOK, "Fetched successfully", respData)
	return c.Status(fiber.StatusOK).JSON(rd)
}

// GetProductCodes godoc
// @Summary Get Product Codes
// @Description Fetches product codes resource from FIRS.
// @Tags Resources
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} entities.Response "Success"
// @Failure 400 {object} entities.Response "Bad request"
// @Router /invoice/resources/product-codes [get]
func (h *Handler) GetProductCodes(c *fiber.Ctx) error {
	respData, errDetails, err := h.svc.GetProductCodes()
	if err != nil {
		return apperror.New(fiber.StatusBadRequest, "error", err.Error(), errDetails, nil)
	}

	h.Logger.Info("Fetched product codes successfully")
	rd := utility.BuildSuccessResponse(fiber.StatusOK, "Fetched successfully", respData)
	return c.Status(fiber.StatusOK).JSON(rd)
}

// GetServiceCodes godoc
// @Summary Get Service Codes
// @Description Fetches service codes resource from FIRS.
// @Tags Resources
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} entities.Response "Success"
// @Failure 400 {object} entities.Response "Bad request"
// @Router /invoice/resources/service-codes [get]
func (h *Handler) GetServiceCodes(c *fiber.Ctx) error {
	respData, errDetails, err := h.svc.GetServiceCodes()
	if err != nil {
		return apperror.New(fiber.StatusBadRequest, "error", err.Error(), errDetails, nil)
	}

	h.Logger.Info("Fetched service codes successfully")
	rd := utility.BuildSuccessResponse(fiber.StatusOK, "Fetched successfully", respData)
	return c.Status(fiber.StatusOK).JSON(rd)
}

// GetCurrencies godoc
// @Summary Get Currencies
// @Description Fetches currencies resource from FIRS.
// @Tags Resources
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} entities.Response "Success"
// @Failure 400 {object} entities.Response "Bad request"
// @Router /invoice/resources/currencies [get]
func (h *Handler) GetCurrencies(c *fiber.Ctx) error {
	respData, errDetails, err := h.svc.GetCurrencies()
	if err != nil {
		return apperror.New(fiber.StatusBadRequest, "error", err.Error(), errDetails, nil)
	}

	h.Logger.Info("Fetched currencies successfully")
	rd := utility.BuildSuccessResponse(fiber.StatusOK, "Fetched successfully", respData)
	return c.Status(fiber.StatusOK).JSON(rd)
}

// GetLGA godoc
// @Summary Get LGAs
// @Description Fetches LGAs resource from FIRS.
// @Tags Resources
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} entities.Response "Success"
// @Failure 400 {object} entities.Response "Bad request"
// @Router /resources/lgas [get]
func (h *Handler) GetLGA(c *fiber.Ctx) error {
	respData, errDetails, err := h.svc.GetLGA()
	if err != nil {
		return apperror.New(fiber.StatusBadRequest, "error", err.Error(), errDetails, nil)
	}

	h.Logger.Info("Fetched LGAs successfully")
	rd := utility.BuildSuccessResponse(fiber.StatusOK, "Fetched successfully", respData)
	return c.Status(fiber.StatusOK).JSON(rd)
}

// GetCountries godoc
// @Summary Get Countries
// @Description Fetches countries resource from FIRS.
// @Tags Resources
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} entities.Response "Success"
// @Failure 400 {object} entities.Response "Bad request"
// @Router /resources/countries [get]
func (h *Handler) GetCountries(c *fiber.Ctx) error {
	respData, errDetails, err := h.svc.GetCountries()
	if err != nil {
		return apperror.New(fiber.StatusBadRequest, "error", err.Error(), errDetails, nil)
	}

	h.Logger.Info("Fetched countries successfully")
	rd := utility.BuildSuccessResponse(fiber.StatusOK, "Fetched successfully", respData)
	return c.Status(fiber.StatusOK).JSON(rd)
}

// GetStates godoc
// @Summary Get States
// @Description Fetches states resource from FIRS.
// @Tags Resources
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} entities.Response "Success"
// @Failure 400 {object} entities.Response "Bad request"
// @Router /resources/states [get]
func (h *Handler) GetStates(c *fiber.Ctx) error {
	respData, errDetails, err := h.svc.GetStates()
	if err != nil {
		return apperror.New(fiber.StatusBadRequest, "error", err.Error(), errDetails, nil)
	}

	h.Logger.Info("Fetched states successfully")
	rd := utility.BuildSuccessResponse(fiber.StatusOK, "Fetched successfully", respData)
	return c.Status(fiber.StatusOK).JSON(rd)
}
