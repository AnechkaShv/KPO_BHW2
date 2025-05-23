package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
)

type S3Client struct {
	s3Client  *s3.Client
	bucket    string
	urlPrefix string
}

func NewS3Client(client *s3.Client, bucket, urlPrefix string) *S3Client {
	return &S3Client{
		s3Client:  client,
		bucket:    bucket,
		urlPrefix: urlPrefix,
	}
}

func (c *S3Client) StoreWordCloud(ctx context.Context, mimeType string, data []byte) (string, error) {
	sha256Hash := sha256.Sum256(data)
	s3Key := uuid.NewString()

	_, err := c.s3Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:         aws.String(c.bucket),
		Key:            aws.String(s3Key),
		Body:           bytes.NewReader(data),
		ContentType:    aws.String(mimeType),
		ChecksumSHA256: aws.String(base64.StdEncoding.EncodeToString(sha256Hash[:])),
	})
	if err != nil {
		return "", fmt.Errorf("failed to put object to s3: %w", err)
	}

	return c.urlPrefix + s3Key, nil
}
