package notification

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
)

type SQSPublisher struct {
	queueURL string
	client   *sqs.Client
}

func NewSQSPublisher(ctx context.Context, queueURL string) (*SQSPublisher, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}

	return &SQSPublisher{
		queueURL: queueURL,
		client:   sqs.NewFromConfig(cfg),
	}, nil
}

func (p *SQSPublisher) Publish(ctx context.Context, msg any) error {
	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal notification message: %w", err)
	}

	bodyStr := string(body)

	if _, err := p.client.SendMessage(ctx, &sqs.SendMessageInput{
		QueueUrl:    &p.queueURL,
		MessageBody: &bodyStr,
	}); err != nil {
		return fmt.Errorf("send sqs message to %s: %w", p.queueURL, err)
	}

	return nil
}
