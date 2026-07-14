package config

import (
	"context"
	"fmt"
	"os"
)

type Config struct {
	App    APP
	DB     DBConfig
	Server ServerConfig
	SQS    SQS
	AI     AI
}

type APP struct {
	ServiceName string
	Env         string
	AWSRegion   string
}

type DBConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
	SSLMode  string
}

type ServerConfig struct {
	MetricsServerPort string
	OTelCollectorAddr string
}

type SQS struct {
	CategorizationSQS string
	NotificationSQS   string
}

type AI struct {
	API_KEY             string
	API_ENDPOINT        string
	Model               string
	HuggingFaceAIModel  string
	PromptFile          string
	LocalOllamaEndpoint string
}

func Load(ctx context.Context) (*Config, error) {
	if getEnv("APP_ENV", "development") == "production" {
		return loadAWS(ctx)
	}

	return loadEnv()
}

func required(key string) string {
	value := os.Getenv(key)

	if value == "" {
		panic(fmt.Sprintf("missing required environment variable: %s", key))
	}

	return value
}

func getEnv(key string, defaultValue string) string {
	value := os.Getenv(key)

	if value == "" {
		return defaultValue
	}

	return value
}

func (c *DBConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.Name, c.SSLMode,
	)
}
