package config

func loadEnv() (*Config, error) {
	return &Config{
		App: APP{
			ServiceName: getEnv("SERVICE_NAME", "categorization-worker"),
			Env:         getEnv("ENV", "development"),
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

		SQS: SQS{
			CategorizationSQS: required("CATEGORIZATION_SQS_URL"),
			NotificationSQS:   required("NOTIFICATION_SQS_URL"),
		},

		AI: AI{
			API_KEY:             required("AI_API_KEY"),
			API_ENDPOINT:        required("AI_API_ENDPOINT"),
			Model:               required("AI_MODEL"),
			HuggingFaceAIModel:  getEnv("HUGGINGFACE_AI_MODEL", "all-minilm:l12-v2"),
			LocalOllamaEndpoint: getEnv("LOCAL_OLLAMA_ENDPOINT", "localhost:11434"),
			PromptFile:          getEnv("PROMPT_FILE", "./internal/prompter/prompts/profession.txt"),
		},
	}, nil
}
