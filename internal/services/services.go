package services

import (
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/tanok/tanok-web-api/internal/config"
)

type Services struct {
	DB     *pgxpool.Pool
	Redis  *redis.Client
	Config *config.AppConfig
}

func New(db *pgxpool.Pool, rdb *redis.Client, cfg *config.AppConfig) *Services {
	return &Services{DB: db, Redis: rdb, Config: cfg}
}
