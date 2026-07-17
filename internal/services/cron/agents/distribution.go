package agents

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

type DistributionResult struct {
	SnippetLen int      `json:"snippetLen"`
	Hashtags   []string `json:"hashtags"`
}

func RunDistributionAgent(ctx context.Context, title, content, postID string, db *pgxpool.Pool) *DistributionResult {
	result := &DistributionResult{}

	snippet := generateLinkedInSnippet(title, content)
	hashtags := generateHashtags(title, content)

	result.SnippetLen = len(snippet)
	result.Hashtags = hashtags

	db.Exec(ctx,
		`UPDATE "Post" SET "linkedinSnippet" = $1, "linkedinHashtags" = $2 WHERE id = $3`,
		sql.NullString{String: snippet, Valid: true},
		hashtags,
		postID,
	)

	return result
}

func generateLinkedInSnippet(title, content string) string {
	snippet := fmt.Sprintf("%s\n\n", title)

	sentences := extractSentences(content)
	added := 0
	for _, s := range sentences {
		if added >= 3 || len(snippet) > 800 {
			break
		}
		snippet += s + " "
		added++
	}

	if len(snippet) > 1200 {
		snippet = snippet[:1200] + "..."
	}

	snippet += "\n\nRead the full article on tanok-tech.com/blog"
	return strings.TrimSpace(snippet)
}

func extractSentences(text string) []string {
	var sentences []string
	current := strings.Builder{}
	for _, r := range text {
		current.WriteRune(r)
		if r == '.' || r == '!' || r == '?' {
			s := strings.TrimSpace(current.String())
			if len(s) > 20 {
				sentences = append(sentences, s)
			}
			current.Reset()
		}
	}
	return sentences
}

func generateHashtags(title, content string) []string {
	fullText := strings.ToLower(title + " " + content)

	hashtagPool := map[string]string{
		"ai":                 "#AI",
		"artificial intelligence": "#ArtificialIntelligence",
		"machine learning":   "#MachineLearning",
		"deep learning":      "#DeepLearning",
		"llm":                "#LLM",
		"chatbot":            "#Chatbot",
		"agent":              "#AIAgents",
		"software":           "#SoftwareDevelopment",
		"web":                "#WebDevelopment",
		"api":                "#API",
		"cloud":              "#CloudComputing",
		"devops":             "#DevOps",
		"security":           "#CyberSecurity",
		"data":               "#DataScience",
		"javascript":         "#JavaScript",
		"typescript":         "#TypeScript",
		"python":             "#Python",
		"react":              "#ReactJS",
		"nextjs":             "#NextJS",
		"postgres":           "#PostgreSQL",
		"docker":             "#Docker",
		"kubernetes":         "#Kubernetes",
		"open source":        "#OpenSource",
		"coding":             "#Coding",
		"programming":        "#Programming",
		"tech":               "#Tech",
	}

	seen := map[string]bool{}
	var hashtags []string
	for key, tag := range hashtagPool {
		if strings.Contains(fullText, key) && !seen[tag] {
			hashtags = append(hashtags, tag)
			seen[tag] = true
		}
		if len(hashtags) >= 8 {
			break
		}
	}

	if len(hashtags) < 3 {
		hashtags = append(hashtags, "#Tech", "#Innovation", "#SoftwareEngineering")
	}

	return hashtags
}

var _ = sql.NullString{}
