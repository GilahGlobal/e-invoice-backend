package resend_email

import (
	"fmt"

	"github.com/resend/resend-go/v3"
)

func sendEmailInternal(email, subject, bodyHtml string) error {
	client := getResendClient()

	params := &resend.SendEmailRequest{
		From:    "Nexar <noreply@einvoice.nexar.ng>",
		To:      []string{email},
		Html:    bodyHtml,
		Subject: subject,
	}

	sent, err := client.Emails.Send(params)
	if err != nil {
		return err
	}

	fmt.Println("Email sent:", sent.Id)
	return nil
}

// SendAggregatorInvitationEmail sends an email to an aggregator when a business invites them
func SendAggregatorInvitationEmail(aggregatorEmail, businessName string) {
	subject := fmt.Sprintf("You have been invited to manage invoices for %s", businessName)
	bodyHtml := fmt.Sprintf(`
		<h2>Aggregator Invitation</h2>
		<p>Hello,</p>
		<p><strong>%s</strong> has invited you to manage their invoices as an aggregator.</p>
		<p>Please log in to your Aggregator Portal to accept or reject this invitation.</p>
	`, businessName)

	if err := sendEmailInternal(aggregatorEmail, subject, bodyHtml); err != nil {
		fmt.Printf("Failed to send aggregator invitation email: %v\n", err)
	}
}

// SendInvitationAcceptedEmail sends an email to a business when an aggregator accepts their invite
func SendInvitationAcceptedEmail(businessEmail, aggregatorName string) {
	subject := fmt.Sprintf("Aggregator Invitation Accepted by %s", aggregatorName)
	bodyHtml := fmt.Sprintf(`
		<h2>Invitation Accepted</h2>
		<p>Hello,</p>
		<p>Your invitation to <strong>%s</strong> has been accepted.</p>
		<p>They can now upload and manage invoices on your behalf.</p>
	`, aggregatorName)

	if err := sendEmailInternal(businessEmail, subject, bodyHtml); err != nil {
		fmt.Printf("Failed to send invitation accepted email: %v\n", err)
	}
}

// SendInvitationRejectedEmail sends an email to a business when an aggregator rejects their invite
func SendInvitationRejectedEmail(businessEmail, aggregatorName string) {
	subject := fmt.Sprintf("Aggregator Invitation Rejected by %s", aggregatorName)
	bodyHtml := fmt.Sprintf(`
		<h2>Invitation Rejected</h2>
		<p>Hello,</p>
		<p>Unfortunately, <strong>%s</strong> has rejected your invitation to manage invoices.</p>
		<p>You can invite another aggregator from your dashboard.</p>
	`, aggregatorName)

	if err := sendEmailInternal(businessEmail, subject, bodyHtml); err != nil {
		fmt.Printf("Failed to send invitation rejected email: %v\n", err)
	}
}

// SendNewAggregatorInvitationEmail sends an email to a new aggregator inviting them to register
func SendNewAggregatorInvitationEmail(aggregatorEmail, businessName, signupLink string) {
	subject := fmt.Sprintf("You have been invited to manage invoices for %s", businessName)
	bodyHtml := fmt.Sprintf(`
		<h2>Aggregator Invitation</h2>
		<p>Hello,</p>
		<p><strong>%s</strong> has invited you to manage their invoices as an aggregator.</p>
		<p>It looks like you do not have an account with us yet. Please click the link below to create your account and accept the invitation.</p>
		<p><a href="%s" style="display:inline-block;padding:10px 20px;background-color:#007bff;color:#ffffff;text-decoration:none;border-radius:5px;">Create Account & Accept Invite</a></p>
		<p>If the button above does not work, copy and paste this link into your browser:</p>
		<p>%s</p>
	`, businessName, signupLink, signupLink)

	if err := sendEmailInternal(aggregatorEmail, subject, bodyHtml); err != nil {
		fmt.Printf("Failed to send new aggregator invitation email: %v\n", err)
	}
}
