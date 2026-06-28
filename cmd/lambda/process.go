package main

import (
	"context"
	"encoding/json"
	"log"

	"github.com/aws/aws-lambda-go/events"
	"github.com/google/uuid"
)

type GigCreatedMessage struct {
	GigID uuid.UUID `json:"gig_id"`
}

func (a *App) process(ctx context.Context, record events.SQSMessage) error {
	var msg GigCreatedMessage

	if err := json.Unmarshal([]byte(record.Body), &msg); err != nil {
		return err
	}

	log.Printf("processing gig %s", msg.GigID)

	cat, err := a.categoryService.ResolveForGig(ctx, msg.GigID)
	if err != nil {
		return err
	}

	providers, err := a.providerRepo.FindByGig(ctx, msg.GigID, cat.ID)
	if err != nil {
		return err
	}

	log.Printf("found %d providers for gig %s", len(providers), msg.GigID)

	for _, p := range providers {
		if err := a.notificationPublisher.Publish(ctx, p.FCMToken, msg.GigID); err != nil {
			log.Printf("failed to publish notification for provider %s: %v", p.ID, err)
		}
	}

	return nil
}
