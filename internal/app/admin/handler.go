package admin

import (
	"einvoice-access-point/internal/apperror"
	"einvoice-access-point/internal/config"
	"einvoice-access-point/internal/data/database"
	"einvoice-access-point/internal/data/dbinit"
	"einvoice-access-point/internal/middleware"
	"einvoice-access-point/internal/utility"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
)

type Handler struct {
	validator *validator.Validate
	logger    *utility.Logger
	svc       Service
}

func NewHandler(validator *validator.Validate, logger *utility.Logger, cfg *config.Configuration, db, testDb *database.Database) *Handler {
	prodDB := dbinit.InitDB(db.Postgresql.DB(), false)
	testDB := dbinit.InitDB(testDb.Postgresql.DB(), false)
	svc := NewServiceWithDB(prodDB, testDB, cfg)
	return &Handler{
		validator: validator,
		logger:    logger,
		svc:       svc,
	}
}

// @Summary Setup Initial Super Admin
// @Description Creates the first Super Admin if no admins exist in the system
// @Tags Admin Auth
// @Accept json
// @Produce json
// @Param data body AdminRegisterDto true "Initial setup request payload"
// @Success 201 {object} utility.Response "Admin created successfully"
// @Failure 400 {object} apperror.AppError "Bad request, validation failed"
// @Failure 403 {object} apperror.AppError "Forbidden, initial setup already complete"
// @Failure 500 {object} apperror.AppError "Internal server error"
// @Router /admin/auth/setup-initial [post]
func (h *Handler) SetupInitial(c *fiber.Ctx) error {
	var req AdminRegisterDto
	if err := c.BodyParser(&req); err != nil {
		return apperror.New(fiber.StatusBadRequest, "error", "Failed to parse request body", err, nil)
	}

	if err := h.validator.Struct(&req); err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusUnprocessableEntity, "error", "Validation failed", utility.ValidationResponse(err, h.validator), nil)
		return c.Status(fiber.StatusUnprocessableEntity).JSON(rd)
	}

	code, err := h.svc.SetupInitialSuperAdmin(req, false)
	if err != nil {
		return apperror.New(code, "error", err.Error(), err, nil)
	}

	// create sandbox as well
	_, _ = h.svc.SetupInitialSuperAdmin(req, true)

	h.logger.Info("Initial SuperAdmin created successfully")
	rd := utility.BuildSuccessResponse(fiber.StatusCreated, "Initial SuperAdmin created successfully", nil)
	return c.Status(code).JSON(rd)
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

	authHeader := c.Get("Authorization")
	isSandbox := strings.Contains(authHeader, "sandbox") // basic check or rely on claims
	// Let's rely on claims since it's protected by AuthorizeAdmin
	claims, ok := c.Locals("adminClaims").(*middleware.AdminDataClaims)
	if !ok {
		isSandbox = false
	} else {
		isSandbox = claims.IsSandbox
	}

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

	isSandbox := false // Defaults to prod, or you can send it in headers
	if c.Get("X-Sandbox-Mode") == "true" {
		isSandbox = true
	}

	respData, code, err := h.svc.LoginAdmin(req, isSandbox)
	if err != nil {
		return apperror.New(code, "error", err.Error(), err, nil)
	}

	h.logger.Info("Admin login successful")
	rd := utility.BuildSuccessResponse(fiber.StatusOK, "Admin login successful", respData)
	return c.Status(code).JSON(rd)
}
