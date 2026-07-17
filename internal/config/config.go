package config

import "os"

type AppConfig struct {
	Port          string
	DatabaseURL   string
	RedisURL      string
	JWTSecret     string
	MiniMaxAPIKey string
	DeepSeekAPIKey  string
	DeepSeekAPIURL  string
	ResendAPIKey    string
	BraveAPIKey     string
	R2AccessKeyID     string
	R2SecretAccessKey string
	R2BucketName      string
	R2PublicURL       string
	R2Endpoint        string
	R2Region          string
	BaseURL              string
	DailyPostsAuthorID   string
	AgentPipelineEnabled bool
	CookieDomain         string
	CronSecret           string
	GinMode              string
	CORSAllowedOrigins   string
}

func Load() *AppConfig {
	cfg := &AppConfig{
		Port:                envOrDefault("PORT", "4733"),
		DatabaseURL:         os.Getenv("DATABASE_URL"),
		RedisURL:            envOrDefault("REDIS_URL", "redis://localhost:6379"),
		JWTSecret:           envOrDefault("JWT_SECRET", "fallback-secret-for-development"),
		MiniMaxAPIKey:       os.Getenv("MINIMAX_API_KEY"),
		DeepSeekAPIKey:      os.Getenv("DEEPSEEK_API_KEY"),
		DeepSeekAPIURL:      envOrDefault("DEEPSEEK_API_URL", "https://api.deepseek.com/v1/chat/completions"),
		ResendAPIKey:        os.Getenv("RESEND_API_KEY"),
		BraveAPIKey:         os.Getenv("BRAVE_API_KEY"),
		R2AccessKeyID:       os.Getenv("R2_ACCESS_KEY_ID"),
		R2SecretAccessKey:   os.Getenv("R2_SECRET_ACCESS_KEY"),
		R2BucketName:        os.Getenv("R2_BUCKET_NAME"),
		R2PublicURL:         os.Getenv("R2_PUBLIC_URL"),
		R2Endpoint:          os.Getenv("R2_ENDPOINT"),
		R2Region:            envOrDefault("R2_REGION", "auto"),
		BaseURL:              os.Getenv("BASE_URL"),
		DailyPostsAuthorID:   os.Getenv("DAILY_POSTS_AUTHOR_ID"),
		AgentPipelineEnabled: parseBoolEnv("AGENT_PIPELINE_ENABLED", true),
		CookieDomain:         envOrDefault("COOKIE_DOMAIN", ""),
		CronSecret:           os.Getenv("CRON_SECRET"),
		GinMode:              os.Getenv("GIN_MODE"),
		CORSAllowedOrigins:   envOrDefault("CORS_ALLOWED_ORIGINS", ""),
	}
	return cfg
}

func envOrDefault(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

func parseBoolEnv(key string, defaultVal bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return defaultVal
	}
	switch v {
	case "true", "1", "yes", "on":
		return true
	default:
		return false
	}
}
