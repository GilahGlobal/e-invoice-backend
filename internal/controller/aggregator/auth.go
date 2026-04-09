package aggregator

import (
	"einvoice-access-point/internal/dtos"
	aggregatorSvc "einvoice-access-point/internal/services/aggregator"
	"einvoice-access-point/pkg/database"
	"einvoice-access-point/pkg/middleware"
	"einvoice-access-point/pkg/utility"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
)

type Controller struct {
	Db        *database.Database
	TestDB    *database.Database
	Logger    *utility.Logger
	Validator *validator.Validate
}

// Register aggregator
func (base *Controller) Register(c *fiber.Ctx) error {
	var req dtos.AggregatorRegisterDto
	db := middleware.GetDatabaseInstance(true, base.Db, base.TestDB)

	if err := c.BodyParser(&req); err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusBadRequest, "error", "Failed to parse request body", err, nil)
		return c.Status(fiber.StatusBadRequest).JSON(rd)
	}

	if err := base.Validator.Struct(&req); err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusUnprocessableEntity, "error", "Validation failed", utility.ValidationErrorsToJSON(err, dtos.AggregatorRegisterDto{}), nil)
		return c.Status(fiber.StatusUnprocessableEntity).JSON(rd)
	}

	req, err := aggregatorSvc.ValidateRegisterRequest(req, base.TestDB.Postgresql.DB(), base.Db.Postgresql.DB())
	if err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusBadRequest, "error", err.Error(), err, nil)
		return c.Status(fiber.StatusBadRequest).JSON(rd)
	}

	status, err := aggregatorSvc.RegisterAggregator(req, db)
	if err != nil {
		rd := utility.BuildErrorResponse(status, "error", err.Error(), err, nil)
		return c.Status(status).JSON(rd)
	}

	rd := utility.BuildSuccessResponse(status, "Aggregator registered successfully. Please verify your email.", nil)
	return c.Status(status).JSON(rd)
}

// Login aggregator
func (base *Controller) Login(c *fiber.Ctx) error {
	var req dtos.AggregatorLoginDto

	if err := c.BodyParser(&req); err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusBadRequest, "error", "Failed to parse request body", err, nil)
		return c.Status(fiber.StatusBadRequest).JSON(rd)
	}

	if err := base.Validator.Struct(&req); err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusUnprocessableEntity, "error", "Validation failed", utility.ValidationErrorsToJSON(err, dtos.AggregatorLoginDto{}), nil)
		return c.Status(fiber.StatusUnprocessableEntity).JSON(rd)
	}

	// Make sure we select the right DB
	activeDb := middleware.GetDatabaseInstance(req.IsSandbox, base.Db, base.TestDB)

	data, status, err := aggregatorSvc.LoginAggregator(req, activeDb)
	if err != nil {
		rd := utility.BuildErrorResponse(status, "error", err.Error(), err, nil)
		return c.Status(status).JSON(rd)
	}

	rd := utility.BuildSuccessResponse(status, "Login successful", data)
	return c.Status(status).JSON(rd)
}

// Logout aggregator
func (base *Controller) Logout(c *fiber.Ctx) error {
	userDetails, err := middleware.GetUserDetails(c)
	if err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusUnauthorized, "error", "Unauthorized", nil, nil)
		return c.Status(fiber.StatusUnauthorized).JSON(rd)
	}

	activeDb := middleware.GetDatabaseInstance(userDetails.IsSandbox, base.Db, base.TestDB)

	status, err := aggregatorSvc.LogoutAggregator(userDetails.AccessUuid, userDetails.ID, activeDb)
	if err != nil {
		rd := utility.BuildErrorResponse(status, "error", err.Error(), err, nil)
		return c.Status(status).JSON(rd)
	}

	rd := utility.BuildSuccessResponse(status, "Logout successful", nil)
	return c.Status(status).JSON(rd)
}

// VerifyEmail aggregator
func (base *Controller) VerifyEmail(c *fiber.Ctx) error {
	var req dtos.AggregatorVerifyEmailDto
	// Email verification checks against test DB or handles correctly depending on DB choice
	db := middleware.GetDatabaseInstance(true, base.Db, base.TestDB)

	if err := c.BodyParser(&req); err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusBadRequest, "error", "Failed to parse request body", err, nil)
		return c.Status(fiber.StatusBadRequest).JSON(rd)
	}

	if err := base.Validator.Struct(&req); err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusUnprocessableEntity, "error", "Validation failed", utility.ValidationErrorsToJSON(err, dtos.AggregatorVerifyEmailDto{}), nil)
		return c.Status(fiber.StatusUnprocessableEntity).JSON(rd)
	}

	data, err := aggregatorSvc.VerifyAggregatorEmail(db, req, true)
	if err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusBadRequest, "error", err.Error(), err, nil)
		return c.Status(fiber.StatusBadRequest).JSON(rd)
	}

	rd := utility.BuildSuccessResponse(fiber.StatusOK, "Email verified successfully", data)
	return c.Status(fiber.StatusOK).JSON(rd)
}

// ResendOTP aggregator
func (base *Controller) ResendOTP(c *fiber.Ctx) error {
	var req dtos.AggregatorResendOtpDto
	db := middleware.GetDatabaseInstance(true, base.Db, base.TestDB)

	if err := c.BodyParser(&req); err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusBadRequest, "error", "Failed to parse request body", err, nil)
		return c.Status(fiber.StatusBadRequest).JSON(rd)
	}

	if err := base.Validator.Struct(&req); err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusUnprocessableEntity, "error", "Validation failed", utility.ValidationErrorsToJSON(err, dtos.AggregatorResendOtpDto{}), nil)
		return c.Status(fiber.StatusUnprocessableEntity).JSON(rd)
	}

	err := aggregatorSvc.ResendVerificationOTP(db, req.Email)
	if err != nil {
		rd := utility.BuildErrorResponse(fiber.StatusBadRequest, "error", err.Error(), err, nil)
		return c.Status(fiber.StatusBadRequest).JSON(rd)
	}

	rd := utility.BuildSuccessResponse(fiber.StatusOK, "OTP sent successfully", nil)
	return c.Status(fiber.StatusOK).JSON(rd)
}
