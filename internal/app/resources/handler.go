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

func (h *Handler) GetInvoiceTypes(c *fiber.Ctx) error {
	respData, errDetails, err := h.svc.GetInvoiceTypes()
	if err != nil {
		return apperror.New(fiber.StatusBadRequest, "error", err.Error(), errDetails, nil)
	}

	h.Logger.Info("Fetched invoice types successfully")
	rd := utility.BuildSuccessResponse(fiber.StatusOK, "Fetched successfully", respData.Data)
	return c.Status(fiber.StatusOK).JSON(rd)
}

func (h *Handler) GetPaymentMeans(c *fiber.Ctx) error {
	respData, errDetails, err := h.svc.GetPaymentMeans()
	if err != nil {
		return apperror.New(fiber.StatusBadRequest, "error", err.Error(), errDetails, nil)
	}

	h.Logger.Info("Fetched payment means successfully")
	rd := utility.BuildSuccessResponse(fiber.StatusOK, "Fetched successfully", respData.Data)
	return c.Status(fiber.StatusOK).JSON(rd)
}

func (h *Handler) GetTaxCategories(c *fiber.Ctx) error {
	respData, errDetails, err := h.svc.GetTaxCategories()
	if err != nil {
		return apperror.New(fiber.StatusBadRequest, "error", err.Error(), errDetails, nil)
	}

	h.Logger.Info("Fetched tax categories successfully")
	rd := utility.BuildSuccessResponse(fiber.StatusOK, "Fetched successfully", respData.Data)
	return c.Status(fiber.StatusOK).JSON(rd)
}

func (h *Handler) GetHSNCodes(c *fiber.Ctx) error {
	respData, errDetails, err := h.svc.GetHSNCodes()
	if err != nil {
		return apperror.New(fiber.StatusBadRequest, "error", err.Error(), errDetails, nil)
	}

	h.Logger.Info("Fetched HSN codes successfully")
	rd := utility.BuildSuccessResponse(fiber.StatusOK, "Fetched successfully", respData.Data)
	return c.Status(fiber.StatusOK).JSON(rd)
}

func (h *Handler) GetServiceCodes(c *fiber.Ctx) error {
	respData, errDetails, err := h.svc.GetServiceCodes()
	if err != nil {
		return apperror.New(fiber.StatusBadRequest, "error", err.Error(), errDetails, nil)
	}

	h.Logger.Info("Fetched service codes successfully")
	rd := utility.BuildSuccessResponse(fiber.StatusOK, "Fetched successfully", respData.Data)
	return c.Status(fiber.StatusOK).JSON(rd)
}

func (h *Handler) GetCurrencies(c *fiber.Ctx) error {
	respData, errDetails, err := h.svc.GetCurrencies()
	if err != nil {
		return apperror.New(fiber.StatusBadRequest, "error", err.Error(), errDetails, nil)
	}

	h.Logger.Info("Fetched currencies successfully")
	rd := utility.BuildSuccessResponse(fiber.StatusOK, "Fetched successfully", respData.Data)
	return c.Status(fiber.StatusOK).JSON(rd)
}

func (h *Handler) GetLGA(c *fiber.Ctx) error {
	respData, errDetails, err := h.svc.GetLGA()
	if err != nil {
		return apperror.New(fiber.StatusBadRequest, "error", err.Error(), errDetails, nil)
	}

	h.Logger.Info("Fetched LGAs successfully")
	rd := utility.BuildSuccessResponse(fiber.StatusOK, "Fetched successfully", respData.Data)
	return c.Status(fiber.StatusOK).JSON(rd)
}

func (h *Handler) GetCountries(c *fiber.Ctx) error {
	respData, errDetails, err := h.svc.GetCountries()
	if err != nil {
		return apperror.New(fiber.StatusBadRequest, "error", err.Error(), errDetails, nil)
	}

	h.Logger.Info("Fetched countries successfully")
	rd := utility.BuildSuccessResponse(fiber.StatusOK, "Fetched successfully", respData.Data)
	return c.Status(fiber.StatusOK).JSON(rd)
}

func (h *Handler) GetStates(c *fiber.Ctx) error {
	respData, errDetails, err := h.svc.GetStates()
	if err != nil {
		return apperror.New(fiber.StatusBadRequest, "error", err.Error(), errDetails, nil)
	}

	h.Logger.Info("Fetched states successfully")
	rd := utility.BuildSuccessResponse(fiber.StatusOK, "Fetched successfully", respData.Data)
	return c.Status(fiber.StatusOK).JSON(rd)
}
