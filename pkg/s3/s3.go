package s3

import (
	"context"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"path/filepath"
	"strings"
	"sync"

	internalConfig "einvoice-access-point/pkg/config"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// Upload handles direct file upload to S3 bucket
func UploadFileToS3(ctx context.Context, file multipart.File, fileHeader *multipart.FileHeader) (string, string, error) {
	// The session the S3 Uploader will use
	folder := "invoices" // specify your folder name here

	cfg, err := config.LoadDefaultConfig(
		ctx,
		config.WithRegion(internalConfig.Config.S3.Region),
		config.WithCredentialsProvider(credentials.StaticCredentialsProvider{
			Value: aws.Credentials{
				AccessKeyID:     internalConfig.Config.S3.AccessKeyID,
				SecretAccessKey: internalConfig.Config.S3.SecretAccessKey,
			},
		}))

	if err != nil {
		log.Println(err)
		log.Fatal(err)
	}
	client := s3.NewFromConfig(cfg)

	fileKey := fmt.Sprintf("%s/%s", folder, filepath.Base(strings.ReplaceAll(fileHeader.Filename, " ", "_")))

	// Upload the file
	_, err = client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(internalConfig.Config.S3.Bucket),
		Key:         aws.String(fileKey),
		Body:        file,
		ContentType: aws.String(fileHeader.Header.Get("Content-Type")),
	})
	if err != nil {
		return "", "", fmt.Errorf("failed to upload file to S3: %v", err)
	}

	// Construct the file URL
	fileURL := fmt.Sprintf("https://%s.s3.amazonaws.com/%s", internalConfig.Config.S3.Bucket, fileKey)
	return fileURL, fileKey, nil
}

func DownloadFileFromS3(ctx context.Context, fileKey string) ([]byte, error) {
	cfg, err := config.LoadDefaultConfig(
		ctx,
		config.WithRegion(internalConfig.Config.S3.Region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			internalConfig.Config.S3.AccessKeyID,
			internalConfig.Config.S3.SecretAccessKey,
			"",
		)),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	client := s3.NewFromConfig(cfg)

	// Get file size
	headOutput, err := client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(internalConfig.Config.S3.Bucket),
		Key:    aws.String(fileKey),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get file size: %w", err)
	}

	fileSize := *headOutput.ContentLength

	// Choose chunk size based on file size
	chunkSize := int64(5 * 1024 * 1024) // 5MB chunks
	if fileSize < chunkSize {
		// Small file, download directly
		return DownloadDirect(ctx, client, fileKey)
	}

	// Calculate number of chunks
	numChunks := (fileSize + chunkSize - 1) / chunkSize
	log.Printf("Downloading %d bytes in %d chunks", fileSize, numChunks)

	// Create buffer for final result
	result := make([]byte, fileSize)

	// Download chunks in parallel
	var wg sync.WaitGroup
	errChan := make(chan error, numChunks)

	for i := int64(0); i < numChunks; i++ {
		wg.Add(1)
		go func(chunkIndex int64) {
			defer wg.Done()

			start := chunkIndex * chunkSize
			end := start + chunkSize - 1
			if end >= fileSize {
				end = fileSize - 1
			}

			// Download this chunk
			data, err := DownloadRange(ctx, client, fileKey, start, end)
			if err != nil {
				errChan <- fmt.Errorf("chunk %d failed: %w", chunkIndex, err)
				return
			}

			// Copy chunk to correct position in result
			copy(result[start:start+int64(len(data))], data)
		}(i)
	}

	// Wait for all chunks
	wg.Wait()
	close(errChan)

	// Check for errors
	if len(errChan) > 0 {
		return nil, <-errChan
	}

	return result, nil
}

// DownloadRange downloads a specific byte range
func DownloadRange(ctx context.Context, client *s3.Client, fileKey string, start, end int64) ([]byte, error) {
	rangeHeader := fmt.Sprintf("bytes=%d-%d", start, end)

	getObjectOutput, err := client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(internalConfig.Config.S3.Bucket),
		Key:    aws.String(fileKey),
		Range:  aws.String(rangeHeader),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to download range %s: %w", rangeHeader, err)
	}
	defer getObjectOutput.Body.Close()

	return io.ReadAll(getObjectOutput.Body)
}

// DownloadDirect downloads entire file at once
func DownloadDirect(ctx context.Context, client *s3.Client, fileKey string) ([]byte, error) {
	getObjectOutput, err := client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(internalConfig.Config.S3.Bucket),
		Key:    aws.String(fileKey),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to download file: %w", err)
	}
	defer getObjectOutput.Body.Close()

	return io.ReadAll(getObjectOutput.Body)
}
