package worker

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/vsayfb/gig-platform-categorization-service/pkg/metrics"
	"github.com/vsayfb/gig-platform-categorization-service/pkg/tracing"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
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

	client := sqs.NewFromConfig(cfg, func(o *sqs.Options) {
		o.BaseEndpoint = aws.String(os.Getenv("AWS_SQS_ENDPOINT"))
	})

	return &Worker{
		client:    client,
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
			MessageAttributeNames: []string{
				"traceparent", "tracestate",
			},
		})

		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}

			slog.Error("receive failed", "err", err)

			continue
		}

		for _, m := range resp.Messages {
			mCtx := tracing.ExtractTraceContext(ctx, m.MessageAttributes)

			mCtx, span := otel.Tracer("worker").Start(mCtx, "categorize-message")

			if err := w.processMessage(mCtx, m); err != nil {
				span.RecordError(err)
				span.SetStatus(codes.Error, err.Error())
				slog.ErrorContext(mCtx, "processing failed", "err", err)
			} else {
				span.SetStatus(codes.Ok, "processing succeed")
			}

			span.End()
		}
	}
}

func (w *Worker) processMessage(ctx context.Context, m types.Message) (err error) {
	start := time.Now()

	defer func() {
		metrics.ObserveWorkerProcessing(start, err)
	}()

	msg := Message{
		ID:   aws.ToString(m.MessageId),
		Body: aws.ToString(m.Body),
	}

	if err = w.processor.Process(ctx, msg); err != nil {
		return err
	}

	_, err = w.client.DeleteMessage(ctx, &sqs.DeleteMessageInput{
		QueueUrl:      aws.String(w.processor.QueueURL()),
		ReceiptHandle: m.ReceiptHandle,
	})

	return err
}
