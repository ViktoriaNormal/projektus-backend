package config

import (
	"log"
	"os"
	"strconv"
	"time"
)

type Config struct {
	ServerPort string

	DBURL string

	JWTAccessSecret  string
	JWTRefreshSecret string

	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration

	RateLimitEmailWindowMinutes int
	RateLimitEmailMaxFailures   int

	RateLimitEmailBlockMinutes int

	RateLimitIPWindowMinutes int
	RateLimitIPMaxFailures   int
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func mustInt(key string, def int) int {
	vStr := getenv(key, "")
	if vStr == "" {
		return def
	}
	v, err := strconv.Atoi(vStr)
	if err != nil {
		log.Fatalf("invalid int value for %s: %v", key, err)
	}
	return v
}

func Load() *Config {
	cfg := &Config{
		ServerPort: getenv("SERVER_PORT", "8080"),
		DBURL:      getenv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/projektus?sslmode=disable"),

		JWTAccessSecret:  getenv("JWT_ACCESS_SECRET", "dev-access-secret"),
		JWTRefreshSecret: getenv("JWT_REFRESH_SECRET", "dev-refresh-secret"),

		AccessTokenTTL:  30 * time.Minute,
		RefreshTokenTTL: 24 * time.Hour,

		RateLimitEmailWindowMinutes: mustInt("RATE_LIMIT_EMAIL_WINDOW_MINUTES", 15),
		RateLimitEmailMaxFailures:   mustInt("RATE_LIMIT_EMAIL_MAX_FAILURES", 5),

		RateLimitEmailBlockMinutes: mustInt("RATE_LIMIT_EMAIL_BLOCK_MINUTES", 15),

		RateLimitIPWindowMinutes: mustInt("RATE_LIMIT_IP_WINDOW_MINUTES", 60),
		RateLimitIPMaxFailures:   mustInt("RATE_LIMIT_IP_MAX_FAILURES", 20),
	}

	return cfg
}

