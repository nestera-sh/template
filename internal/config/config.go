package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	HTTPAddr                string
	ShutdownTimeout         time.Duration
	LogLevel                string
	DatabaseURL             string
	DatabaseMaxConns        int
	DatabaseMinConns        int
	DatabaseConnectTimeout  time.Duration
	DatabaseQueryTimeout    time.Duration
	DatabaseMaxConnLifetime time.Duration
}

func Load() (*Config, error) {
	cfg := &Config{
		HTTPAddr:        getEnv("HTTP_ADDR", ":8080"),
		ShutdownTimeout: getEnvDuration("SHUTDOWN_TIMEOUT", 10*time.Second),
		LogLevel:        getEnv("LOG_LEVEL", "info"),
		DatabaseURL:     os.Getenv("DATABASE_URL"),
	}

	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}

	maxConns, err := getEnvInt("DATABASE_MAX_CONNS", 10)
	if err != nil {
		return nil, err
	}
	cfg.DatabaseMaxConns = maxConns

	minConns, err := getEnvInt("DATABASE_MIN_CONNS", 1)
	if err != nil {
		return nil, err
	}
	cfg.DatabaseMinConns = minConns

	connectTimeout, err := getEnvDuration("DATABASE_CONNECT_TIMEOUT", 5*time.Second)
	if err != nil {
		return nil, err
	}
	cfg.DatabaseConnectTimeout = connectTimeout

	queryTimeout, err := getEnvDuration("DATABASE_QUERY_TIMEOUT", 5*time.Second)
	if err != nil {
		return nil, err
	}
	cfg.DatabaseQueryTimeout = queryTimeout

	maxConnLifetime, err := getEnvDuration("DATABASE_MAX_CONN_LIFETIME", time.Hour)
	if err != nil {
		return nil, err
	}
	cfg.DatabaseMaxConnLifetime = maxConnLifetime

	return cfg, nil
}



func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) (int, error) {
	raw := os.Getenv(key)
	if raw == "" {
		return defaultValue, nil
	}

	parsed, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("parse %s: %w", key, err)
	}
	return parsed, nil
}

func getEnvDuration(key string, defaultValue time.Duration) (time.Duration, error) {
	raw := os.Getenv(key)
	if raw == "" {
		return defaultValue, nil
	}

	parsed, err := time.ParseDuration(raw)
	if err != nil {
		return 0, fmt.Errorf("parse %s: %w", key, err)
	}
	return parsed, nil
}