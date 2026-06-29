package main

import (
	"context"
	"encoding/json"
	"log"
	"log/slog"

	"github.com/google/uuid"
	"github.com/vsayfb/gig-platform-categorization-service/internal/worker"
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

func (a *App) Process(ctx context.Context, record worker.Message) error {
	var msg GigCreatedMessage

	if err := json.Unmarshal([]byte(record.Body), &msg); err != nil {
		return err
	}

	slog.Info("processing gig", "gigID", msg.GigID)

	// categorize text
	cat, err := a.categoryService.Resolve(ctx, msg.Title, msg.Description)
	if err != nil {
		slog.Error("categorize text error", "err", err)

		return err
	}

	// find matching subscribers by category + location
	subscribers, err := a.subscriberRepo.FindByCategoryAndLocation(ctx, cat.ID, msg.Location.Lat, msg.Location.Lng)

	if err != nil {
		return err
	}

	slog.Info(
		"found subscribers for gig",
		"count", len(subscribers),
		"gig_id", msg.GigID,
	)

	// fanout to notification lambda
	for _, s := range subscribers {
		if err := a.notificationPublisher.Publish(ctx, s.FCMToken, msg.GigID); err != nil {
			log.Printf("failed to publish notification for subscriber %s: %v", s.ID, err)
		}
	}

	return nil
}

func (a *App) QueueURL() string {
	return a.cfg.CategorizationSQS
}
