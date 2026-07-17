package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/tanok/tanok-web-api/internal/config"
	"github.com/tanok/tanok-web-api/internal/db"
	"github.com/tanok/tanok-web-api/internal/handlers"
	"github.com/tanok/tanok-web-api/internal/middleware"
	redisclient "github.com/tanok/tanok-web-api/internal/redis"
	"github.com/tanok/tanok-web-api/internal/services"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stderr, nil))

	cfg := config.Load()
	if cfg.DatabaseURL == "" {
		logger.Error("DATABASE_URL is required")
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := db.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()
	logger.Info("connected to database")

	rdb, err := redisclient.NewClient(ctx, cfg.RedisURL)
	if err != nil {
		logger.Error("failed to connect to redis", "error", err)
		os.Exit(1)
	}
	defer rdb.Close()
	logger.Info("connected to redis")

	svc := services.New(pool, rdb, cfg)

	if cfg.GinMode != "" {
		gin.SetMode(cfg.GinMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(gin.Recovery())
	if gin.Mode() != gin.ReleaseMode {
		r.Use(gin.Logger())
	}

	r.Use(middleware.CORS(cfg.CORSAllowedOrigins))

	h := handlers.NewHandler(svc)

	auth := r.Group("/api/auth")
	{
		auth.POST("/login", h.Login)
		auth.POST("/logout", h.Logout)
		auth.GET("/me", middleware.JWTAuth(svc), h.Me)
	}

	posts := r.Group("/api/posts")
	{
		posts.GET("", h.ListPosts)
		posts.GET("/:slug", h.GetPost)
		posts.GET("/:slug/sibling", h.GetPostSibling)
	}

	leads := r.Group("/api/leads")
	{
		leads.POST("/capture",
			middleware.RateLimit(rdb, middleware.RateLimitConfig{
				WindowSeconds: 60,
				MaxRequests:   30,
				KeyPrefix:     "ratelimit:leads:capture:",
			}),
			h.CaptureLead,
		)
		leads.GET("", middleware.JWTAuth(svc), h.ListLeads)
		leads.GET("/:id", middleware.JWTAuth(svc), h.GetLead)
	}

	insights := r.Group("/api/insights")
	{
		insights.POST("/track",
			middleware.RateLimit(rdb, middleware.RateLimitConfig{
				WindowSeconds: 60,
				MaxRequests:   100,
				KeyPrefix:     "ratelimit:insights:track:",
			}),
			h.TrackEvent,
		)
		insights.GET("/stats", middleware.JWTAuth(svc), h.GetStats)
		insights.GET("/events", middleware.JWTAuth(svc), h.GetEvents)
		insights.GET("/top-elements", middleware.JWTAuth(svc), h.GetTopElements)
		insights.GET("/engagement", middleware.JWTAuth(svc), h.GetEngagement)
		insights.GET("/heatmap", middleware.JWTAuth(svc), h.GetHeatmap)
		insights.GET("/traffic-sources", middleware.JWTAuth(svc), h.GetTrafficSources)
		insights.POST("/analyze", middleware.JWTAuth(svc), h.Analyze)
	}

	questionnaire := r.Group("/api")
	{
		questionnaire.POST("/send-web-questionnaire",
			middleware.RateLimit(rdb, middleware.RateLimitConfig{
				WindowSeconds: 60,
				MaxRequests:   5,
				KeyPrefix:     "ratelimit:questionnaire:",
			}),
			h.SendWebQuestionnaire,
		)
	}

	cron := r.Group("/api/cron")
	{
		protect := middleware.CronSecret(cfg.CronSecret)
		cron.POST("/daily-posts", protect, h.RunDailyGeneration)
	}

	images := r.Group("/api/images")
	{
		images.POST("/generate-cover", middleware.JWTAuth(svc), h.GenerateCoverImage)
	}

	r.GET("/healthz", h.Healthz)

	addr := fmt.Sprintf(":%s", cfg.Port)
	srv := &http.Server{
		Addr:    addr,
		Handler: r,
	}

	go func() {
		logger.Info("server starting", "addr", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info("shutting down server")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("server forced to shutdown", "error", err)
	}
	logger.Info("server stopped")
}
