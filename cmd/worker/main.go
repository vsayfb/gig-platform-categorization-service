package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	awscfg "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"

	"github.com/vsayfb/gig-platform-categorization-service/internal/worker"
)

func main() {

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)

	defer stop()

	app, err := getApp(ctx)

	if err != nil {
		slog.Error("failed to initialize app", "err", err)

		os.Exit(1)
	}

	w, err := worker.New(app)

	if err != nil {
		slog.Error("failed to initialize worker", "err", err)

		os.Exit(1)
	}

	slog.Info("worker is running", "queue", app.QueueURL())

	_, err = awscfg.LoadDefaultConfig(
		ctx,
		awscfg.WithRegion("us-east-1"),
		awscfg.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider("test", "test", ""),
		),
	)

	if err != nil {
		slog.Error("failed to load aws config", "err", err)

		os.Exit(1)
	}

	if err := w.Run(ctx); err != nil &&
		err != context.Canceled {

		slog.Error("worker stopped", "err", err)

		os.Exit(1)
	}

	shutdownCtx, cancel := context.WithTimeout(
		context.Background(),
		10*time.Second,
	)

	defer cancel()

	if err := app.Close(shutdownCtx); err != nil {
		slog.Error("application shutdown failed", "err", err)
	}

	slog.Info("worker shut down gracefully")
}
