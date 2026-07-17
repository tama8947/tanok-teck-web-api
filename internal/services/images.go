package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func (s *Services) GenerateCoverImage(ctx context.Context, title, locale string) (string, error) {
	if s.Config.MiniMaxAPIKey == "" {
		return "", fmt.Errorf("MINIMAX_API_KEY is not configured")
	}

	prompt := fmt.Sprintf(
		"A professional tech blog cover image for an article titled '%s'. Modern, clean design with tech/coding motifs. Blue and purple color scheme. No text on the image.",
		title,
	)

	payload := map[string]interface{}{
		"model":  "image-01",
		"prompt": prompt,
		"n":      1,
		"size":   "1024x1024",
	}

	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.minimax.io/v1/image_generation", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+s.Config.MiniMaxAPIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	var imgResult struct {
		Data []struct {
			URL string `json:"url"`
		} `json:"data"`
	}
	if err := json.Unmarshal(respBody, &imgResult); err != nil {
		return "", fmt.Errorf("parse image response: %w", err)
	}
	if len(imgResult.Data) == 0 || imgResult.Data[0].URL == "" {
		return "", fmt.Errorf("no image URL in response: %s", string(respBody))
	}

	imageURL := imgResult.Data[0].URL

	imageData, err := downloadImage(ctx, imageURL)
	if err != nil {
		return "", fmt.Errorf("download image: %w", err)
	}

	publicURL, err := s.uploadToR2(ctx, title, imageData)
	if err != nil {
		return "", fmt.Errorf("upload to R2: %w", err)
	}

	return publicURL, nil
}

func downloadImage(ctx context.Context, url string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

func (s *Services) uploadToR2(ctx context.Context, title string, data []byte) (string, error) {
	accessKey := s.Config.R2AccessKeyID
	secretKey := s.Config.R2SecretAccessKey
	bucket := s.Config.R2BucketName
	endpoint := s.Config.R2Endpoint
	region := s.Config.R2Region

	if endpoint == "" {
		endpoint = fmt.Sprintf("https://%s.r2.cloudflarestorage.com", s.Config.R2AccessKeyID)
	}
	if region == "" {
		region = "auto"
	}

	customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		return aws.Endpoint{
			URL:               endpoint,
			SigningRegion:     region,
			HostnameImmutable: true,
		}, nil
	})

	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")),
		config.WithEndpointResolverWithOptions(customResolver),
		config.WithRegion(region),
	)
	if err != nil {
		return "", fmt.Errorf("aws config: %w", err)
	}

	client := s3.NewFromConfig(cfg)

	filename := fmt.Sprintf("covers/%s/%s_%d.png", title[:min(20, len(title))], sanitizeFilename(title), time.Now().Unix())

	_, err = client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(bucket),
		Key:         aws.String(filename),
		Body:        bytes.NewReader(data),
		ContentType: aws.String("image/png"),
	})
	if err != nil {
		return "", fmt.Errorf("s3 put: %w", err)
	}

	publicURL := s.Config.R2PublicURL
	if publicURL == "" {
		publicURL = fmt.Sprintf("%s/%s", endpoint, bucket)
	}
	return fmt.Sprintf("%s/%s", publicURL, filename), nil
}

func sanitizeFilename(s string) string {
	result := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-' || c == '_' {
			result = append(result, c)
		} else if c == ' ' {
			result = append(result, '-')
		}
	}
	if len(result) == 0 {
		return "image"
	}
	return string(result)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
