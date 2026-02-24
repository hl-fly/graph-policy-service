package config

import (
	"log/slog"
	"os"

	"github.com/joho/godotenv"
)

func LoadConfigs(logger *slog.Logger) *Config {
	err := godotenv.Load(".env")
	if err != nil {
		logger.Warn("Error loading .env file, using environment variables", "error", err)
	}

	var config Config

	config.Server.Address = getEnv(logger, "SERVER_ADDRESS")

	return &config
}

func getEnv(logger *slog.Logger, key string) string {
	value := os.Getenv(key)
	if value == "" {
		logger.Warn("Environment variable not set", "key", key)
	}

	return value
}
