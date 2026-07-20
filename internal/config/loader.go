package config

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

func LoadConfig() *Config {
	cfg, err := loadConfig()
	if err != nil {
		log.Fatalf("Invalid configuration: %v", err)
	}
	return cfg
}

func loadConfig() (*Config, error) {
	configPath := envOrDefault("CONFIG_FILE", filepath.Join("configs", "config.yaml"))
	if err := loadLocalDotEnv(configPath); err != nil {
		return nil, err
	}

	rawBytes, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("read config file %q: %w", configPath, err)
	}

	expanded := os.ExpandEnv(string(rawBytes))

	v := viper.New()
	v.SetConfigType("yaml")
	if err := v.ReadConfig(bytes.NewBufferString(expanded)); err != nil {
		return nil, fmt.Errorf("read config file %q: %w", configPath, err)
	}

	var cfg Config
	if err := v.UnmarshalExact(&cfg); err != nil {
		return nil, fmt.Errorf("decode config file %q: %w", configPath, err)
	}
	if err := validateConfig(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func loadLocalDotEnv(configPath string) error {
	paths := []string{".env", filepath.Join(filepath.Dir(configPath), ".env"), "../.env"}
	for _, path := range paths {
		err := godotenv.Load(path)
		if err == nil {
			return nil
		}
		if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("load environment file %q: %w", path, err)
		}
	}
	return nil
}
