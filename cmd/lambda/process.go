package main

import (
	"context"
	"encoding/json"
	"log"

	"github.com/aws/aws-lambda-go/events"
	"github.com/google/uuid"
)

type Location struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}

type GigCreatedMessage struct {
	GigID       uuid.UUID `json:"gig_id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Location    Location  `json:"location"`
}

func (a *App) process(ctx context.Context, record events.SQSMessage) error {
	var msg GigCreatedMessage

	if err := json.Unmarshal([]byte(record.Body), &msg); err != nil {
		return err
	}

	log.Printf("processing gig %s", msg.GigID)

	// Categorize text
	cat, err := a.categoryService.Resolve(ctx, msg.Title, msg.Description)
	if err != nil {
		return err
	}

	// Find matching subscribers by category + location
	subscribers, err := a.subscriberRepo.FindByCategoryAndLocation(ctx, cat.ID, msg.Location.Lat, msg.Location.Lng)

	if err != nil {
		return err
	}

	log.Printf("found %d subscribers for gig %s", len(subscribers), msg.GigID)

	// 3. Fan out to Notification Lambda
	for _, s := range subscribers {
		if err := a.notificationPublisher.Publish(ctx, s.FCMToken, msg.GigID); err != nil {
			log.Printf("failed to publish notification for subscriber %s: %v", s.ID, err)
		}
	}

	return nil
}
