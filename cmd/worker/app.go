package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"sync"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	pgxvec "github.com/pgvector/pgvector-go/pgx"
	lg "github.com/vsayfb/gig-platform-categorization-service/pkg/logger"

	"github.com/vsayfb/gig-platform-categorization-service/internal/category"
	"github.com/vsayfb/gig-platform-categorization-service/internal/config"
	"github.com/vsayfb/gig-platform-categorization-service/internal/extractor"
	"github.com/vsayfb/gig-platform-categorization-service/internal/notification"
	"github.com/vsayfb/gig-platform-categorization-service/internal/subscriber"
	"github.com/vsayfb/gig-platform-categorization-service/pkg/embeddings"
	"github.com/vsayfb/gig-platform-categorization-service/pkg/metrics"
	"github.com/vsayfb/gig-platform-categorization-service/pkg/tracing"
)

type App struct {
	cfg *config.Config

	categoryService       *category.Service
	subscriberRepo        *subscriber.Repository
	notificationPublisher *notification.SQSPublisher

	db             *pgxpool.Pool
	metricsServer  *http.Server
	tracerShutdown func(context.Context) error
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

		metricsServer := metrics.StartServer(
			cfg.MetricsServerPort,
		)

		shutdownTracer, err := tracing.InitTracer(
			ctx,
			"core-service",
			cfg.OTelCollectorAddr,
		)

		if err != nil {
			initErr = fmt.Errorf(
				"init tracer: %w",
				err,
			)
			return
		}

		poolCfg, err := pgxpool.ParseConfig(cfg.DatabaseURL)

		if err != nil {
			initErr = fmt.Errorf(
				"parse database url: %w",
				err,
			)
			return
		}

		poolCfg.AfterConnect = func(
			ctx context.Context,
			conn *pgx.Conn,
		) error {
			return pgxvec.RegisterTypes(ctx, conn)
		}

		db, err := pgxpool.NewWithConfig(
			ctx,
			poolCfg,
		)

		if err != nil {
			initErr = fmt.Errorf(
				"create db pool: %w",
				err,
			)
			return
		}

		if err := db.Ping(ctx); err != nil {
			initErr = fmt.Errorf(
				"ping db: %w",
				err,
			)
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
			initErr = fmt.Errorf(
				"create sqs publisher: %w",
				err,
			)
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

			db: db,

			metricsServer: metricsServer,

			tracerShutdown: shutdownTracer,
		}
	})

	return app, initErr
}

func (a *App) Close(ctx context.Context) error {

	slog.Info("closing application")

	if a.metricsServer != nil {
		if err := a.metricsServer.Shutdown(ctx); err != nil {
			slog.Error(
				"shutdown metrics server failed",
				"err",
				err,
			)
		}
	}

	if a.tracerShutdown != nil {
		if err := a.tracerShutdown(ctx); err != nil {
			slog.Error(
				"shutdown tracer failed",
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
