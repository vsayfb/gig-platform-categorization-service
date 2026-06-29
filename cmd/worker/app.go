package main

import (
	"context"
	"sync"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	pgxvec "github.com/pgvector/pgvector-go/pgx"
	"github.com/vsayfb/gig-platform-categorization-service/internal/category"
	"github.com/vsayfb/gig-platform-categorization-service/internal/config"
	"github.com/vsayfb/gig-platform-categorization-service/internal/notification"
	"github.com/vsayfb/gig-platform-categorization-service/internal/subscriber"
	"github.com/vsayfb/gig-platform-categorization-service/pkg/embeddings"
)

type App struct {
	cfg *config.Config

	categoryService       *category.Service
	subscriberRepo        *subscriber.Repository
	notificationPublisher *notification.SQSPublisher
}

var (
	app     *App
	once    sync.Once
	initErr error
)

func getApp(ctx context.Context) (*App, error) {
	once.Do(func() {
		cfg, err := config.Load()

		if err != nil {
			initErr = err
			return
		}

		db, err := pgxpool.New(ctx, cfg.DatabaseURL)

		db.Config().AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
			return pgxvec.RegisterTypes(ctx, conn)
		}

		if err != nil {
			initErr = err
			return
		}

		embeddingClient := embeddings.NewLocalClient(cfg)

		app = &App{
			cfg: cfg,

			categoryService: category.NewService(
				category.NewRepository(db),
				embeddingClient,
				cfg,
			),

			subscriberRepo: subscriber.NewRepository(db),

			notificationPublisher: notification.NewSQSPublisher(
				cfg.NotificationSQS,
			),
		}
	})

	return app, initErr
}
