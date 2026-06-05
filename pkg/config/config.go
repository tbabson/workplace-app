package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	Port             string
	Env              string
	DatabaseURL      string
	RedisURL         string
	JWTSecret        string
	JWTExpirationHrs int
	CompanyLat       float64
	CompanyLng       float64
	GeofenceRadius   float64
	DeviceLockEnabled bool
	SMTPHost         string
	SMTPPort         int
	SMTPUser         string
	SMTPPassword     string
	SMTPFrom         string
}

func Load() *Config {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using OS environment")
	}

	return &Config{
		Port:             env("PORT", "5002"),
		Env:              env("ENV", "development"),
		DatabaseURL:      env("DATABASE_URL", "postgres://postgres:password@localhost:5432/workplace?sslmode=disable"),
		RedisURL:         env("REDIS_URL", "redis://localhost:6379"),
		JWTSecret:        env("JWT_SECRET", "change-me"),
		JWTExpirationHrs: envInt("JWT_EXPIRATION_HOURS", 24),
		CompanyLat:       envFloat("COMPANY_LAT", 6.5244),
		CompanyLng:       envFloat("COMPANY_LNG", 3.3792),
		GeofenceRadius:   envFloat("GEOFENCE_RADIUS", 100),
		DeviceLockEnabled: env("ENV", "development") != "development",
		SMTPHost:         env("SMTP_HOST", "mailhog"),
		SMTPPort:         envInt("SMTP_PORT", 1025),
		SMTPUser:         env("SMTP_USER", ""),
		SMTPPassword:     env("SMTP_PASSWORD", ""),
		SMTPFrom:         env("SMTP_FROM", "noreply@workplace.local"),
	}
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
