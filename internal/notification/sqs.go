package notification

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/vsayfb/gig-platform-categorization-service/internal/awsclient"
	"github.com/vsayfb/gig-platform-categorization-service/internal/config"
)

type SQSPublisher struct {
	queueURL string
	client   *sqs.Client
}

func NewNotificationPublisher(ctx context.Context, cfg *config.Config) (*SQSPublisher, error) {

	awsCfg, err := awsclient.New(ctx, cfg)

	if err != nil {
		return nil, fmt.Errorf("initialize aws client: %w", err)
	}

	sqs := awsclient.NewSQS(awsCfg, cfg)

	return &SQSPublisher{
		queueURL: cfg.AWS.NotificationEventsQueue,
		client:   sqs,
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

func (p *SQSPublisher) Close() error {
	return p.Close()
}
