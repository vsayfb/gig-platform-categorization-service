package config

import (
	"fmt"
	"os"
)

type Config struct {
	DatabaseURL          string
	CategorizationSQS    string
	NotificationSQS      string
	AI_API_KEY           string
	AI_API_ENDPOINT      string
	AI_MODEL             string
	HuggingFaceAPIKey    string
	HUGGINGFACE_AI_MODEL string
	HUGGINGFACE_ENDPOINT string
	PromptFile           string
}

func Load() (*Config, error) {
	cfg := &Config{
		DatabaseURL:          required("DATABASE_URL"),
		CategorizationSQS:    required("CATEGORIZATION_SQS_URL"),
		NotificationSQS:      required("NOTIFICATION_SQS_URL"),
		AI_API_KEY:           required("AI_API_KEY"),
		AI_API_ENDPOINT:      required("AI_API_ENDPOINT"),
		AI_MODEL:             required("AI_MODEL"),
		HuggingFaceAPIKey:    required("HUGGINGFACE_API_KEY"),
		HUGGINGFACE_AI_MODEL: required("HUGGINGFACE_AI_MODEL"),
		HUGGINGFACE_ENDPOINT: required("HUGGINGFACE_ENDPOINT"),

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
