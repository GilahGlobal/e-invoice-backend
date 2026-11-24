package resend_email

import (
	"einvoice-access-point/templates"
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

	bodyHtml, err := templates.RenderEmail("aggregator_invitation.html", map[string]any{
		"BusinessName": businessName,
	})
	if err != nil {
		fmt.Printf("Failed to render aggregator_invitation.html: %v\n", err)
		return
	}

	if err := sendEmailInternal(aggregatorEmail, subject, bodyHtml); err != nil {
		fmt.Printf("Failed to send aggregator invitation email: %v\n", err)
	}
}

// SendInvitationAcceptedEmail sends an email to a business when an aggregator accepts their invite
func SendInvitationAcceptedEmail(businessEmail, aggregatorName string) {
	subject := fmt.Sprintf("Aggregator Invitation Accepted by %s", aggregatorName)

	bodyHtml, err := templates.RenderEmail("invitation_accepted.html", map[string]any{
		"AggregatorName": aggregatorName,
	})
	if err != nil {
		fmt.Printf("Failed to render invitation_accepted.html: %v\n", err)
		return
	}

	if err := sendEmailInternal(businessEmail, subject, bodyHtml); err != nil {
		fmt.Printf("Failed to send invitation accepted email: %v\n", err)
	}
}

// SendInvitationRejectedEmail sends an email to a business when an aggregator rejects their invite
func SendInvitationRejectedEmail(businessEmail, aggregatorName string) {
	subject := fmt.Sprintf("Aggregator Invitation Rejected by %s", aggregatorName)

	bodyHtml, err := templates.RenderEmail("invitation_rejected.html", map[string]any{
		"AggregatorName": aggregatorName,
	})
	if err != nil {
		fmt.Printf("Failed to render invitation_rejected.html: %v\n", err)
		return
	}

	if err := sendEmailInternal(businessEmail, subject, bodyHtml); err != nil {
		fmt.Printf("Failed to send invitation rejected email: %v\n", err)
	}
}

// SendNewAggregatorInvitationEmail sends an email to a new aggregator inviting them to register
func SendNewAggregatorInvitationEmail(aggregatorEmail, businessName, signupLink string) {
	subject := fmt.Sprintf("You have been invited to manage invoices for %s", businessName)

	bodyHtml, err := templates.RenderEmail("new_aggregator_invitation.html", map[string]any{
		"BusinessName": businessName,
		"SignupLink":   signupLink,
	})
	if err != nil {
		fmt.Printf("Failed to render new_aggregator_invitation.html: %v\n", err)
		return
	}

	if err := sendEmailInternal(aggregatorEmail, subject, bodyHtml); err != nil {
		fmt.Printf("Failed to send new aggregator invitation email: %v\n", err)
	}
}
