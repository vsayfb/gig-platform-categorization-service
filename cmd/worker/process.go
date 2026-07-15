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

	// categorize text
	cat, err := a.categoryService.Resolve(ctx, msg.Title, msg.Description)
	if err != nil {
		slog.ErrorContext(ctx, "categorize text error", "err", err, "gig_id", msg.GigID)
		return err
	}

	slog.InfoContext(ctx, "categorized text", "category", cat)

	// find matching subscribers by category
	subscribers, err := a.subscriberRepo.FindByCategory(ctx, cat.ID)

	slog.InfoContext(ctx, "found subscribers", "subs", subscribers)

	if err != nil {
		return err
	}

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
				slog.ErrorContext(ctx, "failed to publish notification",
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
		slog.WarnContext(ctx, "some notifications failed to publish",
			"failed", n,
			"total", len(subscribers),
			"gig_id", msg.GigID,
		)
	}

	return nil
}

func (a *App) CategorizationSQSQueue() string {
	return a.cfg.AWS.CategorizationEventsQueue
}
