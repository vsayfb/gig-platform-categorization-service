package main

import (
	"context"
	"sync"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/vsayfb/gig-platform-categorization-service/internal/category"
	"github.com/vsayfb/gig-platform-categorization-service/internal/config"
	"github.com/vsayfb/gig-platform-categorization-service/internal/notification"
	"github.com/vsayfb/gig-platform-categorization-service/internal/prompter"
	"github.com/vsayfb/gig-platform-categorization-service/internal/provider"
	"github.com/vsayfb/gig-platform-categorization-service/pkg/embeddings"
)

type App struct {
	categoryService       *category.Service
	providerRepo          *provider.Repository
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
		if err != nil {
			initErr = err
			return
		}

		if err := prompter.Init(cfg.PromptFile); err != nil {
			initErr = err
			return
		}

		embeddingClient := embeddings.NewHuggingFaceClient(cfg)

		app = &App{
			categoryService: category.NewService(
				category.NewRepository(db),
				embeddingClient,
				cfg,
			),
			providerRepo: provider.NewRepository(db),
			notificationPublisher: notification.NewSQSPublisher(
				cfg.NotificationSQS,
			),
		}
	})

	return app, initErr
}
