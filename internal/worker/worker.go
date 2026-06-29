package worker

import (
	"context"
	"log/slog"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
)

type Worker struct {
	client    *sqs.Client
	processor Processor
}

func New(p Processor) (*Worker, error) {
	cfg, err := awsconfig.LoadDefaultConfig(context.Background())
	if err != nil {
		return nil, err
	}

	return &Worker{
		client:    sqs.NewFromConfig(cfg),
		processor: p,
	}, nil
}

func (w *Worker) Run(ctx context.Context) error {
	for {
		resp, err := w.client.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
			QueueUrl:            aws.String(w.processor.QueueURL()),
			MaxNumberOfMessages: 10,
			WaitTimeSeconds:     20,
			VisibilityTimeout:   60,
		})

		if err != nil {
			slog.Error("receive fialed", "err", err)
			time.Sleep(time.Second)
			continue
		}

		for _, m := range resp.Messages {
			msg := Message{
				ID:   aws.ToString(m.MessageId),
				Body: aws.ToString(m.Body),
			}

			if err := w.processor.Process(ctx, msg); err != nil {
				slog.Warn("processing failed", "err", err)

				continue
			}

			_, err = w.client.DeleteMessage(ctx, &sqs.DeleteMessageInput{
				QueueUrl:      aws.String(w.processor.QueueURL()),
				ReceiptHandle: m.ReceiptHandle,
			})

			if err != nil {
				slog.Error("delete failed", "err", err)
			}
		}
	}
}
