package resend_email

import (
	"fmt"
)

// SendAdminCreatedEmail sends an email to a newly created admin with their login credentials
func SendAdminCreatedEmail(email, password string) {
	subject := "Your Admin Account Has Been Created"
	bodyHtml := fmt.Sprintf(`
		<h2>Admin Account Created</h2>
		<p>Hello,</p>
		<p>An administrator account has been created for you.</p>
		<p>You can log in using the following credentials:</p>
		<ul>
			<li><strong>Email:</strong> %s</li>
			<li><strong>Password:</strong> %s</li>
		</ul>
		<p>Please log in and change your password as soon as possible.</p>
	`, email, password)

	if err := sendEmailInternal(email, subject, bodyHtml); err != nil {
		fmt.Printf("Failed to send admin created email: %v\n", err)
	}
}
