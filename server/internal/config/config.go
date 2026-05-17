package config

import (
	"os"
	"time"
)

type Config struct {
	HTTPAddress       string
	ReadHeaderTimeout time.Duration
}

func Load() Config {
	return Config{
		HTTPAddress:       envString("HTTP_ADDRESS", ":8080"),
		ReadHeaderTimeout: 5 * time.Second,
	}
}

func envString(key string, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	return value
}
