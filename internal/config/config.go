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
	Env                 string
	MetricsServerPort   string
	ServiceName         string
	OTelCollectorAddr   string
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
		Env:                 required("ENV"),
		MetricsServerPort:   required("METRICS_SERVER_PORT"),
		OTelCollectorAddr:   required("OTEL_COLLECTOR_ADDR"),
		ServiceName:         required("SERVICE_NAME"),

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
