package main

import (
	"context"
	"log"

	"github.com/aws/aws-lambda-go/events"
)

func handler(ctx context.Context, event events.SQSEvent) error {
	app, err := getApp(ctx)
	if err != nil {
		return err
	}

	for _, record := range event.Records {
		if err := app.process(ctx, record); err != nil {
			log.Printf("failed to process record %s: %v", record.MessageId, err)
		}
	}

	return nil
}
