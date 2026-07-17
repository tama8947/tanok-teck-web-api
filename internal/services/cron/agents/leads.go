package agents

import (
	"context"
	"database/sql"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

type LeadsAgentResult struct {
	MagnetID string `json:"magnetId"`
}

func RunLeadsAgent(ctx context.Context, title, content string, db *pgxpool.Pool) *LeadsAgentResult {
	result := &LeadsAgentResult{}

	keywords := extractLeadKeywords(title + " " + content)
	if len(keywords) == 0 {
		return result
	}

	rows, err := db.Query(ctx,
		`SELECT id, name, slug FROM "LeadMagnet" LIMIT 20`,
	)
	if err != nil {
		return result
	}
	defer rows.Close()

	var bestMatch struct {
		id   string
		name string
	}
	bestScore := 0
	for rows.Next() {
		var magnetID, magnetName, magnetSlug string
		if err := rows.Scan(&magnetID, &magnetName, &magnetSlug); err != nil {
			continue
		}
		score := matchScore(keywords, magnetName+" "+magnetSlug)
		if score > bestScore {
			bestScore = score
			bestMatch.id = magnetID
			bestMatch.name = magnetName
		}
	}

	if bestScore > 0 {
		result.MagnetID = bestMatch.id
	}

	return result
}

func extractLeadKeywords(text string) []string {
	aiKeywords := []string{
		"ai", "artificial intelligence", "inteligencia artificial", "machine learning",
		"llm", "chatbot", "agent", "agente", "automation", "automatizacion",
		"software", "development", "desarrollo", "web", "app", "api",
		"consulting", "consultoria", "cloud", "devops", "security", "seguridad",
	}
	textLower := strings.ToLower(text)
	var found []string
	for _, kw := range aiKeywords {
		if strings.Contains(textLower, kw) {
			found = append(found, kw)
		}
	}
	return found
}

func matchScore(keywords []string, magnetText string) int {
	score := 0
	lower := strings.ToLower(magnetText)
	for _, kw := range keywords {
		if strings.Contains(lower, kw) {
			score++
		}
	}
	return score
}

var _ = sql.NullString{}
