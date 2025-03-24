package handler

import (
	"context"

	"github.com/akaswenwilk/sqs-dlq-ui/model"
)

type Repo interface {
	ListQueues(ctx context.Context, page, size int, search string) ([]model.QueueInfo, int, error)
	FetchMessages(ctx context.Context, queueName string) ([]model.Message, int, error)
	DeleteMessage(ctx context.Context, queueName, messageHandle string) error
	PurgeQueue(ctx context.Context, queueName string) error
	ListDeadLetterSourceQueues(ctx context.Context, queueName string) ([]model.QueueInfo, error)
	PublishMessage(ctx context.Context, queueURL string, message model.Message) error
}
