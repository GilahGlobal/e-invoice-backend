package resend_email

import (
	"fmt"
	"log"

	internalConfig "einvoice-access-point/pkg/config"

	"github.com/resend/resend-go/v3"
)

func getResendClient() *resend.Client {
	apiKey := internalConfig.Config.Mail.ResendApiKey
	return resend.NewClient(apiKey)
}

func SendEmail(email, otp string) {
	client := getResendClient()

	params := &resend.SendEmailRequest{
		From:    "noreply@einvoice.nexar.ng",
		To:      []string{email},
		Html:    fmt.Sprintf("<h1>Otp: %s </h1>", otp),
		Subject: "Your OTP Code",
	}

	sent, err := client.Emails.Send(params)
	if err != nil {
		panic(err)
	}

	fmt.Println("Email sent:", sent.Id)
}

type EmailRequest struct {
	To      string `json:"to"`
	Subject string `json:"subject"`
	HTML    string `json:"html"`
}

func Send(email, otp string) error {
	client := getResendClient()

	params := &resend.SendEmailRequest{
		From: "no-reply <noreply@einvoice.nexar.ng>",
		To:   []string{email},
		Html: fmt.Sprintf(`
			<h2>Verification Code</h2>
			<p>Your OTP is:</p>
			<h1>%s</h1>
			<p>This code will expire shortly.</p>
		`, otp),
		Subject: "Your OTP Code",
	}

	a, err := client.Emails.Send(params)
	if err != nil {
		log.Println("err: ", err)
		return err
	}
	log.Println("response: ", *a)

	return nil
}
