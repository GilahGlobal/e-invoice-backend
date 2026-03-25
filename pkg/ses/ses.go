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
