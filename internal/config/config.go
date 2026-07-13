package config

import (
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
	MODEL               string
	HuggingFaceAIModel  string
	PromptFile          string
	LocalOllamaEndpoint string
}

func Load() (*Config, error) {
	cfg := &Config{
		App: APP{
			ServiceName: getEnv("SERVICE_NAME", "categorization-worker"),
			Env:         getEnv("ENV", "production"),
			AWSRegion:   getEnv("AWS_REGION", "eu-central-1"),
		},
		DB: DBConfig{
			Host:     required("DB_HOST"),
			Port:     required("DB_PORT"),
			User:     required("DB_USER"),
			Password: required("DB_PASSWORD"),
			Name:     required("DB_NAME"),
			SSLMode:  getEnv("DB_SSLMODE", "require"),
		},
		Server: ServerConfig{
			MetricsServerPort: getEnv("METRICS_SERVER_PORT", ":9100"),
			OTelCollectorAddr: getEnv("OTEL_COLLECTOR_ADDR", "localhost:4317"),
		},
		AI: AI{
			API_KEY:             required("AI_API_KEY"),
			API_ENDPOINT:        required("AI_API_ENDPOINT"),
			MODEL:               required("AI_MODEL"),
			HuggingFaceAIModel:  getEnv("HUGGINGFACE_AI_MODEL", "all-minilm:l12-v2"),
			LocalOllamaEndpoint: getEnv("LOCAL_OLLAMA_ENDPOINT", "localhost:11434"),
			PromptFile:          getEnv("PROMPT_FILE", ""),
		},
		SQS: SQS{
			CategorizationSQS: required("CATEGORIZATION_SQS_URL"),
			NotificationSQS:   required("NOTIFICATION_SQS_URL"),
		},
	}

	return cfg, nil
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

func (c *DBConfig) URL() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		c.User, c.Password, c.Host, c.Port, c.Name, c.SSLMode,
	)
}
