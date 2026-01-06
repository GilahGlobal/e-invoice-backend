package auth

import (
	"einvoice-access-point/internal/services/auth"
	"einvoice-access-point/pkg/database"
	"einvoice-access-point/pkg/middleware"
	"einvoice-access-point/pkg/models"
	"einvoice-access-point/pkg/utility"
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
)

type Controller struct {
	Db        *database.Database
	Validator *validator.Validate
	Logger    *utility.Logger
}

// @Summary Register
// @Description Onboard to the system
// @Tags Auth
// @Accept json
// @Produce json
// @Param data body models.CreateUserRequestModel true "Login request payload"
// @Success 200 {object} models.Response "Registered successfully"
// @Failure 400 {object} models.Response "Bad request, validation failed"
// @Failure 401 {object} models.Response "Unauthorized"
// @Failure 422 {object} models.Response "Unprocessable entity"
// @Failure 500 {object} models.Response "Internal server error"
// @Router /auth/register [post]
func (base *Controller) Register(c *fiber.Ctx) error {
	var req models.CreateUserRequestModel

	err := c.BodyParser(&req)
	if err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusBadRequest, "error", "Failed to parse request body", err.Error(), nil)
		return c.Status(fiber.StatusBadRequest).JSON(rd)
	}

	err = base.Validator.Struct(&req)
	if err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusUnprocessableEntity, "error", "Validation failed", utility.ValidationResponse(err, base.Validator), nil)
		return c.Status(fiber.StatusUnprocessableEntity).JSON(rd)
	}

	reqData, err := auth.ValidateCreateUserRequest(req, base.Db.Postgresql.DB())
	if err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusBadRequest, "error", err.Error(), err, nil)
		return c.Status(fiber.StatusBadRequest).JSON(rd)
	}

	respData, code, err := auth.CreateUser(reqData, base.Db.Postgresql.DB())
	if err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusBadRequest, "error", err.Error(), err, nil)
		return c.Status(fiber.StatusBadRequest).JSON(rd)
	}

	base.Logger.Info("user created successfully")
	rd := utility.BuildSuccessResponse(fiber.StatusCreated, "user created successfully", respData)
	return c.Status(code).JSON(rd)
}

// @Summary Login
// @Description Login to the system
// @Tags Auth
// @Accept json
// @Produce json
// @Param data body models.LoginRequestModel true "Login request payload"
// @Success 200 {object} models.Response "Login successfully"
// @Failure 400 {object} models.Response "Bad request, validation failed"
// @Failure 401 {object} models.Response "Unauthorized"
// @Failure 422 {object} models.Response "Unprocessable entity"
// @Failure 500 {object} models.Response "Internal server error"
// @Router /auth/login [post]
func (base *Controller) Login(c *fiber.Ctx) error {
	var req models.LoginRequestModel

	err := c.BodyParser(&req)
	if err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusBadRequest, "error", "Failed to parse request body", err, nil)
		return c.Status(fiber.StatusBadRequest).JSON(rd)

	}

	err = base.Validator.Struct(&req)
	if err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusUnprocessableEntity, "error", "Validation failed", utility.ValidationResponse(err, base.Validator), nil)
		return c.Status(fiber.StatusUnprocessableEntity).JSON(rd)
	}

	respData, code, err := auth.LoginUser(req, base.Db.Postgresql.DB())
	if err != nil {
		rd := utility.BuildErrorResponse(code, "error", err.Error(), err, nil)
		return c.Status(fiber.StatusBadRequest).JSON(rd)
	}

	base.Logger.Info("user login successfully")

	rd := utility.BuildSuccessResponse(fiber.StatusOK, "user login successfully", respData)
	return c.Status(code).JSON(rd)
}

// @Summary Logout
// @Description Logout from the system
// @Tags Auth
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} models.Response "user logout successfully"
// @Failure 400 {object} models.Response "Bad request, validation failed"
// @Failure 401 {object} models.Response "Unauthorized"
// @Failure 422 {object} models.Response "Unprocessable entity"
// @Failure 500 {object} models.Response "Internal server error"
// @Router /auth/logout [get]
func (base *Controller) Logout(c *fiber.Ctx) error {
	userDetails, err := middleware.GetUserDetails(c)
	if err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusBadRequest, "error", "unable to get user claims", nil, nil)
		return c.Status(fiber.StatusBadRequest).JSON(rd)
	}

	accessUuid := userDetails.AccessUuid
	ownerId := userDetails.ID

	respData, code, err := auth.LogoutUser(accessUuid, ownerId, base.Db.Postgresql.DB())
	if err != nil {
		rd := utility.BuildErrorResponse(code, "error", err.Error(), err, nil)
		return c.Status(fiber.StatusBadRequest).JSON(rd)
	}

	base.Logger.Info("user logout successfully")

	rd := utility.BuildSuccessResponse(fiber.StatusOK, "user logout successfully", respData)
	return c.Status(code).JSON(rd)
}

// @Summary Initiate Forgot Password
// @Description Initiate forgot password process
// @Tags Auth
// @Accept json
// @Produce json
// @Param data body models.InitiateForgotPassword true "Forgot password request payload"
// @Success 200 {object} models.Response "forgot password initiated successfully"
// @Failure 400 {object} models.Response "Bad request, validation failed"
// @Failure 401 {object} models.Response "Unauthorized"
// @Failure 422 {object} models.Response "Unprocessable entity"
// @Failure 500 {object} models.Response "Internal server error"
// @Router /auth/initiate-forgot-password [post]
func (base *Controller) InitiateForgotPassword(c *fiber.Ctx) error {
	var req models.InitiateForgotPassword
	err := c.BodyParser(&req)
	if err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusBadRequest, "error", "Failed to parse request body", err, nil)
		return c.Status(fiber.StatusBadRequest).JSON(rd)

	}
	err = base.Validator.Struct(&req)
	if err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusUnprocessableEntity, "error", "Validation failed", utility.ValidationResponse(err, base.Validator), nil)
		return c.Status(fiber.StatusUnprocessableEntity).JSON(rd)
	}

	err = auth.InitiateForgotPassword(req, base.Db.Postgresql.DB())
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", err.Error(), err, nil)
		return c.Status(fiber.StatusBadRequest).JSON(rd)
	}

	base.Logger.Info("forgot password initiated successfully")

	rd := utility.BuildSuccessResponse(fiber.StatusOK, "forgot password initiated successfully", nil)
	return c.Status(http.StatusOK).JSON(rd)
}

// @Summary Complete Forgot Password
// @Description Complete forgot password process
// @Tags Auth
// @Accept json
// @Produce json
// @Param data body models.CompleteForgotPassword true "Complete forgot password request payload"
// @Success 200 {object} models.Response "forgot password complete successfully"
// @Failure 400 {object} models.Response "Bad request, validation failed"
// @Failure 401 {object} models.Response "Unauthorized"
// @Failure 422 {object} models.Response "Unprocessable entity"
// @Failure 500 {object} models.Response "Internal server error"
// @Router /auth/complete-forgot-password [post]
func (base *Controller) CompleteForgotPassword(c *fiber.Ctx) error {
	var req models.CompleteForgotPassword
	err := c.BodyParser(&req)
	if err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusBadRequest, "error", "Failed to parse request body", err, nil)
		return c.Status(fiber.StatusBadRequest).JSON(rd)

	}

	err = base.Validator.Struct(&req)
	if err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusUnprocessableEntity, "error", "Validation failed", utility.ValidationResponse(err, base.Validator), nil)
		return c.Status(fiber.StatusUnprocessableEntity).JSON(rd)
	}

	err = auth.CompleteForgotPassword(req, base.Db.Postgresql.DB())
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", err.Error(), err, nil)
		return c.Status(fiber.StatusBadRequest).JSON(rd)
	}

	base.Logger.Info("forgot password completed successfully")

	rd := utility.BuildSuccessResponse(fiber.StatusOK, "forgot password completed successfully", nil)
	return c.Status(http.StatusOK).JSON(rd)
}
