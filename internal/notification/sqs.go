package notification

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/google/uuid"
)

type NotificationMessage struct {
	FCMToken string    `json:"fcm_token"`
	GigID    uuid.UUID `json:"gig_id"`
}

type SQSPublisher struct {
	queueURL string
	client   *sqs.Client
}

func NewSQSPublisher(queueURL string) *SQSPublisher {
	cfg, _ := config.LoadDefaultConfig(context.Background())
	return &SQSPublisher{
		queueURL: queueURL,
		client:   sqs.NewFromConfig(cfg),
	}
}

func (p *SQSPublisher) Publish(ctx context.Context, fcmToken string, gigID uuid.UUID) error {
	msg := NotificationMessage{
		FCMToken: fcmToken,
		GigID:    gigID,
	}

	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal notification message: %w", err)
	}

	bodyStr := string(body)
	_, err = p.client.SendMessage(ctx, &sqs.SendMessageInput{
		QueueUrl:    &p.queueURL,
		MessageBody: &bodyStr,
	})
	return err
}
