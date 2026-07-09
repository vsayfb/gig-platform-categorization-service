package main

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vsayfb/gig-platform-categorization-service/internal/category"
	"github.com/vsayfb/gig-platform-categorization-service/internal/config"
	"github.com/vsayfb/gig-platform-categorization-service/internal/extractor"
	"github.com/vsayfb/gig-platform-categorization-service/internal/notification"
	"github.com/vsayfb/gig-platform-categorization-service/internal/subscriber"
	"github.com/vsayfb/gig-platform-categorization-service/pkg/embeddings"
	"github.com/vsayfb/gig-platform-categorization-service/pkg/metrics"
	"github.com/vsayfb/gig-platform-categorization-service/pkg/telemetry"
	"github.com/vsayfb/gig-platform-categorization-service/pkg/tracing"

	pgxvec "github.com/pgvector/pgvector-go/pgx"
	lg "github.com/vsayfb/gig-platform-categorization-service/pkg/logger"
)

type App struct {
	cfg *config.Config

	categoryService       *category.Service
	subscriberRepo        *subscriber.Repository
	notificationPublisher *notification.SQSPublisher

	db *pgxpool.Pool

	telemetryShutdown func(context.Context) error
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

		logHandler := lg.Init(cfg.Env)

		shutdownTelemetry, err := telemetry.Init(ctx, cfg.ServiceName, cfg.OTelCollectorAddr)

		if err != nil {
			initErr = fmt.Errorf("initialize telemetry: %w", err)
			return
		}

		if err := metrics.Register(); err != nil {
			initErr = fmt.Errorf("register metrics: %w", err)
			return
		}

		slog.SetDefault(slog.New(tracing.NewOTelHandler(logHandler)))

		poolCfg, err := pgxpool.ParseConfig(cfg.DatabaseURL)

		if err != nil {
			initErr = fmt.Errorf("parse database url: %w", err)
			return
		}

		poolCfg.AfterConnect = func(
			ctx context.Context,
			conn *pgx.Conn,
		) error {
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

		groqClient := extractor.NewGroqExtractor(
			embeddingClient,
			cfg,
		)

		publisher, err := notification.NewSQSPublisher(
			ctx,
			cfg.NotificationSQS,
		)

		if err != nil {
			initErr = fmt.Errorf("create sqs publisher: %w", err)
			return
		}

		app = &App{
			cfg:                   cfg,
			categoryService:       category.NewService(category.NewRepository(db), embeddingClient, groqClient, cfg),
			subscriberRepo:        subscriber.NewRepository(db),
			notificationPublisher: publisher,
			db:                    db,
			telemetryShutdown:     shutdownTelemetry,
		}
	})

	return app, initErr
}

func (a *App) Close(ctx context.Context) error {

	slog.Info("closing application")

	if a.telemetryShutdown != nil {
		if err := a.telemetryShutdown(ctx); err != nil {
			slog.Error(
				"shutdown telemetry failed",
				"err",
				err,
			)
		}
	}

	if a.notificationPublisher != nil {
		if err := a.notificationPublisher.Close(); err != nil {
			slog.Error(
				"close sqs publisher failed",
				"err",
				err,
			)
		}
	}

	if a.db != nil {
		a.db.Close()
	}

	slog.Info("application closed")

	return nil
}
