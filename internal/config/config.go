package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

// Config holds all configuration for the application
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	JWT      JWTConfig
	Log      LogConfig
	Redis    RedisConfig
	Jobs     JobsConfig
	OTel     OTelConfig
}

// ServerConfig holds server-related configuration
type ServerConfig struct {
	Port         string
	Host         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
}

// DatabaseConfig holds database-related configuration
type DatabaseConfig struct {
	URL      string
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode  string
}

// JWTConfig holds JWT-related configuration
type JWTConfig struct {
	Secret     string
	Expiration time.Duration
}

// LogConfig holds logging-related configuration
type LogConfig struct {
	Level string
	Env   string
}

// RedisConfig holds Redis-related configuration
type RedisConfig struct {
	URL      string
	Host     string
	Port     string
	Password string
	DB       int
}

// JobsConfig holds background job configuration
type JobsConfig struct {
	Concurrency int
	Queues      []string
}

// OTelConfig holds OpenTelemetry configuration
type OTelConfig struct {
	Enabled     bool
	ServiceName string
	JaegerURL   string
	Environment string
}

// Load loads configuration from environment variables and .env file
func Load() (*Config, error) {
	// Load .env file if it exists
	if err := godotenv.Load(); err != nil {
		// It's okay if .env doesn't exist in production
		// fmt.Printf("Warning: .env file not found: %v\n", err)
	}

	config := &Config{
		Server: ServerConfig{
			Port:         getEnv("PORT", "8080"),
			Host:         getEnv("HOST", "0.0.0.0"),
			ReadTimeout:  getDurationEnv("READ_TIMEOUT", 30*time.Second),
			WriteTimeout: getDurationEnv("WRITE_TIMEOUT", 30*time.Second),
			IdleTimeout:  getDurationEnv("IDLE_TIMEOUT", 60*time.Second),
		},
		Database: DatabaseConfig{
			URL:      getEnv("DATABASE_URL", ""),
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnv("DB_PORT", "5432"),
			User:     getEnv("DB_USER", "postgres"),
			Password: getEnv("DB_PASSWORD", ""),
			DBName:   getEnv("DB_NAME", "connex"),
			SSLMode:  getEnv("DB_SSLMODE", "disable"),
		},
		JWT: JWTConfig{
			Secret:     getEnvStrict("JWT_SECRET"),
			Expiration: getDurationEnv("JWT_EXPIRATION", 24*time.Hour),
		},
		Log: LogConfig{
			Level: getEnv("LOG_LEVEL", "info"),
			Env:   getEnv("ENV", "development"),
		},
		Redis: RedisConfig{
			URL:      getEnv("REDIS_URL", ""),
			Host:     getEnv("REDIS_HOST", "localhost"),
			Port:     getEnv("REDIS_PORT", "6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getIntEnv("REDIS_DB", 0),
		},
		Jobs: JobsConfig{
			Concurrency: getIntEnv("JOBS_CONCURRENCY", 10),
			Queues:      []string{"default", "critical", "low"},
		},
		OTel: OTelConfig{
			Enabled:     getEnv("OTEL_ENABLED", "false") == "true",
			ServiceName: getEnv("OTEL_SERVICE_NAME", "connex"),
			JaegerURL:   getEnv("OTEL_JAEGER_URL", "http://localhost:14268/api/traces"),
			Environment: getEnv("OTEL_ENVIRONMENT", "development"),
		},
	}

	// Validate required configuration
	if err := config.validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return config, nil
}

// validate checks if all required configuration values are set
func (c *Config) validate() error {
	if c.Server.Port == "" {
		return fmt.Errorf("PORT is required")
	}

	if c.JWT.Secret == "" {
		return fmt.Errorf("JWT_SECRET must be set")
	}

	return nil
}

// getEnv gets an environment variable with a fallback default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getDurationEnv gets a duration environment variable with a fallback default value
func getDurationEnv(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

// getIntEnv gets an integer environment variable with a fallback default value
func getIntEnv(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// getEnvStrict gets an environment variable or panics if not set
func getEnvStrict(key string) string {
	value := os.Getenv(key)
	if value == "" {
		panic("Required environment variable missing: " + key)
	}
	return value
}
