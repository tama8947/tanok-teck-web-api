package cron

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/tanok/tanok-web-api/internal/config"
)

type IndexNowResult struct {
	Submitted   int      `json:"submitted"`
	RateLimited bool     `json:"rateLimited"`
	Message     string   `json:"message"`
	URLs        []string `json:"urls"`
}

func SubmitToIndexNow(ctx context.Context, cfg *config.AppConfig, urls []string) (*IndexNowResult, error) {
	if len(urls) == 0 {
		return &IndexNowResult{Submitted: 0}, nil
	}

	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = "https://tanok-tech.com"
	}

	host := extractHost(baseURL)
	indexNowURL := fmt.Sprintf("https://api.indexnow.org/indexnow?key=%s&keyLocation=%s/key.txt", host, baseURL)

	payload := map[string]interface{}{
		"host":        host,
		"key":         host,
		"keyLocation": fmt.Sprintf("%s/key.txt", baseURL),
		"urlList":     urls,
	}

	body, _ := json.Marshal(payload)

	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST", indexNowURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("indexnow request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("indexnow http: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	result := &IndexNowResult{
		URLs: urls,
	}

	switch resp.StatusCode {
	case 200, 202:
		result.Submitted = len(urls)
		result.Message = "URLs submitted successfully"
	case 429:
		result.RateLimited = true
		result.Message = "Rate limited. URLs not submitted."
	case 403:
		result.RateLimited = true
		result.Message = fmt.Sprintf("Forbidden: %s", string(respBody))
	default:
		result.Message = fmt.Sprintf("IndexNow returned status %d: %s", resp.StatusCode, string(respBody))
	}

	return result, nil
}

func extractHost(baseURL string) string {
	s := baseURL
	if len(s) > 8 && s[:8] == "https://" {
		s = s[8:]
	} else if len(s) > 7 && s[:7] == "http://" {
		s = s[7:]
	}
	for i := 0; i < len(s); i++ {
		if s[i] == '/' || s[i] == ':' {
			return s[:i]
		}
	}
	return s
}
