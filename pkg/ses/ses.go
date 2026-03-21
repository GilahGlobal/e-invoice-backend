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
	// creds := aws.NewCredentialsCache(
	// 	credentials.NewStaticCredentialsProvider(
	// 		internalConfig.Config.S3.AccessKeyID,
	// 		internalConfig.Config.S3.SecretAccessKey,
	// 		"",
	// 	),
	// )

	// // Build AWS config
	// cfg := aws.Config{
	// 	Region:      internalConfig.Config.S3.Region,
	// 	Credentials: creds,
	// }
	cfg, err := config.LoadDefaultConfig(
		context.TODO(),
		config.WithRegion(internalConfig.Config.S3.Region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			internalConfig.Config.S3.AccessKeyID,
			internalConfig.Config.S3.SecretAccessKey,
			"optional",
		)),
	)

	// cfg, err := config.LoadDefaultConfig(context.TODO(),
	// 	config.WithRegion("us-east-1"),
	// )

	if err != nil {
		panic(err)
	}

	client := ses.NewFromConfig(cfg)

	input := &ses.SendEmailInput{
		Source: aws.String("joel@gention.tech"),
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
