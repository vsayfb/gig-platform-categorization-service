package main

import (
	"context"
	"encoding/json"
	"log"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/vsayfb/gig-platform-categorization-service/internal/category"
	"github.com/vsayfb/gig-platform-categorization-service/internal/config"
	"github.com/vsayfb/gig-platform-categorization-service/internal/notification"
	"github.com/vsayfb/gig-platform-categorization-service/internal/prompter"
	"github.com/vsayfb/gig-platform-categorization-service/internal/provider"
	"github.com/vsayfb/gig-platform-categorization-service/pkg/embeddings"
)

var (
	categoryService       *category.Service
	providerRepo          *provider.Repository
	notificationPublisher *notification.SQSPublisher
)

func init() {
	ctx := context.Background()

	cfg, err := config.Load()

	if err != nil {
		log.Fatal(err)
	}

	db, err := pgxpool.New(ctx, cfg.DatabaseURL)

	if err != nil {
		log.Fatalf("failed to connect to postgres: %v", err)
	}

	if err := prompter.Init(cfg.PromptFile); err != nil {
		log.Fatal(err)
	}

	embeddingClient := embeddings.NewHuggingFaceClient(cfg)
	categoryRepo := category.NewRepository(db)
	categoryService = category.NewService(categoryRepo, embeddingClient, cfg)
	providerRepo = provider.NewRepository(db)
	notificationPublisher = notification.NewSQSPublisher(cfg.NotificationSQS)
}

type GigCreatedMessage struct {
	GigID uuid.UUID `json:"gig_id"`
}

func handler(ctx context.Context, event events.SQSEvent) error {
	for _, record := range event.Records {
		if err := process(ctx, record); err != nil {
			// Log but continue — failed messages return to queue via SQS visibility timeout
			log.Printf("failed to process record %s: %v", record.MessageId, err)
		}
	}
	return nil
}

func process(ctx context.Context, record events.SQSMessage) error {
	var msg GigCreatedMessage

	if err := json.Unmarshal([]byte(record.Body), &msg); err != nil {
		return err
	}

	log.Printf("processing gig %s", msg.GigID)

	// 1. Extract profession, match or create category
	cat, err := categoryService.ResolveForGig(ctx, msg.GigID)

	if err != nil {
		return err
	}

	// 2. Find matching providers by category + geo
	providers, err := providerRepo.FindByGig(ctx, msg.GigID, cat.ID)

	if err != nil {
		return err
	}

	log.Printf("found %d providers for gig %s", len(providers), msg.GigID)

	// 3. Publish each provider to Notification Lambda via SQS
	for _, p := range providers {
		if err := notificationPublisher.Publish(ctx, p.FCMToken, msg.GigID); err != nil {
			log.Printf("failed to publish notification for provider %s: %v", p.ID, err)
		}
	}

	return nil
}

func main() {
	lambda.Start(handler)
}
