package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/tanok/tanok-web-api/internal/models"
)

func (s *Services) ListPosts(ctx context.Context, locale string, page, limit int) (*models.PostListResponse, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 50 {
		limit = 20
	}
	offset := (page - 1) * limit

	cacheKey := fmt.Sprintf("posts:%s:page:%d:limit:%d", locale, page, limit)
	cached, err := s.Redis.Get(ctx, cacheKey).Result()
	if err == nil && cached != "" {
		var resp models.PostListResponse
		if json.Unmarshal([]byte(cached), &resp) == nil {
			return &resp, nil
		}
	}

	var total int64
	err = s.DB.QueryRow(ctx,
		`SELECT COUNT(*) FROM "Post" WHERE locale = $1 AND status = $2`,
		locale, models.PostStatusPublished,
	).Scan(&total)
	if err != nil {
		return nil, fmt.Errorf("count posts: %w", err)
	}

	rows, err := s.DB.Query(ctx,
		`SELECT id, title, slug, COALESCE(excerpt, ''), COALESCE("coverImage", ''),
		        COALESCE("publishedAt"::text, ''), locale,
		        COALESCE("metaTitle", ''), COALESCE("metaDescription", ''),
		        COALESCE("translationGroupId", '')
		 FROM "Post"
		 WHERE locale = $1 AND status = $2
		 ORDER BY "publishedAt" DESC
		 LIMIT $3 OFFSET $4`,
		locale, models.PostStatusPublished, limit, offset,
	)
	if err != nil {
		return nil, fmt.Errorf("query posts: %w", err)
	}
	defer rows.Close()

	var posts []models.PostSummary
	for rows.Next() {
		var p models.PostSummary
		if err := rows.Scan(&p.ID, &p.Title, &p.Slug, &p.Excerpt, &p.CoverImage,
			&p.PublishedAt, &p.Locale, &p.MetaTitle, &p.MetaDescription,
			&p.TranslationGroupID); err != nil {
			return nil, fmt.Errorf("scan post: %w", err)
		}
		posts = append(posts, p)
	}

	resp := &models.PostListResponse{
		Posts:   posts,
		Page:    page,
		Limit:   limit,
		Total:   total,
		HasMore: int64(offset+limit) < total,
	}

	data, _ := json.Marshal(resp)
	s.Redis.Set(ctx, cacheKey, data, 1*time.Hour)

	return resp, nil
}

func (s *Services) GetPostBySlug(ctx context.Context, locale, slug string) (*models.PostDetail, error) {
	cacheKey := fmt.Sprintf("post:%s:%s", locale, slug)
	cached, err := s.Redis.Get(ctx, cacheKey).Result()
	if err == nil && cached != "" {
		var post models.PostDetail
		if json.Unmarshal([]byte(cached), &post) == nil {
			return &post, nil
		}
	}

	var post models.PostDetail
	var publishedAt, createdAt, updatedAt sql.NullTime
	var ogImage, generatedBy, sourceURL, sourceName, authorID sql.NullString

	err = s.DB.QueryRow(ctx,
		`SELECT id, title, slug, COALESCE(excerpt, ''), content,
		        COALESCE("coverImage", ''), status,
		        "publishedAt", "createdAt", "updatedAt",
		        "authorId", locale,
		        COALESCE("translationGroupId", ''),
		        COALESCE("metaTitle", ''), COALESCE("metaDescription", ''),
		        COALESCE("ogImage", ''), COALESCE("generatedBy", ''),
		        COALESCE("sourceUrl", ''), COALESCE("sourceName", '')
		 FROM "Post"
		 WHERE slug = $1 AND locale = $2 AND status = $3`,
		slug, locale, models.PostStatusPublished,
	).Scan(&post.ID, &post.Title, &post.Slug, &post.Excerpt, &post.Content,
		&post.CoverImage, &post.Status,
		&publishedAt, &createdAt, &updatedAt,
		&authorID, &post.Locale,
		&post.TranslationGroupID,
		&post.MetaTitle, &post.MetaDescription,
		&ogImage, &generatedBy, &sourceURL, &sourceName,
	)
	if err != nil {
		return nil, fmt.Errorf("post not found: %w", err)
	}

	post.OgImage = ogImage.String
	post.GeneratedBy = generatedBy.String
	post.SourceURL = sourceURL.String
	post.SourceName = sourceName.String
	post.AuthorID = authorID.String

	if publishedAt.Valid {
		post.PublishedAt = publishedAt.Time.Format(time.RFC3339)
	}
	if createdAt.Valid {
		post.CreatedAt = createdAt.Time.Format(time.RFC3339)
	}
	if updatedAt.Valid {
		post.UpdatedAt = updatedAt.Time.Format(time.RFC3339)
	}

	data, _ := json.Marshal(post)
	s.Redis.Set(ctx, cacheKey, data, 1*time.Hour)

	return &post, nil
}

func (s *Services) GetSiblingSlug(ctx context.Context, translationGroupID, locale string) (*models.SiblingResponse, error) {
	var slug, siblingLocale string
	err := s.DB.QueryRow(ctx,
		`SELECT slug, locale FROM "Post"
		 WHERE "translationGroupId" = $1 AND locale != $2 AND status = $3
		 LIMIT 1`,
		translationGroupID, locale, models.PostStatusPublished,
	).Scan(&slug, &siblingLocale)
	if err != nil {
		return nil, err
	}
	return &models.SiblingResponse{Slug: slug, Locale: siblingLocale}, nil
}
