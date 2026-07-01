package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync/atomic"

	"github.com/google/uuid"
	"github.com/vsayfb/gig-platform-categorization-service/internal/worker"
	"golang.org/x/sync/errgroup"
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

type Subscriber struct {
	UserID   uuid.UUID `json:"user_id"`
	FCMToken string    `json:"fcm_token"`
}

type GigCreatedNotification struct {
	GigCreatedMessage
	Subscriber
}

const maxFanoutConcurrency = 10

func (a *App) Process(ctx context.Context, record worker.Message) error {
	var msg GigCreatedMessage

	if err := json.Unmarshal([]byte(record.Body), &msg); err != nil {
		return err
	}

	slog.Info("processing gig", "gig_id", msg.GigID)

	// categorize text
	cat, err := a.categoryService.Resolve(ctx, msg.Title, msg.Description)
	if err != nil {
		slog.Error("categorize text error", "err", err, "gig_id", msg.GigID)
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

	// fanout to notification lambda, bounded concurrency, best-effort
	g, gCtx := errgroup.WithContext(ctx)

	g.SetLimit(maxFanoutConcurrency)

	var failed atomic.Int64

	for _, s := range subscribers {
		g.Go(func() error {
			select {
			case <-gCtx.Done():
				return nil // don't propagate cancellation as a fanout failure; best-effort
			default:
			}

			n := GigCreatedNotification{
				GigCreatedMessage: msg,
				Subscriber: Subscriber{
					UserID:   s.ID,
					FCMToken: s.FCMToken,
				},
			}

			if err := a.notificationPublisher.Publish(gCtx, n); err != nil {
				slog.Error("failed to publish notification",
					"err", err,
					"subscriber_id", s.ID,
					"gig_id", msg.GigID,
				)
				failed.Add(1)
			}

			return nil
		})
	}

	_ = g.Wait() // every g.Go() returns nil, so this is always nil; kept for goroutine cleanup

	if n := failed.Load(); n > 0 {
		slog.Warn("some notifications failed to publish",
			"failed", n,
			"total", len(subscribers),
			"gig_id", msg.GigID,
		)
	}

	return nil
}

func (a *App) QueueURL() string {
	return a.cfg.CategorizationSQS
}
