package config

import (
	"os"
	"strconv"
)

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if value, exists := os.LookupEnv(key); exists {
		if val, err := strconv.Atoi(value); err == nil {
			return val
		}
	}
	return fallback
}

func getEnvBool(key string, fallback bool) bool {
	if value, exists := os.LookupEnv(key); exists {
		if val, err := strconv.ParseBool(value); err == nil {
			return val
		}
	}
	return fallback
}
