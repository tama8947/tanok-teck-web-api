package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/tanok/tanok-web-api/internal/config"
)

type GeneratedPost struct {
	Title          string `json:"title"`
	Slug           string `json:"slug"`
	Excerpt        string `json:"excerpt"`
	Content        string `json:"content"`
	MetaTitle      string `json:"metaTitle"`
	MetaDescription string `json:"metaDescription"`
}

type deepseekResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

func GeneratePostContent(ctx context.Context, cfg *config.AppConfig, topic, locale, model string) (*GeneratedPost, error) {
	langName := "Spanish"
	if locale == "en" {
		langName = "English"
	}

	systemPrompt := fmt.Sprintf(`You are a professional tech blogger for Tanok Tech, a software development and AI consulting company. Write a comprehensive, well-researched blog post in %s.

Your response must be VALID JSON with exactly these fields:
- title: An engaging, SEO-friendly title
- slug: A URL-friendly version of the title (lowercase, hyphens)
- excerpt: A 2-3 sentence summary that hooks the reader (max 250 chars)
- content: Full blog post in Markdown format with H2/H3 headings, code blocks, bullet points. At least 1500 words.
- metaTitle: SEO title (max 60 chars)
- metaDescription: SEO meta description (max 160 chars)

Guidelines:
- Write in a professional but engaging tone
- Include practical examples and actionable insights
- Use real technical data and statistics when possible
- Structure with clear headings and subheadings
- End with a conclusion and call to action

IMPORTANT: Return ONLY the JSON object, no markdown fences, no extra text.`, langName)

	payload := map[string]interface{}{
		"model": model,
		"messages": []map[string]string{
			{"role": "system", "content": systemPrompt},
			{"role": "user", "content": fmt.Sprintf("Write a blog post about: %s", topic)},
		},
		"response_format": map[string]string{"type": "json_object"},
		"temperature":     0.7,
		"max_tokens":      4096,
	}

	body, _ := json.Marshal(payload)
	apiURL := cfg.DeepSeekAPIURL
	if apiURL == "" {
		apiURL = "https://api.deepseek.com/v1/chat/completions"
	}

	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("deepseek request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+cfg.DeepSeekAPIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("deepseek call: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("deepseek error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var dsResp deepseekResponse
	if err := json.Unmarshal(respBody, &dsResp); err != nil {
		return nil, fmt.Errorf("parse deepseek response: %w", err)
	}

	if len(dsResp.Choices) == 0 {
		return nil, fmt.Errorf("no choices in deepseek response")
	}

	content := dsResp.Choices[0].Message.Content

	var post GeneratedPost
	if err := json.Unmarshal([]byte(content), &post); err != nil {
		return nil, fmt.Errorf("parse post JSON from deepseek: %w (raw: %s)", err, content[:min(200, len(content))])
	}

	if post.Title == "" {
		return nil, fmt.Errorf("generated post has empty title")
	}
	if post.Slug == "" {
		post.Slug = slugify(post.Title)
	}

	return &post, nil
}

func slugify(s string) string {
	result := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 32
		}
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') {
			result = append(result, c)
		} else if c == ' ' || c == '-' || c == '_' {
			result = append(result, '-')
		}
	}
	if len(result) == 0 {
		return "untitled"
	}
	return string(result)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func init() {
	_ = min
}
