package auth

import (
	"einvoice-access-point/internal/apperror"
	"einvoice-access-point/internal/config"
	"einvoice-access-point/internal/data/database"
	"einvoice-access-point/internal/data/dbinit"
	"einvoice-access-point/internal/middleware"
	"einvoice-access-point/internal/utility"
	"net/http"
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

// @Summary Register
// @Description Onboard to the system
// @Tags Auth
// @Accept json
// @Produce json
// @Param data body RegisterDto true "Register request payload"
// @Success 200 {object} RegisterResponseDto "Registered successfully"
// @Failure 400 {object} apperror.AppError "Bad request, validation failed"
// @Failure 401 {object} apperror.AppError "Unauthorized"
// @Failure 422 {object} apperror.AppError "Unprocessable entity"
// @Failure 500 {object} apperror.AppError "Internal server error"
// @Router /auth/register [post]
func (h *Handler) Register(c *fiber.Ctx) error {
	var req RegisterDto

	err := c.BodyParser(&req)
	if err != nil {
		return apperror.New(fiber.StatusBadRequest, "error", "Failed to parse request body", err, nil)
	}

	err = h.validator.Struct(&req)
	if err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusUnprocessableEntity, "error", "Validation failed", utility.ValidationResponse(err, h.validator), nil)
		return c.Status(fiber.StatusUnprocessableEntity).JSON(rd)
	}

	reqData, err := h.svc.ValidateCreateUserRequest(req)
	if err != nil {
		return apperror.New(fiber.StatusBadRequest, "error", err.Error(), err, nil)
	}

	// create test account
	code, err := h.svc.CreateUser(reqData, true)
	if err != nil {
		return apperror.New(fiber.StatusBadRequest, "error", err.Error(), err, nil)
	}

	// create prod account
	_, err = h.svc.CreateUser(reqData, false)
	if err != nil {
		return apperror.New(fiber.StatusBadRequest, "error", err.Error(), err, nil)
	}

	go h.svc.SendOtp(strings.ToLower(reqData.Email), VerifyEmailKey(req.Email))

	h.logger.Info("user created successfully")
	rd := utility.BuildSuccessResponse(fiber.StatusCreated, "An otp has been sent to your mail, use it to verify your account", nil)
	return c.Status(code).JSON(rd)
}

// @Summary Resend verification OTP
// @Description Resend email verification OTP
// @Tags Auth
// @Accept json
// @Produce json
// @Param data body ResendVerificationOtpDto true "Resend verification OTP payload"
// @Success 200 {object} BaseResponseDto "OTP sent successfully"
// @Failure 400 {object} apperror.AppError "Bad request, validation failed"
// @Failure 401 {object} apperror.AppError "Unauthorized"
// @Failure 422 {object} apperror.AppError "Unprocessable entity"
// @Failure 500 {object} apperror.AppError "Internal server error"
// @Router /auth/resend-otp [post]
func (h *Handler) ResendVerificationOTP(c *fiber.Ctx) error {
	var req ResendVerificationOtpDto

	err := c.BodyParser(&req)
	if err != nil {
		return apperror.New(fiber.StatusBadRequest, "error", "Failed to parse request body", err, nil)
	}

	err = h.validator.Struct(&req)
	if err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusUnprocessableEntity, "error", "Validation failed", utility.ValidationResponse(err, h.validator), nil)
		return c.Status(fiber.StatusUnprocessableEntity).JSON(rd)
	}

	if err := h.svc.ResendVerificationOTP(req.Email, true); err != nil {
		return apperror.New(fiber.StatusBadRequest, "error", err.Error(), err, nil)
	}

	h.logger.Info("verification otp resent successfully")
	rd := utility.BuildSuccessResponse(fiber.StatusOK, "An otp has been sent to your mail, use it to verify your account", nil)
	return c.Status(fiber.StatusOK).JSON(rd)
}

// @Summary Login
// @Description Login to the system
// @Tags Auth
// @Accept json
// @Produce json
// @Param data body LoginRequestDto true "Login request payload"
// @Success 200 {object} LoginResponseDto "Login successfully"
// @Failure 400 {object} apperror.AppError "Bad request, validation failed"
// @Failure 401 {object} apperror.AppError "Unauthorized"
// @Failure 422 {object} apperror.AppError "Unprocessable entity"
// @Failure 500 {object} apperror.AppError "Internal server error"
// @Router /auth/login [post]
func (h *Handler) Login(c *fiber.Ctx) error {
	var req LoginRequestDto

	err := c.BodyParser(&req)
	if err != nil {
		return apperror.New(fiber.StatusBadRequest, "error", "Failed to parse request body", err, nil)

	}

	err = h.validator.Struct(&req)
	if err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusUnprocessableEntity, "error", "Validation failed", utility.ValidationResponse(err, h.validator), nil)
		return c.Status(fiber.StatusUnprocessableEntity).JSON(rd)
	}

	if !req.IsSandbox {
		err = h.svc.SynchronizeSandboxToProduction(req.Email)
		if err != nil {
			return apperror.New(fiber.StatusBadRequest, "error", err.Error(), err, nil)
		}
	}

	respData, code, err := h.svc.LoginUser(req, req.IsSandbox)
	if err != nil {
		return apperror.New(code, "error", err.Error(), err, nil)
	}

	h.logger.Info("user login successfully")

	rd := utility.BuildSuccessResponse(fiber.StatusOK, "user login successfully", respData)
	return c.Status(code).JSON(rd)
}

// @Summary Logout
// @Description Logout from the system
// @Tags Auth
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} BaseResponseDto "user logout successfully"
// @Failure 400 {object} apperror.AppError "Bad request, validation failed"
// @Failure 401 {object} apperror.AppError "Unauthorized"
// @Failure 422 {object} apperror.AppError "Unprocessable entity"
// @Failure 500 {object} apperror.AppError "Internal server error"
// @Router /auth/logout [get]
func (h *Handler) Logout(c *fiber.Ctx) error {
	userDetails, err := middleware.GetUserDetails(c)

	if err != nil {
		return apperror.New(fiber.StatusBadRequest, "error", "unable to get user claims", nil, nil)
	}

	accessUuid := userDetails.AccessUuid
	ownerId := userDetails.ID

	respData, code, err := h.svc.LogoutUser(accessUuid, ownerId, userDetails.IsSandbox)
	if err != nil {
		return apperror.New(code, "error", err.Error(), err, nil)
	}

	h.logger.Info("user logout successfully")

	rd := utility.BuildSuccessResponse(fiber.StatusOK, "user logout successfully", respData)
	return c.Status(code).JSON(rd)
}

// @Summary Change Password
// @Description Change the password of an authenticated user by verifying the old password first
// @Tags Auth
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param data body ChangePasswordDto true "Change password request payload"
// @Success 200 {object} BaseResponseDto "password changed successfully"
// @Failure 400 {object} apperror.AppError "Bad request"
// @Failure 401 {object} apperror.AppError "Unauthorized"
// @Failure 422 {object} apperror.AppError "Unprocessable entity"
// @Failure 500 {object} apperror.AppError "Internal server error"
// @Router /auth/change-password [post]
func (h *Handler) ChangePassword(c *fiber.Ctx) error {
	userDetails, err := middleware.GetUserDetails(c)
	if err != nil {
		return apperror.New(fiber.StatusUnauthorized, "error", "Unauthorized", err, nil)
	}

	var req ChangePasswordDto
	if err := c.BodyParser(&req); err != nil {
		return apperror.New(fiber.StatusBadRequest, "error", "Failed to parse request body", err, nil)
	}

	if err := h.validator.Struct(&req); err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusUnprocessableEntity, "error", "Validation failed", utility.ValidationResponse(err, h.validator), nil)
		return c.Status(fiber.StatusUnprocessableEntity).JSON(rd)
	}

	if err := h.svc.ChangePassword(userDetails.ID, req, userDetails.IsSandbox); err != nil {
		return apperror.New(fiber.StatusBadRequest, "error", err.Error(), err, nil)
	}

	rd := utility.BuildSuccessResponse(fiber.StatusOK, "password changed successfully", nil)
	return c.Status(fiber.StatusOK).JSON(rd)
}

// @Summary Initiate Forgot Password
// @Description Initiate forgot password process
// @Tags Auth
// @Accept json
// @Produce json
// @Param data body InitiateForgotPasswordDto true "Forgot password request payload"
// @Success 200 {object} BaseResponseDto "forgot password initiated successfully"
// @Failure 400 {object} apperror.AppError "Bad request, validation failed"
// @Failure 401 {object} apperror.AppError "Unauthorized"
// @Failure 422 {object} apperror.AppError "Unprocessable entity"
// @Failure 500 {object} apperror.AppError "Internal server error"
// @Router /auth/initiate-forgot-password [post]
func (h *Handler) InitiateForgotPassword(c *fiber.Ctx) error {
	var req InitiateForgotPasswordDto
	err := c.BodyParser(&req)
	if err != nil {
		return apperror.New(fiber.StatusBadRequest, "error", "Failed to parse request body", err, nil)

	}
	err = h.validator.Struct(&req)
	if err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusUnprocessableEntity, "error", "Validation failed", utility.ValidationResponse(err, h.validator), nil)
		return c.Status(fiber.StatusUnprocessableEntity).JSON(rd)
	}

	err = h.svc.InitiateForgotPasswordAcrossEnvironments(req)
	if err != nil {
		return apperror.New(http.StatusBadRequest, "error", err.Error(), err, nil)
	}

	h.logger.Info("forgot password initiated successfully")

	rd := utility.BuildSuccessResponse(fiber.StatusOK, "forgot password initiated successfully", nil)
	return c.Status(http.StatusOK).JSON(rd)
}

// @Summary Complete Forgot Password
// @Description Complete forgot password process
// @Tags Auth
// @Accept json
// @Produce json
// @Param data body CompleteForgotPasswordDto true "Complete forgot password request payload"
// @Success 200 {object} BaseResponseDto "forgot password complete successfully"
// @Failure 400 {object} apperror.AppError "Bad request, validation failed"
// @Failure 401 {object} apperror.AppError "Unauthorized"
// @Failure 422 {object} apperror.AppError "Unprocessable entity"
// @Failure 500 {object} apperror.AppError "Internal server error"
// @Router /auth/complete-forgot-password [post]
func (h *Handler) CompleteForgotPassword(c *fiber.Ctx) error {
	var req CompleteForgotPasswordDto
	err := c.BodyParser(&req)
	if err != nil {
		return apperror.New(fiber.StatusBadRequest, "error", "Failed to parse request body", err, nil)

	}

	err = h.validator.Struct(&req)
	if err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusUnprocessableEntity, "error", "Validation failed", utility.ValidationResponse(err, h.validator), nil)
		return c.Status(fiber.StatusUnprocessableEntity).JSON(rd)
	}

	err = h.svc.CompleteForgotPasswordAcrossEnvironments(req)
	if err != nil {
		return apperror.New(http.StatusBadRequest, "error", err.Error(), err, nil)
	}

	h.logger.Info("forgot password completed successfully")

	rd := utility.BuildSuccessResponse(fiber.StatusOK, "forgot password completed successfully", nil)
	return c.Status(http.StatusOK).JSON(rd)
}

// @Summary Toggle Application mode
// @Description Toggle Application mode from either sandox to prod or vice cers
// @Tags Auth
// @Produce json
// @Security BearerAuth
// @Success 200 {object} LoginResponseDto "Application mode toggled successfully"
// @Failure 400 {object} apperror.AppError "Bad request, validation failed"
// @Failure 401 {object} apperror.AppError "Unauthorized"
// @Failure 422 {object} apperror.AppError "Unprocessable entity"
// @Failure 500 {object} apperror.AppError "Internal server error"
// @Router /auth/toggle-mode [get]
func (h *Handler) ToggleApplicationMode(c *fiber.Ctx) error {
	userDetails, err := middleware.GetUserDetails(c)

	if err != nil {
		return apperror.New(fiber.StatusBadRequest, "error", "unable to get user claims", nil, nil)
	}
	if !userDetails.IsSandbox {
		err := h.svc.SynchronizeSandboxToProduction(userDetails.Email)
		if err != nil {
			return apperror.New(fiber.StatusBadRequest, "error", err.Error(), err, nil)
		}
	}

	respData, code, err := h.svc.ToggleApplicationMode(userDetails.Email, !userDetails.IsSandbox)
	if err != nil {
		return apperror.New(code, "error", err.Error(), err, nil)
	}

	h.logger.Info("application mode switched successfully")

	rd := utility.BuildSuccessResponse(fiber.StatusOK, "application mode switched successfully", respData)
	return c.Status(code).JSON(rd)
}

// @Summary Verify Email of Business Accounts
// @Description Verify email of business accounts
// @Tags Auth
// @Accept json
// @Produce json
// @Param data body VerifyEmailDto true "Verify Email request payload"
// @Success 200 {object} LoginResponseDto "Verified successfully"
// @Failure 400 {object} apperror.AppError "Bad request, validation failed"
// @Failure 401 {object} apperror.AppError "Unauthorized"
// @Failure 422 {object} apperror.AppError "Unprocessable entity"
// @Failure 500 {object} apperror.AppError "Internal server error"
// @Router /auth/verify-email [post]
func (h *Handler) VerifyEmail(c *fiber.Ctx) error {
	var req VerifyEmailDto

	err := c.BodyParser(&req)
	if err != nil {
		return apperror.New(fiber.StatusBadRequest, "error", "Failed to parse request body", err, nil)
	}

	err = h.validator.Struct(&req)
	if err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusUnprocessableEntity, "error", "Validation failed", utility.ValidationResponse(err, h.validator), nil)
		return c.Status(fiber.StatusUnprocessableEntity).JSON(rd)
	}

	respData, err := h.svc.VerifyBusinessAccount(req, true)
	if err != nil {
		return apperror.New(fiber.StatusBadRequest, "error", err.Error(), err, nil)
	}

	// verify prod account
	err = h.svc.VerifyProdBuisnessAccount(req)
	if err != nil {
		return apperror.New(fiber.StatusBadRequest, "error", err.Error(), err, nil)
	}

	h.logger.Info("user verified successfully")
	rd := utility.BuildSuccessResponse(fiber.StatusCreated, "user verified successfully", respData)
	return c.Status(http.StatusOK).JSON(rd)
}
