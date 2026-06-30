package main

import (
	"context"
	"fmt"
	"sync"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	pgxvec "github.com/pgvector/pgvector-go/pgx"
	"github.com/vsayfb/gig-platform-categorization-service/internal/category"
	"github.com/vsayfb/gig-platform-categorization-service/internal/config"
	"github.com/vsayfb/gig-platform-categorization-service/internal/extractor"
	"github.com/vsayfb/gig-platform-categorization-service/internal/notification"
	"github.com/vsayfb/gig-platform-categorization-service/internal/subscriber"
	"github.com/vsayfb/gig-platform-categorization-service/pkg/embeddings"
	lg "github.com/vsayfb/gig-platform-categorization-service/pkg/logger"
	"github.com/vsayfb/gig-platform-categorization-service/pkg/metrics"
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
			initErr = fmt.Errorf("load config: %w", err)
			return
		}

		lg.Init(cfg.Env)

		metrics.Register()

		poolCfg, err := pgxpool.ParseConfig(cfg.DatabaseURL)

		if err != nil {
			initErr = fmt.Errorf("parse database url: %w", err)
			return
		}

		poolCfg.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
			return pgxvec.RegisterTypes(ctx, conn)
		}

		db, err := pgxpool.NewWithConfig(ctx, poolCfg)

		if err != nil {
			initErr = fmt.Errorf("create db pool: %w", err)
			return
		}

		if err := db.Ping(ctx); err != nil {
			initErr = fmt.Errorf("ping db: %w", err)
			return
		}

		embeddingClient := embeddings.NewLocalClient(cfg)
		groqClient := extractor.NewGroqExtractor(embeddingClient, cfg)

		publisher, err := notification.NewSQSPublisher(ctx, cfg.NotificationSQS)

		if err != nil {
			initErr = fmt.Errorf("create sqs publisher: %w", err)
			return
		}

		app = &App{
			cfg: cfg,

			categoryService: category.NewService(
				category.NewRepository(db),
				embeddingClient,
				groqClient,
				cfg,
			),

			subscriberRepo: subscriber.NewRepository(db),

			notificationPublisher: publisher,
		}
	})

	return app, initErr
}
