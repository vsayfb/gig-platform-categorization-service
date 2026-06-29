package main

import (
	"context"
	"log/slog"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
)

type Worker struct {
	app      *App
	client   *sqs.Client
	queueURL string
}

func NewWorker(app *App) *Worker {
	cfg, err := config.LoadDefaultConfig(context.Background())

	if err != nil {
		panic(err)
	}

	return &Worker{
		app:    app,
		client: sqs.NewFromConfig(cfg),
	}
}

func (w *Worker) Run(ctx context.Context) error {
	for {
		resp, err := w.client.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
			QueueUrl:            aws.String(w.queueURL),
			MaxNumberOfMessages: 10,
			WaitTimeSeconds:     20,
			VisibilityTimeout:   60,
		})

		if err != nil {
			slog.Error("processing failed", "err", err)

			time.Sleep(time.Second)
			continue
		}

		if len(resp.Messages) == 0 {
			continue
		}

		for _, msg := range resp.Messages {

			record := Message{
				ID:   aws.ToString(msg.MessageId),
				Body: aws.ToString(msg.Body),
			}

			if err := w.app.process(ctx, record); err != nil {
				slog.Error("processing failed", "err", err)

				continue
			}

			_, err := w.client.DeleteMessage(ctx, &sqs.DeleteMessageInput{
				QueueUrl:      aws.String(w.queueURL),
				ReceiptHandle: msg.ReceiptHandle,
			})

			if err != nil {
				slog.Error("delete failed", "err", err)

			}
		}
	}
}
