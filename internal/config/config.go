package config

import (
	"log"
	"os"
	"strconv"
)

type Config struct {
	Port             string
	DatabaseURL      string
	JWTPrivateKeyB64 string
	JWTPublicKeyB64  string
	GoogleClientID   string
	GoogleClientSecret string
	GoogleRedirectURL  string
	AccessTokenMaxAge  int // minutes
	RefreshTokenMaxAge int // days
	CookieSecure       bool
	CookieDomain       string
}

func Load() *Config {
	cfg := &Config{
		Port:               getEnv("PORT", "8000"),
		DatabaseURL:        mustEnv("DATABASE_URL"),
		JWTPrivateKeyB64:   mustEnv("JWT_PRIVATE_KEY_B64"),
		JWTPublicKeyB64:    mustEnv("JWT_PUBLIC_KEY_B64"),
		GoogleClientID:     mustEnv("GOOGLE_CLIENT_ID"),
		GoogleClientSecret: mustEnv("GOOGLE_CLIENT_SECRET"),
		GoogleRedirectURL:  mustEnv("GOOGLE_REDIRECT_URL"),
		AccessTokenMaxAge:  getEnvInt("ACCESS_TOKEN_MAX_AGE", 5), // 5 minutes
		RefreshTokenMaxAge: getEnvInt("REFRESH_TOKEN_MAX_AGE", 1), // 1 day
		CookieSecure:       getEnv("COOKIE_SECURE", "true") == "true",
		CookieDomain:       getEnv("COOKIE_DOMAIN", ""),
	}
	return cfg
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("Required environment variable %s is not set", key)
	}
	return v
}

func getEnvInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	i, err := strconv.Atoi(v)
	if err != nil {
		log.Printf("Warning: Environment variable %s is not a valid integer, using fallback %d", key, fallback)
		return fallback
	}
	return i
}
