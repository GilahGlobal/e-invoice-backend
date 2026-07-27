package admin

import (
	"einvoice-access-point/internal/apperror"
	"einvoice-access-point/internal/config"
	"einvoice-access-point/internal/data/database"
	"einvoice-access-point/internal/data/dbinit"
	"einvoice-access-point/internal/middleware"
	"einvoice-access-point/internal/utility"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
)

type Handler struct {
	validator *validator.Validate
	logger    *utility.Logger
	svc       Service
	Db        *database.Database
	TestDB    *database.Database
}

func NewHandler(validator *validator.Validate, logger *utility.Logger, cfg *config.Configuration, db, testDb *database.Database) *Handler {
	prodDB := dbinit.InitDB(db.Postgresql.DB(), false)
	testDB := dbinit.InitDB(testDb.Postgresql.DB(), false)
	svc := NewServiceWithDB(prodDB, testDB, cfg)
	return &Handler{
		validator: validator,
		logger:    logger,
		svc:       svc,
		Db:        db,
		TestDB:    testDb,
	}
}

// @Summary Register Admin
// @Description Register a new admin (Requires SuperAdmin privileges)
// @Tags Admin Auth
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param data body AdminRegisterDto true "Register request payload"
// @Success 201 {object} utility.Response "Admin created successfully"
// @Failure 400 {object} apperror.AppError "Bad request, validation failed"
// @Failure 401 {object} apperror.AppError "Unauthorized"
// @Failure 403 {object} apperror.AppError "Forbidden"
// @Failure 500 {object} apperror.AppError "Internal server error"
// @Router /admin/auth/register [post]
func (h *Handler) Register(c *fiber.Ctx) error {
	var req AdminRegisterDto
	if err := c.BodyParser(&req); err != nil {
		return apperror.New(fiber.StatusBadRequest, "error", "Failed to parse request body", err, nil)
	}

	if err := h.validator.Struct(&req); err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusUnprocessableEntity, "error", "Validation failed", utility.ValidationResponse(err, h.validator), nil)
		return c.Status(fiber.StatusUnprocessableEntity).JSON(rd)
	}

	claims, ok := c.Locals("adminClaims").(*middleware.AdminDataClaims)
	if !ok {
		return apperror.New(fiber.StatusUnauthorized, "error", "Unauthorized", nil, nil)
	}

	isSandbox := claims.IsSandbox

	code, err := h.svc.RegisterAdmin(req, isSandbox)
	if err != nil {
		return apperror.New(code, "error", err.Error(), err, nil)
	}

	h.logger.Info("Admin created successfully")
	rd := utility.BuildSuccessResponse(fiber.StatusCreated, "Admin created successfully", nil)
	return c.Status(code).JSON(rd)
}

// @Summary Login Admin
// @Description Login for admins
// @Tags Admin Auth
// @Accept json
// @Produce json
// @Param data body AdminLoginRequestDto true "Login request payload"
// @Success 200 {object} AdminLoginResponseDto "Login successful"
// @Failure 400 {object} apperror.AppError "Bad request, validation failed"
// @Failure 500 {object} apperror.AppError "Internal server error"
// @Router /admin/auth/login [post]
func (h *Handler) Login(c *fiber.Ctx) error {
	var req AdminLoginRequestDto
	if err := c.BodyParser(&req); err != nil {
		return apperror.New(fiber.StatusBadRequest, "error", "Failed to parse request body", err, nil)
	}

	if err := h.validator.Struct(&req); err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusUnprocessableEntity, "error", "Validation failed", utility.ValidationResponse(err, h.validator), nil)
		return c.Status(fiber.StatusUnprocessableEntity).JSON(rd)
	}

	respData, code, err := h.svc.LoginAdmin(req, req.IsSandbox)
	if err != nil {
		return apperror.New(code, "error", err.Error(), err, nil)
	}

	h.logger.Info("Admin login successful")
	rd := utility.BuildSuccessResponse(fiber.StatusOK, "Admin login successful", respData)
	return c.Status(code).JSON(rd)
}
