package config

import (
	"os"
	"time"
)

type Config struct {
	HTTPAddress       string
	ReadHeaderTimeout time.Duration
	SharedConfigDir   string
}

func Load() Config {
	return Config{
		HTTPAddress:       envString("HTTP_ADDRESS", ":8080"),
		ReadHeaderTimeout: 5 * time.Second,
		SharedConfigDir:   envString("SHARED_CONFIG_DIR", "../shared/config"),
	}
}

func envString(key string, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	return value
}
