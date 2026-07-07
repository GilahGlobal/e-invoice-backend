package routes

import (
	"einvoice-access-point/internal/app/auth"
	"einvoice-access-point/internal/core"
	"einvoice-access-point/internal/middleware"
	"fmt"

	"github.com/gofiber/fiber/v2"
)

func AuthRoute(app *fiber.App, ApiVersion string, container *core.Container) *fiber.App {
	authController := auth.NewHandler(container.Validator, container.Logger, container.Config, container.DB, container.TestDB)

	authGroup := app.Group(fmt.Sprintf("%v/auth", ApiVersion))
	authGroup.Post("/login", authController.Login)
	authGroup.Post("/register", authController.Register)
	authGroup.Post("/resend-otp", authController.ResendVerificationOTP)
	authGroup.Post("/verify-email", authController.VerifyEmail)
	authGroup.Post("/initiate-forgot-password", authController.InitiateForgotPassword)
	authGroup.Post("/complete-forgot-password", authController.CompleteForgotPassword)

	authUrlSec := app.Group(fmt.Sprintf("%v/auth", ApiVersion), middleware.Authorize(container.DB.Postgresql.DB(), container.TestDB.Postgresql.DB()))
	{
		authUrlSec.Get("/logout", authController.Logout)
		authUrlSec.Post("/register1", authController.Register)
		authUrlSec.Get("/toggle-mode", authController.ToggleApplicationMode)
		authUrlSec.Post("/change-password", authController.ChangePassword)
	}

	return app
}
