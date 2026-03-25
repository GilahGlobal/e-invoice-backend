package auth

import (
	"einvoice-access-point/internal/dtos"
	"einvoice-access-point/internal/services/auth"
	"einvoice-access-point/pkg/database"
	"einvoice-access-point/pkg/middleware"
	"einvoice-access-point/pkg/utility"
	"net/http"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type Controller struct {
	Db        *database.Database
	TestDB    *database.Database
	Validator *validator.Validate
	Logger    *utility.Logger
}

func (base *Controller) forgotPasswordTargets() []*gorm.DB {
	targets := make([]*gorm.DB, 0, 2)

	if base.TestDB != nil && base.TestDB.Postgresql != nil {
		targets = append(targets, base.TestDB.Postgresql.DB())
	}

	if base.Db != nil && base.Db.Postgresql != nil {
		targets = append(targets, base.Db.Postgresql.DB())
	}

	return targets
}

// @Summary Register
// @Description Onboard to the system
// @Tags Auth
// @Accept json
// @Produce json
// @Param data body dtos.RegisterDto true "Register request payload"
// @Success 200 {object} dtos.RegisterResponseDto "Registered successfully"
// @Failure 400 {object} models.Response "Bad request, validation failed"
// @Failure 401 {object} models.Response "Unauthorized"
// @Failure 422 {object} models.Response "Unprocessable entity"
// @Failure 500 {object} models.Response "Internal server error"
// @Router /auth/register [post]
func (base *Controller) Register(c *fiber.Ctx) error {
	var req dtos.RegisterDto

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

	reqData, err := auth.ValidateCreateUserRequest(req, base.TestDB.Postgresql.DB())
	if err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusBadRequest, "error", err.Error(), err, nil)
		return c.Status(fiber.StatusBadRequest).JSON(rd)
	}

	// create test account
	code, err := auth.CreateUser(reqData, base.TestDB.Postgresql.DB())
	if err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusBadRequest, "error", err.Error(), err, nil)
		return c.Status(fiber.StatusBadRequest).JSON(rd)
	}

	// create prod account
	_, err = auth.CreateUser(reqData, base.Db.Postgresql.DB())
	if err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusBadRequest, "error", err.Error(), err, nil)
		return c.Status(fiber.StatusBadRequest).JSON(rd)
	}

	auth.SendOtp(strings.ToLower(reqData.Email))

	base.Logger.Info("user created successfully")
	rd := utility.BuildSuccessResponse(fiber.StatusCreated, "An otp has been sent to your mail, use it to verify your account", nil)
	return c.Status(code).JSON(rd)
}

// @Summary Resend verification OTP
// @Description Resend email verification OTP
// @Tags Auth
// @Accept json
// @Produce json
// @Param data body dtos.ResendVerificationOtpDto true "Resend verification OTP payload"
// @Success 200 {object} dtos.BaseResponseDto "OTP sent successfully"
// @Failure 400 {object} models.Response "Bad request, validation failed"
// @Failure 401 {object} models.Response "Unauthorized"
// @Failure 422 {object} models.Response "Unprocessable entity"
// @Failure 500 {object} models.Response "Internal server error"
// @Router /auth/resend-verification-otp [post]
func (base *Controller) ResendVerificationOTP(c *fiber.Ctx) error {
	var req dtos.ResendVerificationOtpDto

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

	if err := auth.ResendVerificationOTP(base.TestDB.Postgresql.DB(), req.Email); err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusBadRequest, "error", err.Error(), err, nil)
		return c.Status(fiber.StatusBadRequest).JSON(rd)
	}

	base.Logger.Info("verification otp resent successfully")
	rd := utility.BuildSuccessResponse(fiber.StatusOK, "An otp has been sent to your mail, use it to verify your account", nil)
	return c.Status(fiber.StatusOK).JSON(rd)
}

// @Summary Login
// @Description Login to the system
// @Tags Auth
// @Accept json
// @Produce json
// @Param data body dtos.LoginRequestDto true "Login request payload"
// @Success 200 {object} dtos.LoginResponseDto "Login successfully"
// @Failure 400 {object} models.Response "Bad request, validation failed"
// @Failure 401 {object} models.Response "Unauthorized"
// @Failure 422 {object} models.Response "Unprocessable entity"
// @Failure 500 {object} models.Response "Internal server error"
// @Router /auth/login [post]
func (base *Controller) Login(c *fiber.Ctx) error {
	var req dtos.LoginRequestDto

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

	if !req.IsSandbox {
		err = auth.SynchronizeSandboxToProduction(base.Db, base.TestDB, req.Email)
		if err != nil {
			rd := utility.BuildErrorResponse(fiber.StatusBadRequest, "error", err.Error(), err, nil)
			return c.Status(fiber.StatusBadRequest).JSON(rd)
		}
		// rd := utility.BuildErrorResponse(400, "error", "live environment not available at the moment", err, nil)
		// return c.Status(fiber.StatusBadRequest).JSON(rd)
	}

	db := middleware.GetDatabaseInstance(req.IsSandbox, base.Db, base.TestDB)

	respData, code, err := auth.LoginUser(req, db)
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
// @Success 200 {object} dtos.BaseResponseDto "user logout successfully"
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
	db := middleware.GetDatabaseInstance(userDetails.IsSandbox, base.Db, base.TestDB)

	accessUuid := userDetails.AccessUuid
	ownerId := userDetails.ID

	respData, code, err := auth.LogoutUser(accessUuid, ownerId, db)
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
// @Param data body dtos.InitiateForgotPasswordDto true "Forgot password request payload"
// @Success 200 {object} dtos.BaseResponseDto "forgot password initiated successfully"
// @Failure 400 {object} models.Response "Bad request, validation failed"
// @Failure 401 {object} models.Response "Unauthorized"
// @Failure 422 {object} models.Response "Unprocessable entity"
// @Failure 500 {object} models.Response "Internal server error"
// @Router /auth/initiate-forgot-password [post]
func (base *Controller) InitiateForgotPassword(c *fiber.Ctx) error {
	var req dtos.InitiateForgotPasswordDto
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

	err = auth.InitiateForgotPassword(req, base.TestDB.Postgresql.DB())
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
// @Param data body dtos.CompleteForgotPasswordDto true "Complete forgot password request payload"
// @Success 200 {object} dtos.BaseResponseDto "forgot password complete successfully"
// @Failure 400 {object} models.Response "Bad request, validation failed"
// @Failure 401 {object} models.Response "Unauthorized"
// @Failure 422 {object} models.Response "Unprocessable entity"
// @Failure 500 {object} models.Response "Internal server error"
// @Router /auth/complete-forgot-password [post]
func (base *Controller) CompleteForgotPassword(c *fiber.Ctx) error {
	var req dtos.CompleteForgotPasswordDto
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

	err = auth.CompleteForgotPasswordAcrossEnvironments(req, base.forgotPasswordTargets()...)
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", err.Error(), err, nil)
		return c.Status(fiber.StatusBadRequest).JSON(rd)
	}

	base.Logger.Info("forgot password completed successfully")

	rd := utility.BuildSuccessResponse(fiber.StatusOK, "forgot password completed successfully", nil)
	return c.Status(http.StatusOK).JSON(rd)
}

// @Summary Toggle Application mode
// @Description Toggle Application mode from either sandox to prod or vice cers
// @Tags Auth
// @Produce json
// @Security BearerAuth
// @Success 200 {object} dtos.LoginResponseDto "Application mode toggled successfully"
// @Failure 400 {object} models.Response "Bad request, validation failed"
// @Failure 401 {object} models.Response "Unauthorized"
// @Failure 422 {object} models.Response "Unprocessable entity"
// @Failure 500 {object} models.Response "Internal server error"
// @Router /auth/toggle-mode [get]
func (base *Controller) ToggleApplicationMode(c *fiber.Ctx) error {
	userDetails, err := middleware.GetUserDetails(c)

	if err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusBadRequest, "error", "unable to get user claims", nil, nil)
		return c.Status(fiber.StatusBadRequest).JSON(rd)
	}
	if !userDetails.IsSandbox {
		err := auth.SynchronizeSandboxToProduction(base.Db, base.TestDB, userDetails.Email)
		if err != nil {
			rd := utility.BuildErrorResponse(fiber.StatusBadRequest, "error", err.Error(), err, nil)
			return c.Status(fiber.StatusBadRequest).JSON(rd)
		}
		// rd := utility.BuildErrorResponse(400, "error", "live environment not available at the moment", err, nil)
		// return c.Status(fiber.StatusBadRequest).JSON(rd)
	}
	db := middleware.GetDatabaseInstance(!userDetails.IsSandbox, base.Db, base.TestDB)

	respData, code, err := auth.ToggleApllicationMode(db, userDetails.Email, !userDetails.IsSandbox)
	if err != nil {
		rd := utility.BuildErrorResponse(code, "error", err.Error(), err, nil)
		return c.Status(fiber.StatusBadRequest).JSON(rd)
	}

	base.Logger.Info("application mode switched successfully")

	rd := utility.BuildSuccessResponse(fiber.StatusOK, "application mode switched successfully", respData)
	return c.Status(code).JSON(rd)
}

// @Summary Verify Email of Business Accounts
// @Description Verify email of business accounts
// @Tags Auth
// @Accept json
// @Produce json
// @Param data body dtos.VerifyEmailDto true "Verify Email request payload"
// @Success 200 {object} dtos.LoginResponseDto "Verified successfully"
// @Failure 400 {object} models.Response "Bad request, validation failed"
// @Failure 401 {object} models.Response "Unauthorized"
// @Failure 422 {object} models.Response "Unprocessable entity"
// @Failure 500 {object} models.Response "Internal server error"
// @Router /auth/verify-email [post]
func (base *Controller) VerifyEmail(c *fiber.Ctx) error {
	var req dtos.VerifyEmailDto

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

	respData, err := auth.VerifyBusinessAccount(base.TestDB.Postgresql.DB(), req, true)
	if err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusBadRequest, "error", err.Error(), err, nil)
		return c.Status(fiber.StatusBadRequest).JSON(rd)
	}

	// verify prod account
	err = auth.VerifyProdBuisnessAccount(base.Db.Postgresql.DB(), req)
	if err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusBadRequest, "error", err.Error(), err, nil)
		return c.Status(fiber.StatusBadRequest).JSON(rd)
	}

	base.Logger.Info("user verified successfully")
	rd := utility.BuildSuccessResponse(fiber.StatusCreated, "user verified successfully", respData)
	return c.Status(http.StatusOK).JSON(rd)
}
