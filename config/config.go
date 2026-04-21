package config

import (
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	ServerPort string

	DBURL string

	CORSAllowedOrigin string

	JWTAccessSecret  string
	JWTRefreshSecret string

	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration

	RateLimitEmailWindowMinutes int
	RateLimitEmailMaxFailures   int

	RateLimitEmailBlockMinutes int

	RateLimitIPWindowMinutes int
	RateLimitIPMaxFailures   int

	AllowPublicRegistration bool

	InitialAdminUsername string
	InitialAdminEmail    string
	InitialAdminPassword string
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

func getbool(key string, def bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		log.Fatalf("invalid bool value for %s: %q", key, v)
		return def
	}
}

func Load() *Config {
	cfg := &Config{
		ServerPort: getenv("SERVER_PORT", "8080"),
		DBURL:      getenv("DATABASE_URL", "postgres://projektus:projektus@localhost:5432/projektus?sslmode=disable"),

		CORSAllowedOrigin: getenv("CORS_ALLOWED_ORIGIN", "http://localhost:5173"),

		JWTAccessSecret:  getenv("JWT_ACCESS_SECRET", "dev-access-secret"),
		JWTRefreshSecret: getenv("JWT_REFRESH_SECRET", "dev-refresh-secret"),

		AccessTokenTTL:  30 * time.Minute,
		RefreshTokenTTL: 24 * time.Hour,

		RateLimitEmailWindowMinutes: mustInt("RATE_LIMIT_EMAIL_WINDOW_MINUTES", 15),
		RateLimitEmailMaxFailures:   mustInt("RATE_LIMIT_EMAIL_MAX_FAILURES", 5),

		RateLimitEmailBlockMinutes: mustInt("RATE_LIMIT_EMAIL_BLOCK_MINUTES", 15),

		RateLimitIPWindowMinutes: mustInt("RATE_LIMIT_IP_WINDOW_MINUTES", 60),
		RateLimitIPMaxFailures:   mustInt("RATE_LIMIT_IP_MAX_FAILURES", 20),

		AllowPublicRegistration: getbool("ALLOW_PUBLIC_REGISTRATION", false),

		InitialAdminUsername: getenv("INITIAL_ADMIN_USERNAME", "admin"),
		InitialAdminEmail:    getenv("INITIAL_ADMIN_EMAIL", "admin@projektus.local"),
		InitialAdminPassword: os.Getenv("INITIAL_ADMIN_PASSWORD"),
	}

	return cfg
}

