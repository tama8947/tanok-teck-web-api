package cron

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/tanok/tanok-web-api/internal/config"
)

type DiscoveredTopic struct {
	ES          string
	EN          string
	SearchQuery string
}

func DiscoverTopics(ctx context.Context, cfg *config.AppConfig) ([]DiscoveredTopic, error) {
	if cfg.BraveAPIKey == "" {
		return nil, fmt.Errorf("BRAVE_API_KEY is not configured")
	}

	queries := []string{
		"software development trends 2025",
		"AI artificial intelligence latest news",
		"web development best practices",
		"cloud computing devops technology",
		"cybersecurity trends enterprise",
	}

	var allTopics []DiscoveredTopic

	for _, q := range queries {
		topics, err := searchBrave(ctx, cfg.BraveAPIKey, q)
		if err != nil {
			continue
		}
		allTopics = append(allTopics, topics...)
	}

	return allTopics, nil
}

func searchBrave(ctx context.Context, apiKey, query string) ([]DiscoveredTopic, error) {
	searchURL := fmt.Sprintf("https://api.search.brave.com/res/v1/web/search?q=%s&count=5",
		url.QueryEscape(query))

	req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Accept-Encoding", "gzip")
	req.Header.Set("X-Subscription-Token", apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	var result struct {
		Web struct {
			Results []struct {
				Title       string `json:"title"`
				Description string `json:"description"`
			} `json:"results"`
		} `json:"web"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("parse brave response: %w", err)
	}

	var topics []DiscoveredTopic
	for _, r := range result.Web.Results {
		if r.Title == "" || len(r.Title) < 10 {
			continue
		}
		cleanTitle := cleanTitle(r.Title)
		if len(cleanTitle) < 10 {
			continue
		}
		topics = append(topics, DiscoveredTopic{
			ES:          cleanTitle,
			EN:          translateToEnglish(cleanTitle),
			SearchQuery: query,
		})
	}

	return topics, nil
}

func cleanTitle(title string) string {
	result := make([]rune, 0, len(title))
	for _, r := range title {
		if r == '-' || r == '—' || r == '|' || r == '·' {
			break
		}
		result = append(result, r)
	}
	s := string(result)
	if len(s) > 200 {
		s = s[:200]
	}
	return s
}

func translateToEnglish(es string) string {
	common := map[string]string{
		"inteligencia artificial": "artificial intelligence",
		"desarrollo de software":  "software development",
		"tendencias":              "trends",
		"mejores prácticas":       "best practices",
		"seguridad":               "security",
		"empresarial":             "enterprise",
	}
	for esw, enw := range common {
		if containsWord(es, esw) {
			return replaceWord(es, esw, enw)
		}
	}
	return es + " (English topic)"
}

func containsWord(s, word string) bool {
	return len(s) >= len(word) && containsIgnoreCase(s, word)
}

func containsIgnoreCase(s, sub string) bool {
	if len(sub) > len(s) {
		return false
	}
	limit := len(sub)
	for i := 0; i <= len(s)-limit; i++ {
		match := true
		for j := 0; j < limit; j++ {
			a := s[i+j]
			b := sub[j]
			if a >= 'A' && a <= 'Z' {
				a += 32
			}
			if b >= 'A' && b <= 'Z' {
				b += 32
			}
			if a != b {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

func replaceWord(s, old, new string) string {
	result := make([]byte, 0, len(s)+len(new))
	i := 0
	limit := len(old)
	for i <= len(s)-limit {
		match := true
		for j := 0; j < limit; j++ {
			a := s[i+j]
			b := old[j]
			if a >= 'A' && a <= 'Z' {
				a += 32
			}
			if b >= 'A' && b <= 'Z' {
				b += 32
			}
			if a != b {
				match = false
				break
			}
		}
		if match {
			result = append(result, new...)
			i += limit
		} else {
			result = append(result, s[i])
			i++
		}
	}
	result = append(result, s[i:]...)
	return string(result)
}

func PersistDiscoveredTopics(ctx context.Context, db *pgxpool.Pool, topics []DiscoveredTopic) error {
	for _, t := range topics {
		db.Exec(ctx,
			`INSERT INTO "CronTopic" (id, "esText", "enText", "searchQuery", "usedAt", "useCount", "isActive", "createdAt")
			 VALUES ($1, $2, $3, $4, NULL, 0, true, NOW())
			 ON CONFLICT DO NOTHING`,
			generateID("ct_"), t.ES, t.EN, sql.NullString{String: t.SearchQuery, Valid: t.SearchQuery != ""},
		)

		// Sleep briefly to avoid CUID collisions
		time.Sleep(10 * time.Millisecond)
	}
	return nil
}


