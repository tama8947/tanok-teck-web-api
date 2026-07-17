package cron

import (
	"context"
	"database/sql"
	"fmt"
	"math/rand"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/tanok/tanok-web-api/internal/config"
)

const discoveryThreshold = 5

type TopicPair struct {
	ID          string
	ES          string
	EN          string
	SearchQuery string
}

type PoolStatus struct {
	Available         int
	Used              int
	Total             int
	DiscoveryTriggered bool
}

func GetPoolStatus(ctx context.Context, db *pgxpool.Pool) (*PoolStatus, error) {
	var available, used, total int

	db.QueryRow(ctx,
		`SELECT COUNT(*) FROM "CronTopic" WHERE "isActive" = true AND "usedAt" IS NULL`,
	).Scan(&available)

	db.QueryRow(ctx,
		`SELECT COUNT(*) FROM "CronTopic" WHERE "isActive" = true AND "usedAt" IS NOT NULL`,
	).Scan(&used)

	total = available + used

	return &PoolStatus{
		Available:         available,
		Used:              used,
		Total:             total,
		DiscoveryTriggered: false,
	}, nil
}

func PickNextTopic(ctx context.Context, db *pgxpool.Pool, cfg *config.AppConfig, rdb *redis.Client) (*TopicPair, error) {
	lockKey := "cron:topic_select:lock"
	locked, err := rdb.SetNX(ctx, lockKey, "1", 30*time.Second).Result()
	if err != nil || !locked {
		if err == nil {
			time.Sleep(500 * time.Millisecond)
			retryLocked, retryErr := rdb.SetNX(ctx, lockKey, "1", 30*time.Second).Result()
			if retryErr != nil || !retryLocked {
				return nil, fmt.Errorf("failed to acquire topic selection lock")
			}
		} else {
			time.Sleep(500 * time.Millisecond)
		}
	}
	defer rdb.Del(ctx, lockKey)

	var available int
	db.QueryRow(ctx,
		`SELECT COUNT(*) FROM "CronTopic" WHERE "isActive" = true AND "usedAt" IS NULL`,
	).Scan(&available)

	discoveryTriggered := false
	if available < discoveryThreshold {
		discoveryTriggered = true
		fmt.Printf("[topic-selector] Pool low (%d available, threshold=%d) — triggering discovery\n", available, discoveryThreshold)

		topics, err := DiscoverTopics(ctx, cfg)
		if err != nil {
			fmt.Printf("[topic-selector] Discovery failed: %v\n", err)
		} else if len(topics) > 0 {
			if err := PersistDiscoveredTopics(ctx, db, topics); err != nil {
				fmt.Printf("[topic-selector] Persist failed: %v\n", err)
			}
		}

		db.QueryRow(ctx,
			`SELECT COUNT(*) FROM "CronTopic" WHERE "isActive" = true AND "usedAt" IS NULL`,
		).Scan(&available)
	}

	if available == 0 {
		fmt.Println("[topic-selector] Pool exhausted — resetting all topics (rotation)")
		db.Exec(ctx,
			`UPDATE "CronTopic" SET "usedAt" = NULL, "useCount" = "useCount" + 1 WHERE "isActive" = true`,
		)

		db.QueryRow(ctx,
			`SELECT COUNT(*) FROM "CronTopic" WHERE "isActive" = true AND "usedAt" IS NULL`,
		).Scan(&available)
	}

	if available == 0 {
		return nil, fmt.Errorf("no topics available even after reset. Seed the CronTopic table first")
	}

	skip := rand.Intn(available)
	row := db.QueryRow(ctx,
		`SELECT id, "esText", "enText", COALESCE("searchQuery", '')
		 FROM "CronTopic"
		 WHERE "isActive" = true AND "usedAt" IS NULL
		 ORDER BY "useCount" ASC
		 LIMIT 1 OFFSET $1`,
		skip,
	)

	var topic TopicPair
	err = row.Scan(&topic.ID, &topic.ES, &topic.EN, &topic.SearchQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to select topic: %w", err)
	}

	db.Exec(ctx,
		`UPDATE "CronTopic" SET "usedAt" = NOW() WHERE id = $1`,
		topic.ID,
	)

	fmt.Printf("[topic-selector] Picked topic: %q\n", topic.ES)
	_ = discoveryTriggered

	return &topic, nil
}

func resetAndRotate(ctx context.Context, db *pgxpool.Pool) error {
	fmt.Println("[topic-selector] Pool exhausted — resetting all topics")
	_, err := db.Exec(ctx,
		`UPDATE "CronTopic" SET "usedAt" = NULL, "useCount" = "useCount" + 1 WHERE "isActive" = true`,
	)
	return err
}

var _ = sql.NullString{}
