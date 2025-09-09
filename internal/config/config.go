package config

import (
	"os"
)

type Config struct {
	DBUrl     string
	JWTSecret string
}

func LoadConfig() *Config {
	cfg := &Config{
		DBUrl:     getEnv("DATABASE_URL", "postgres://collab:collab123@localhost:5432/collabdb?sslmode=disable"),
		JWTSecret: getEnv("JWT_SECRET", "vvvsupersecret"),
	}

	return cfg
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
