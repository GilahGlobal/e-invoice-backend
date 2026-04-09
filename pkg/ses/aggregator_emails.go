package ses

import (
	"context"
	"fmt"

	internalConfig "einvoice-access-point/pkg/config"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/ses"
	"github.com/aws/aws-sdk-go-v2/service/ses/types"
)

func getSESClient() (*ses.Client, error) {
	cfg, err := config.LoadDefaultConfig(
		context.TODO(),
		config.WithRegion(internalConfig.Config.S3.Region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			internalConfig.Config.S3.AccessKeyID,
			internalConfig.Config.S3.SecretAccessKey,
			"",
		)),
	)
	if err != nil {
		return nil, err
	}
	return ses.NewFromConfig(cfg), nil
}

func sendEmailInternal(email, subject, bodyHtml string) error {
	client, err := getSESClient()
	if err != nil {
		return err
	}

	input := &ses.SendEmailInput{
		Source: aws.String("devops@nexar.ng"),
		Destination: &types.Destination{
			ToAddresses: []string{email},
		},
		Message: &types.Message{
			Subject: &types.Content{
				Data: aws.String(subject),
			},
			Body: &types.Body{
				Html: &types.Content{
					Data: aws.String(bodyHtml),
				},
			},
		},
	}

	result, err := client.SendEmail(context.TODO(), input)
	if err != nil {
		return err
	}

	fmt.Println("Email sent:", result.MessageId)
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
