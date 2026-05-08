package config

import (
	"os"
)

type Config struct {
	TemporalHost string
	DBPath       string
	ListenAddr   string
	IAMPath      string
}

func Load() *Config {
	return &Config{
		TemporalHost: getEnv("TEMPORAL_HOST", "localhost:7233"),
		DBPath:       getEnv("DB_PATH", "./workflow_engine.db"),
		ListenAddr:   getEnv("LISTEN_ADDR", ":8080"),
		IAMPath:      getEnv("IAM_PATH", "config/iam.yaml"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
