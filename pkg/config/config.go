package config

import (
	"errors"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

const defaultJWTSecret = "change-me"

type Config struct {
	Port              string
	Env               string
	DatabaseURL       string
	RedisURL          string
	JWTSecret         string
	JWTExpirationHrs  int
	AllowedOrigins    []string
	Location          *time.Location
	CompanyLat        float64
	CompanyLng        float64
	GeofenceRadius    float64
	DeviceLockEnabled bool
	SMTPHost          string
	SMTPPort          int
	SMTPUser          string
	SMTPPassword      string
	SMTPFrom          string
	// Optional: creates the first super_admin on startup if no super_admin exists.
	BootstrapAdminEmail    string
	BootstrapAdminPassword string
}

func (c *Config) IsDevelopment() bool { return c.Env == "development" }

func Load() *Config {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using OS environment")
	}

	tzName := env("APP_TIMEZONE", "Africa/Lagos")
	loc, err := time.LoadLocation(tzName)
	if err != nil {
		log.Fatalf("config: invalid APP_TIMEZONE %q: %v", tzName, err)
	}

	appEnv := env("ENV", "development")
	cfg := &Config{
		Port:                   env("PORT", "5002"),
		Env:                    appEnv,
		DatabaseURL:            env("DATABASE_URL", "postgres://postgres:password@localhost:5432/workplace?sslmode=disable"),
		RedisURL:               env("REDIS_URL", "redis://localhost:6379"),
		JWTSecret:              env("JWT_SECRET", defaultJWTSecret),
		JWTExpirationHrs:       envInt("JWT_EXPIRATION_HOURS", 24),
		AllowedOrigins:         strings.Split(env("ALLOWED_ORIGINS", "http://localhost:5173"), ","),
		Location:               loc,
		CompanyLat:             envFloat("COMPANY_LAT", 6.5244),
		CompanyLng:             envFloat("COMPANY_LNG", 3.3792),
		GeofenceRadius:         envFloat("GEOFENCE_RADIUS", 100),
		DeviceLockEnabled:      appEnv != "development",
		SMTPHost:               env("SMTP_HOST", "mailhog"),
		SMTPPort:               envInt("SMTP_PORT", 1025),
		SMTPUser:               env("SMTP_USER", ""),
		SMTPPassword:           env("SMTP_PASSWORD", ""),
		SMTPFrom:               env("SMTP_FROM", "noreply@workplace.local"),
		BootstrapAdminEmail:    env("BOOTSTRAP_ADMIN_EMAIL", ""),
		BootstrapAdminPassword: env("BOOTSTRAP_ADMIN_PASSWORD", ""),
	}

	if err := cfg.validate(); err != nil {
		log.Fatalf("config: %v", err)
	}
	return cfg
}

func (c *Config) validate() error {
	if c.JWTExpirationHrs <= 0 {
		return errors.New("JWT_EXPIRATION_HOURS must be positive")
	}
	if c.IsDevelopment() {
		if c.JWTSecret == defaultJWTSecret {
			log.Println("WARNING: using the default JWT_SECRET; set a real one before deploying")
		}
		return nil
	}
	if c.JWTSecret == defaultJWTSecret || len(c.JWTSecret) < 32 {
		return errors.New("JWT_SECRET must be set to a random value of at least 32 characters outside development")
	}
	return nil
}

func env(key, def string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return def
}

func envInt(key string, def int) int {
	if v, ok := os.LookupEnv(key); ok {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return def
}

func envFloat(key string, def float64) float64 {
	if v, ok := os.LookupEnv(key); ok {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return def
}
