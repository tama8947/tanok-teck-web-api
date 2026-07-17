package cron

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/tanok/tanok-web-api/internal/config"
	"github.com/tanok/tanok-web-api/internal/services"

	ai "github.com/tanok/tanok-web-api/internal/services/cron/ai"
	"github.com/tanok/tanok-web-api/internal/services/cron/agents"
)

type RunDailyOptions struct {
	Now              *time.Time
	SkipIndexNow     bool
	Force            bool
	DryRun           bool
	AuthorIDOverride string
	TopicOverride    *TopicPair
}

type CronRunSummary struct {
	Slot       string
	Status     string
	Topic      TopicPair
	ESPostID   string
	ENPostID   string
	ESSlug     string
	ENSlug     string
	IndexNow   *IndexNowResult
	DurationMs int64
	Error      string
}

func FormatSlotKey(date time.Time) string {
	return date.UTC().Format("2006-01-02-15")
}

func RunDailyGeneration(ctx context.Context, svc *services.Services, opts RunDailyOptions) *CronRunSummary {
	start := time.Now()
	now := time.Now()
	if opts.Now != nil {
		now = *opts.Now
	}
	slot := FormatSlotKey(now)

	var topic TopicPair
	if opts.TopicOverride != nil {
		topic = *opts.TopicOverride
	} else {
		picked, err := PickNextTopic(ctx, svc.DB, svc.Config, svc.Redis)
		if err != nil {
			return &CronRunSummary{
				Slot:       slot,
				Status:     "FAILED",
				DurationMs: time.Since(start).Milliseconds(),
				Error:      fmt.Sprintf("topic selection failed: %v", err),
			}
		}
		topic = TopicPair{ES: picked.ES, EN: picked.EN}
	}

	base := &CronRunSummary{
		Slot:       slot,
		Status:     "FAILED",
		Topic:      topic,
		DurationMs: 0,
	}

	if !opts.Force {
		var status string
		err := svc.DB.QueryRow(ctx,
			`SELECT status FROM "CronRun" WHERE slot = $1`, slot,
		).Scan(&status)
		if err == nil && status == "SUCCESS" {
			base.Status = "SKIPPED"
			base.DurationMs = time.Since(start).Milliseconds()
			return base
		}
	}

	authorID := opts.AuthorIDOverride
	if authorID == "" {
		var err error
		authorID, err = svc.ResolveDefaultAuthorID(ctx)
		if err != nil {
			base.Status = "FAILED"
			base.DurationMs = time.Since(start).Milliseconds()
			base.Error = err.Error()
			recordCronFailure(ctx, svc.DB, slot, base.Error, opts.DryRun)
			return base
		}
	}

	// Generate ES post
	esContent, err := ai.GeneratePostContent(ctx, svc.Config, topic.ES, "es", "deepseek-chat")
	if err != nil {
		base.Status = "FAILED"
		base.DurationMs = time.Since(start).Milliseconds()
		base.Error = fmt.Sprintf("ES generation failed: %v", err)
		recordCronFailure(ctx, svc.DB, slot, base.Error, opts.DryRun)
		return base
	}

	esSlug := ensureUniqueSlug(ctx, svc.DB, esContent.Slug, "es")
	esPostID := generateID("post_")
	publishedAt := time.Now()
	_, err = svc.DB.Exec(ctx,
		`INSERT INTO "Post" (id, title, slug, excerpt, content, status, "publishedAt", "authorId", locale,
		 "metaTitle", "metaDescription", "generatedBy", "createdAt", "updatedAt")
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,NOW(),NOW())`,
		esPostID, esContent.Title, esSlug, sql.NullString{String: esContent.Excerpt, Valid: true},
		esContent.Content, "PUBLISHED", publishedAt, authorID, "es",
		sql.NullString{String: esContent.MetaTitle, Valid: esContent.MetaTitle != ""},
		sql.NullString{String: esContent.MetaDescription, Valid: esContent.MetaDescription != ""},
		sql.NullString{String: "deepseek", Valid: true},
	)
	if err != nil {
		base.Status = "FAILED"
		base.DurationMs = time.Since(start).Milliseconds()
		base.Error = fmt.Sprintf("ES post insert: %v", err)
		recordCronFailure(ctx, svc.DB, slot, base.Error, opts.DryRun)
		return base
	}
	base.ESPostID = esPostID
	base.ESSlug = esSlug

	// Generate EN post
	var enPostID, enSlug string
	enContent, err := ai.GeneratePostContent(ctx, svc.Config, topic.EN, "en", "deepseek-chat")
	if err != nil {
		log.Printf("[daily-posts] EN generation failed: %v", err)
		base.Error = fmt.Sprintf("EN generation failed: %v", err)
	} else {
		enSlug = ensureUniqueSlug(ctx, svc.DB, enContent.Slug, "en")
		enPostID = generateID("post_")
		_, err = svc.DB.Exec(ctx,
			`INSERT INTO "Post" (id, title, slug, excerpt, content, status, "publishedAt", "authorId", locale,
			 "metaTitle", "metaDescription", "generatedBy", "createdAt", "updatedAt")
			 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,NOW(),NOW())`,
			enPostID, enContent.Title, enSlug,
			sql.NullString{String: enContent.Excerpt, Valid: true},
			enContent.Content, "PUBLISHED", publishedAt, authorID, "en",
			sql.NullString{String: enContent.MetaTitle, Valid: enContent.MetaTitle != ""},
			sql.NullString{String: enContent.MetaDescription, Valid: enContent.MetaDescription != ""},
			sql.NullString{String: "deepseek", Valid: true},
		)
		if err != nil {
			log.Printf("[daily-posts] EN post insert: %v", err)
		}
		base.ENPostID = enPostID
		base.ENSlug = enSlug
	}

	// Link via translationGroupId
	groupID := generateID("grp_")
	svc.DB.Exec(ctx, `UPDATE "Post" SET "translationGroupId" = $1 WHERE id = $2`, groupID, esPostID)
	if enPostID != "" {
		svc.DB.Exec(ctx, `UPDATE "Post" SET "translationGroupId" = $1 WHERE id = $2`, groupID, enPostID)
	}

	// Agent pipeline
	if svc.Config.AgentPipelineEnabled {
		agentPipelineResult := runAgentPipeline(ctx, svc, map[string]string{"es": esPostID, "en": enPostID})
		if agentPipelineResult != nil {
			agentJSON, _ := json.Marshal(agentPipelineResult)
			svc.DB.Exec(ctx,
				`UPDATE "CronRun" SET "agentPipeline" = $1 WHERE slot = $2`,
				string(agentJSON), slot,
			)
		}
	}

	// IndexNow
	if !opts.SkipIndexNow {
		urls := []string{postURL(svc.Config, "es", esSlug)}
		if enSlug != "" {
			urls = append(urls, postURL(svc.Config, "en", enSlug))
		}
		result, err := SubmitToIndexNow(ctx, svc.Config, urls)
		if err == nil {
			base.IndexNow = result
			if result.Submitted > 0 && !result.RateLimited {
				svc.DB.Exec(ctx,
					`UPDATE "Post" SET "indexNowSubmittedAt" = NOW() WHERE id IN ($1, $2)`,
					esPostID, enPostID,
				)
			}
		}
	}

	base.Status = "SUCCESS"
	base.DurationMs = time.Since(start).Milliseconds()
	recordCronSuccess(ctx, svc.DB, slot, esPostID, enPostID, opts.DryRun)
	return base
}

func runAgentPipeline(ctx context.Context, svc *services.Services, postIDs map[string]string) map[string]interface{} {
	result := make(map[string]interface{})

	if esPostID, ok := postIDs["es"]; ok {
		var title, content string
		svc.DB.QueryRow(ctx, `SELECT title, content FROM "Post" WHERE id = $1`, esPostID).Scan(&title, &content)

		seoResult := agents.RunSEOAgent(ctx, esPostID, title, content, svc.DB)
		result["seo"] = seoResult

		result["leads"] = agents.RunLeadsAgent(ctx, title, content, svc.DB)

		distResult := agents.RunDistributionAgent(ctx, title, content, esPostID, svc.DB)
		result["distribution"] = distResult
	}

	return result
}

func postURL(cfg *config.AppConfig, locale, slug string) string {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = "https://tanok-tech.com"
	}
	return fmt.Sprintf("%s/%s/blog/%s", baseURL, locale, slug)
}

func ensureUniqueSlug(ctx context.Context, db *pgxpool.Pool, baseSlug, locale string) string {
	slug := baseSlug
	for i := 1; i < 10; i++ {
		var count int
		err := db.QueryRow(ctx,
			`SELECT COUNT(*) FROM "Post" WHERE slug = $1 AND locale = $2`,
			slug, locale,
		).Scan(&count)
		if err != nil || count == 0 {
			return slug
		}
		slug = fmt.Sprintf("%s-%d", baseSlug, i)
	}
	return slug
}

func recordCronSuccess(ctx context.Context, db *pgxpool.Pool, slot, esPostID, enPostID string, dryRun bool) {
	if dryRun {
		return
	}
	postID := esPostID
	if postID == "" {
		postID = enPostID
	}
	db.Exec(ctx,
		`INSERT INTO "CronRun" (id, slot, status, "postId", error, "createdAt")
		 VALUES ($1, $2, 'SUCCESS', $3, NULL, NOW())
		 ON CONFLICT (slot) DO UPDATE SET status = 'SUCCESS', "postId" = $3, error = NULL`,
		generateID("cr_"), slot, postID,
	)
}

func recordCronFailure(ctx context.Context, db *pgxpool.Pool, slot, errMsg string, dryRun bool) {
	if dryRun {
		return
	}
	db.Exec(ctx,
		`INSERT INTO "CronRun" (id, slot, status, "postId", error, "createdAt")
		 VALUES ($1, $2, 'FAILED', NULL, $3, NOW())
		 ON CONFLICT (slot) DO UPDATE SET status = 'FAILED', error = $3`,
		generateID("cr_"), slot, errMsg,
	)
}

func generateID(prefix string) string {
	b := make([]byte, 12)
	rand.Read(b)
	return prefix + hex.EncodeToString(b)
}
