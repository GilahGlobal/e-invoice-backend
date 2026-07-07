package middleware

import (
	"einvoice-access-point/internal/apperror"
	"einvoice-access-point/internal/utility"
	"errors"

	"github.com/gofiber/fiber/v2"
)

// GlobalErrorHandler is the custom error handler for Fiber
var GlobalErrorHandler = func(c *fiber.Ctx, err error) error {
	// Default to 500 Internal Server Error
	statusCode := fiber.StatusInternalServerError
	status := "error"
	message := "Internal Server Error"
	var errDetail interface{}
	var data interface{}

	// Check if it's a custom AppError
	var appErr *apperror.AppError
	if errors.As(err, &appErr) {
		statusCode = appErr.StatusCode
		status = appErr.Status
		message = appErr.Message
		errDetail = appErr.Err
		if errDetail != nil {
			if e, ok := errDetail.(error); ok {
				errDetail = e.Error()
			}
		}
		data = appErr.Data
	} else {
		// Handle generic Fiber errors
		var fiberErr *fiber.Error
		if errors.As(err, &fiberErr) {
			statusCode = fiberErr.Code
			message = fiberErr.Message
		} else {
			message = err.Error()
		}
	}

	rd := utility.BuildErrorResponse(statusCode, status, message, errDetail, data)

	// Return the response as JSON
	return c.Status(statusCode).JSON(rd)
}
