package config

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	awscfg "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
)

type rdsSecret struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type groqSecret struct {
	APIKey string `json:"api_key"`
}

const parameterPath = "/gerek/app"

func loadAWS(ctx context.Context) (*Config, error) {

	awsCfg, err := awscfg.LoadDefaultConfig(ctx)

	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}

	ssmClient := ssm.NewFromConfig(awsCfg)
	secretsClient := secretsmanager.NewFromConfig(awsCfg)

	params, err := loadParameters(ctx, ssmClient)

	if err != nil {
		return nil, err
	}

	var db rdsSecret

	if err := loadSecret(ctx, secretsClient, params["rds-secret-arn"], &db); err != nil {
		return nil, err
	}

	var groq groqSecret

	if err := loadSecret(ctx, secretsClient, params["groq-api-key-secret-arn"], &groq); err != nil {
		return nil, err
	}

	return &Config{
		App: APP{
			ServiceName: getOrDefault(params, "service-name", "categorization-worker"),
			Env:         "production",
			AWSRegion:   awsCfg.Region,
		},

		DB: DBConfig{
			Host:     params["db-host"],
			Port:     params["db-port"],
			Name:     params["db-name"],
			User:     db.Username,
			Password: db.Password,
			SSLMode:  getOrDefault(params, "db-sslmode", "require"),
		},

		Server: ServerConfig{
			MetricsServerPort: getOrDefault(params, "metrics-server-port", ":9100"),
			OTelCollectorAddr: getOrDefault(params, "otel-collector-addr", "localhost:4317"),
		},

		SQS: SQS{
			CategorizationSQS: params["sqs-category-events-queue-url"],
			NotificationSQS:   getOrDefault(params, "sqs-notification-events-queue-url", ""),
		},

		AI: AI{
			API_KEY:             groq.APIKey,
			API_ENDPOINT:        params["groq-ai-endpoint"],
			Model:               params["groq-ai-model"],
			HuggingFaceAIModel:  getOrDefault(params, "huggingface-ai-model", "all-minilm:l12-v2"),
			LocalOllamaEndpoint: getOrDefault(params, "local-ollama-endpoint", "localhost:11434"),
			PromptFile:          getOrDefault(params, "prompt-file", "./internal/prompter/prompts/profession.txt"),
		},
	}, nil
}

func loadParameters(ctx context.Context, client *ssm.Client) (map[string]string, error) {
	params := make(map[string]string)

	var nextToken *string

	for {
		out, err := client.GetParametersByPath(ctx, &ssm.GetParametersByPathInput{
			Path:           aws.String(parameterPath),
			Recursive:      aws.Bool(true),
			WithDecryption: aws.Bool(true),
			NextToken:      nextToken,
		})

		if err != nil {
			return nil, fmt.Errorf("read parameter store: %w", err)
		}

		for _, p := range out.Parameters {
			name := strings.TrimPrefix(aws.ToString(p.Name), parameterPath+"/")
			params[name] = aws.ToString(p.Value)
		}

		if out.NextToken == nil {
			break
		}

		nextToken = out.NextToken
	}

	return params, nil
}

func loadSecret(
	ctx context.Context,
	client *secretsmanager.Client,
	name string,
	dst any,
) error {

	out, err := client.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{
		SecretId: aws.String(name),
	})

	if err != nil {
		return err
	}

	return json.Unmarshal([]byte(aws.ToString(out.SecretString)), dst)
}

func getOrDefault(values map[string]string, key, def string) string {
	envKey := strings.ToUpper(strings.ReplaceAll(key, "-", "_"))

	if v := os.Getenv(envKey); v != "" {
		return v
	}

	if v, ok := values[key]; ok && v != "" {
		return v
	}

	return def
}
