package agents

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

type SEOAgentResult struct {
	LinksCreated int      `json:"linksCreated"`
	SchemaTypes  []string `json:"schemaTypes"`
}

func RunSEOAgent(ctx context.Context, postID, title, content string, db *pgxpool.Pool) *SEOAgentResult {
	result := &SEOAgentResult{}

	keywords := extractKeywords(title + " " + content)

	rows, err := db.Query(ctx,
		`SELECT id, title, slug, locale FROM "Post"
		 WHERE id != $1 AND status = 'PUBLISHED'
		 LIMIT 20`,
		postID,
	)
	if err != nil {
		return result
	}
	defer rows.Close()

	for rows.Next() {
		var targetID, targetTitle, targetSlug, targetLocale string
		if err := rows.Scan(&targetID, &targetTitle, &targetSlug, &targetLocale); err != nil {
			continue
		}

		score := relevanceScore(keywords, targetTitle)
		if score > 0.3 {
			db.Exec(ctx,
				`INSERT INTO "PostLink" (id, "sourcePostId", "targetPostId", score, "createdAt")
				 VALUES ($1, $2, $3, $4, NOW())
				 ON CONFLICT ("sourcePostId", "targetPostId") DO UPDATE SET score = $4`,
				generateID("pl_"), postID, targetID, score,
			)
			result.LinksCreated++
		}
	}

	result.SchemaTypes = []string{"BlogPosting", "Article"}
	return result
}

func extractKeywords(text string) []string {
	words := strings.Fields(strings.ToLower(text))
	stopWords := map[string]bool{
		"the": true, "a": true, "an": true, "and": true, "or": true, "but": true,
		"in": true, "on": true, "at": true, "to": true, "for": true, "of": true,
		"with": true, "by": true, "from": true, "is": true, "are": true, "was": true,
		"were": true, "be": true, "been": true, "being": true, "have": true, "has": true,
		"had": true, "do": true, "does": true, "did": true, "will": true, "would": true,
		"could": true, "should": true, "may": true, "might": true, "can": true,
		"that": true, "this": true, "these": true, "those": true, "it": true,
		"its": true, "de": true, "la": true, "el": true, "los": true, "las": true,
		"un": true, "una": true, "y": true, "e": true, "o": true, "que": true,
		"en": true, "se": true, "no": true, "su": true, "por": true, "como": true,
	}
	freq := make(map[string]int)
	for _, w := range words {
		w = strings.Trim(w, ".,;:!?\"'()[]{}")
		if len(w) < 3 || stopWords[w] {
			continue
		}
		freq[w]++
	}
	var keywords []string
	for k, v := range freq {
		if v >= 2 {
			keywords = append(keywords, k)
		}
	}
	return keywords
}

func relevanceScore(keywords []string, targetTitle string) float64 {
	if len(keywords) == 0 {
		return 0
	}
	titleLower := strings.ToLower(targetTitle)
	matches := 0
	for _, kw := range keywords {
		if strings.Contains(titleLower, kw) {
			matches++
		}
	}
	return float64(matches) / float64(len(keywords))
}

func generateID(prefix string) string {
	b := make([]byte, 12)
	for i := range b {
		b[i] = byte(i * 17 % 256)
	}
	return fmt.Sprintf("%s%x", prefix, b)
}

var _ = sql.NullString{}
