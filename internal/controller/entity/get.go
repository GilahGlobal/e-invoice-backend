package entity

import (
	"einvoice-access-point/internal/services/entity"
	"einvoice-access-point/pkg/database"
	"einvoice-access-point/pkg/models"
	"einvoice-access-point/pkg/utility"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
)

type Controller struct {
	Db        *database.Database
	TestDB    *database.Database
	Validator *validator.Validate
	Logger    *utility.Logger
}

// @Summary      Get Entities
// @Description  Retrieve a paginated list of entities
// @Tags         Entity
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        query  query     models.PaginationQuery  false  "Pagination and sorting"
// @Success      200 {object} models.Response "Entities retrieved successfully"
// @Failure      400 {object} models.Response "Bad request"
// @Failure      401 {object} models.Response "Unauthorized"
// @Failure      500 {object} models.Response "Internal server error"
// @Router       /entity [get]
func (base *Controller) GetEntities(c *fiber.Ctx) error {
	var query models.PaginationQuery
	if err := c.QueryParser(&query); err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusBadRequest, "error", "Invalid query parameters", err, nil)
		return c.Status(fiber.StatusBadRequest).JSON(rd)
	}

	queries := entity.FetchQueryItems(query)

	respData, errDetails, err := entity.GetEntities(queries)
	if err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusBadRequest, "error", err.Error(), errDetails, nil)
		return c.Status(fiber.StatusBadRequest).JSON(rd)
	}

	base.Logger.Info("Entities gotten successfully")
	rd := utility.BuildSuccessResponse(fiber.StatusOK, "Entities gotten successfully", respData)
	return c.Status(fiber.StatusOK).JSON(rd)
}

// @Summary      Get Entity by ID
// @Description  Retrieve details of a specific entity using its ID
// @Tags         Entity
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        entity_id   path      string  true  "Entity ID"
// @Success      200 {object} models.Response "Entity retrieved successfully"
// @Failure      400 {object} models.Response "Bad request"
// @Failure      401 {object} models.Response "Unauthorized"
// @Failure      404 {object} models.Response "Entity not found"
// @Failure      500 {object} models.Response "Internal server error"
// @Router       /entity/{entity_id} [get]
func (base *Controller) GetEntity(c *fiber.Ctx) error {
	entityId := c.Params("entity_id")
	if entityId == "" {
		rd := utility.BuildErrorResponse(fiber.StatusBadRequest, "error", "entity id is required", nil, nil)
		return c.Status(fiber.StatusBadRequest).JSON(rd)
	}

	respData, errDetails, err := entity.GetEntity(entityId)
	if err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusBadRequest, "error", err.Error(), errDetails, nil)
		return c.Status(fiber.StatusBadRequest).JSON(rd)
	}

	base.Logger.Info("successfully")
	rd := utility.BuildSuccessResponse(fiber.StatusOK, "successfully", respData)
	return c.Status(fiber.StatusOK).JSON(rd)
}
