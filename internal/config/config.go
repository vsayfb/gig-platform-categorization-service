package config

import (
	"fmt"
	"os"
)

type Config struct {
	DatabaseURL         string
	CategorizationSQS   string
	NotificationSQS     string
	AI_API_KEY          string
	AI_API_ENDPOINT     string
	AI_MODEL            string
	HuggingFaceAIModel  string
	PromptFile          string
	LocalOllamaEndpoint string
	AppEnv              string
	MetricsServerPort   string
	ServiceName         string
	OTelCollectorAddr   string
	AWSRegion           string
}

func Load() (*Config, error) {
	cfg := &Config{
		DatabaseURL:         required("DATABASE_URL"),
		CategorizationSQS:   required("CATEGORIZATION_SQS_URL"),
		NotificationSQS:     required("NOTIFICATION_SQS_URL"),
		AI_API_KEY:          required("AI_API_KEY"),
		AI_API_ENDPOINT:     required("AI_API_ENDPOINT"),
		AI_MODEL:            required("AI_MODEL"),
		HuggingFaceAIModel:  required("HUGGINGFACE_AI_MODEL"),
		LocalOllamaEndpoint: required("LOCAL_OLLAMA_ENDPOINT"),
		AppEnv:              getEnv("ENV", "production"),
		MetricsServerPort:   required("METRICS_SERVER_PORT"),
		OTelCollectorAddr:   getEnv("OTEL_COLLECTOR_ADDR", "localhost:4317"),
		ServiceName:         required("SERVICE_NAME"),
		AWSRegion:           required("AWS_REGION"),

		// Optional
		PromptFile: os.Getenv("PROMPT_FILE"),
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
