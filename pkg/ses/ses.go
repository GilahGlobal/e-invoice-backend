package ses

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	internalConfig "einvoice-access-point/pkg/config"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/ses"
	"github.com/aws/aws-sdk-go-v2/service/ses/types"
)

func SendEmail(email, otp string) {
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
		panic(err)
	}

	client := ses.NewFromConfig(cfg)

	input := &ses.SendEmailInput{
		Source: aws.String("devops@nexar.ng"),
		Destination: &types.Destination{
			ToAddresses: []string{"devops@nexar.ng"},
		},
		Message: &types.Message{
			Subject: &types.Content{
				Data: aws.String("Hello from Go SES"),
			},
			Body: &types.Body{
				Html: &types.Content{
					Data: aws.String(
						fmt.Sprintf("<h1>Otp: %s </h1>", otp),
					),
				},
			},
		},
	}

	result, err := client.SendEmail(context.TODO(), input)
	if err != nil {
		panic(err)
	}

	fmt.Println("Email sent:", result.MessageId)
}

type EmailRequest struct {
	To      string `json:"to"`
	Subject string `json:"subject"`
	HTML    string `json:"html"`
}

func Send(email, otp string) error {
	url := "https://ami-portal-backend-smdw.onrender.com/email/send"

	payload := EmailRequest{
		To:      email,
		Subject: "Your OTP Code",
		HTML: fmt.Sprintf(`
			<h2>Verification Code</h2>
			<p>Your OTP is:</p>
			<h1>%s</h1>
			<p>This code will expire shortly.</p>
		`, otp),
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("failed to send email, status: %s", resp.Status)
	}

	return nil
}
