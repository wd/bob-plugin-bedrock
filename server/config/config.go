package config

import (
	"os"
)

// Config holds server configuration
type Config struct {
	AWSRegion  string
	AWSProfile string
	ServerPort string
}

// Load reads configuration from environment variables with defaults
func Load() *Config {
	return &Config{
		AWSRegion:  getEnv("AWS_REGION", "us-east-2"),
		AWSProfile: getEnv("AWS_PROFILE", ""),
		ServerPort: getEnv("SERVER_PORT", "18081"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
