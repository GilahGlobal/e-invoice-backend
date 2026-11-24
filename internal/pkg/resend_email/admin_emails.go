package resend_email

import (
	"einvoice-access-point/templates"
	"fmt"
)

// SendAdminCreatedEmail sends an email to a newly created admin with their login credentials
func SendAdminCreatedEmail(email, password string) {
	subject := "Your Admin Account Has Been Created"

	bodyHtml, err := templates.RenderEmail("admin_created.html", map[string]any{
		"Email":    email,
		"Password": password,
	})
	if err != nil {
		fmt.Printf("Failed to render admin_created.html template: %v\n", err)
		return
	}

	if err := sendEmailInternal(email, subject, bodyHtml); err != nil {
		fmt.Printf("Failed to send admin created email: %v\n", err)
	}
}
