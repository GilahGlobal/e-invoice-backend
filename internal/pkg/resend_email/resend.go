package resend_email

import (
	internalConfig "einvoice-access-point/internal/config"
	"einvoice-access-point/templates"
	"fmt"
	"log"

	"github.com/resend/resend-go/v3"
)

func getResendClient() *resend.Client {
	apiKey := internalConfig.Config.Mail.ResendApiKey
	return resend.NewClient(apiKey)
}

func SendEmail(email, otp string) {
	htmlBody, err := templates.RenderEmail("otp.html", map[string]any{
		"OTP": otp,
	})
	if err != nil {
		log.Printf("Failed to render otp.html template: %v", err)
		return
	}

	if err := sendEmailInternal(email, "Your OTP Code", htmlBody); err != nil {
		log.Printf("Failed to send OTP email: %v", err)
	}
}

type EmailRequest struct {
	To      string `json:"to"`
	Subject string `json:"subject"`
	HTML    string `json:"html"`
}

func Send(email, otp string) error {
	htmlBody, err := templates.RenderEmail("otp.html", map[string]any{
		"OTP": otp,
	})
	if err != nil {
		return fmt.Errorf("failed to render otp.html template: %w", err)
	}

	return sendEmailInternal(email, "Your OTP Code", htmlBody)
}
