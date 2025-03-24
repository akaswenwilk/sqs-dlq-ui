package repo

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/akaswenwilk/sqs-dlq-ui/model"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/samber/lo"
)

const waitMinutes = 120

type repo struct {
	client *sqs.Client
	queues []Queue
	mu     sync.RWMutex
}

type Queue struct {
	Name string
	URL  string
}

func New(client *sqs.Client) *repo {
	r := &repo{client: client}
	go r.watchQueues()
	return r
}

func (r *repo) watchQueues() {
	for {
		ctx := context.Background()
		var (
			gatheredQueues []Queue
			nextToken      *string
		)
		for {
			result, err := r.client.ListQueues(ctx, &sqs.ListQueuesInput{
				MaxResults: aws.Int32(1000),
				NextToken: nextToken,
			})
			if err != nil {
				slog.ErrorContext(ctx, fmt.Sprintf("failed to list queues: %v", err))
				continue
			}
			for _, url := range result.QueueUrls {
				segments := strings.Split(url, "/")
				gatheredQueues = append(gatheredQueues, Queue{Name: segments[len(segments)-1], URL: url})
			}
			if result.NextToken == nil {
				break
			}
			nextToken = result.NextToken
		}
		r.mu.Lock()
		r.queues = gatheredQueues
		r.mu.Unlock()
		time.Sleep(waitMinutes * time.Minute)
	}
}

func (r *repo) ListQueues(ctx context.Context, page, size int, search string) ([]model.QueueInfo, int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var queues []Queue
	allQueues := r.queues
	if search != "" {
		allQueues = lo.Filter(r.queues, func(q Queue, _ int) bool {
			return strings.Contains(q.Name, search)
		})
	}
	for i := page*size - size; i < page*size; i++ {
		if i >= len(allQueues) {
			break
		}
		queues = append(queues, allQueues[i])
	}
	return lo.Map(queues, func(q Queue, _ int) model.QueueInfo {
		count, err := r.getMessageCount(ctx, q.URL)
		if err != nil {
			slog.ErrorContext(ctx, fmt.Sprintf("failed to get message count for queue %s: %v", q.Name, err))
			return model.QueueInfo{
				URL:          q.URL,
				Name:         q.Name,
				MessageCount: -1,
			}
		}
		return model.QueueInfo{
			URL:          q.URL,
			Name:         q.Name,
			MessageCount: count,
		}
	}), len(allQueues), nil
}

func (r *repo) FetchMessages(ctx context.Context, queueName string) ([]model.Message, int, error) {
	queueURL, err := r.getQueueURLByName(ctx, queueName)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get queue URL: %v", err)
	}
	count, err := r.getMessageCount(ctx, queueURL)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get message count: %v", err)
	}
	var messages []model.Message
	for i := 0; i < count; i += 10 {
		res, err := r.client.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
			QueueUrl:              &queueURL,
			MaxNumberOfMessages:   10,
			VisibilityTimeout:     3,
			MessageAttributeNames: []string{string(types.QueueAttributeNameAll)},
			MessageSystemAttributeNames: []types.MessageSystemAttributeName{
				types.MessageSystemAttributeNameMessageDeduplicationId, types.MessageSystemAttributeNameMessageGroupId,
			},
		})
		if err != nil {
			return nil, 0, fmt.Errorf("failed to receive messages: %v", err)
		}
		for _, msg := range res.Messages {
			meta := make(map[string]string)
			for k, v := range msg.MessageAttributes {
				meta[k] = *v.StringValue
			}
			systemMeta := make(map[string]string)
			for k, v := range msg.Attributes {
				systemMeta[k] = v
			}
			messages = append(messages, model.Message{
				MessageId:     *msg.MessageId,
				ReceiptHandle: *msg.ReceiptHandle,
				Body:          *msg.Body,
				Attributes:    meta,
				SystemAttributes: systemMeta,
			})
		}
	}
	return lo.UniqBy(messages, func(msg model.Message) string {
		return msg.MessageId
	}), count, nil
}

func (r *repo) getMessageCount(ctx context.Context, queueURL string) (int, error) {
	res, err := r.client.GetQueueAttributes(ctx, &sqs.GetQueueAttributesInput{
		QueueUrl: &queueURL,
		AttributeNames: []types.QueueAttributeName{
			types.QueueAttributeNameApproximateNumberOfMessages,
		},
	})
	if err != nil {
		return 0, fmt.Errorf("failed to get queue attributes: %v", err)
	}
	count, _ := strconv.Atoi(res.Attributes[string(types.QueueAttributeNameApproximateNumberOfMessages)])
	return count, nil
}

func (r *repo) getQueueURLByName(_ context.Context, queueName string) (string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	queue, found := lo.Find(r.queues, func(q Queue) bool {
		return q.Name == queueName
	})
	if !found {
		return "", fmt.Errorf("queue not found")
	}
	return queue.URL, nil
}

func (r *repo) DeleteMessage(ctx context.Context, queueName, messageHandle string) error {
	queueURL, err := r.getQueueURLByName(ctx, queueName)
	if err != nil {
		return fmt.Errorf("failed to get queue URL: %v", err)
	}
	_, err = r.client.DeleteMessage(ctx, &sqs.DeleteMessageInput{
		QueueUrl:      &queueURL,
		ReceiptHandle: &messageHandle,
	})
	if err != nil {
		return fmt.Errorf("failed to delete message: %v", err)
	}
	return nil
}

func (r *repo) PurgeQueue(ctx context.Context, queueName string) error {
	queueURL, err := r.getQueueURLByName(ctx, queueName)
	if err != nil {
		return fmt.Errorf("failed to get queue URL: %v", err)
	}
	_, err = r.client.PurgeQueue(context.TODO(), &sqs.PurgeQueueInput{
		QueueUrl: &queueURL,
	})
	if err != nil {
		return fmt.Errorf("failed to purge queue: %v", err)
	}
	return nil
}

func (r *repo) ListDeadLetterSourceQueues(ctx context.Context, queueName string) ([]model.QueueInfo, error) {
	queueURL, err := r.getQueueURLByName(ctx, queueName)
	if err != nil {
		return nil, fmt.Errorf("failed to get queue URL: %v", err)
	}
	var (
		queues    []model.QueueInfo
		nextToken *string
	)
	for {
		res, err := r.client.ListDeadLetterSourceQueues(ctx, &sqs.ListDeadLetterSourceQueuesInput{
			QueueUrl:  &queueURL,
			NextToken: nextToken,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to list dead letter source queues: %v", err)
		}

		for _, url := range res.QueueUrls {
			segments := strings.Split(url, "/")
			name := segments[len(segments)-1]
			queues = append(queues, model.QueueInfo{
				URL:  url,
				Name: name,
			})
		}
		if res.NextToken == nil {
			break
		}
		nextToken = res.NextToken
	}

	return queues, nil
}

func (r *repo) PublishMessage(ctx context.Context, queueURL string, message model.Message) error {
	attrs := make(map[string]types.MessageAttributeValue)
	for k, v := range message.Attributes {
		if v == "" {
			continue
		}
		attrs[k] = types.MessageAttributeValue{
			DataType:    aws.String("String"),
			StringValue: aws.String(v),
		}
	}
	systemAttrs := make(map[string]types.MessageSystemAttributeValue)
	for k, v := range message.SystemAttributes {
		if v == "" {
			continue
		}
		systemAttrs[k] = types.MessageSystemAttributeValue{
			DataType: aws.String("String"),
			StringValue: aws.String(v),
		}
	}
			
	input := &sqs.SendMessageInput{
		QueueUrl:    &queueURL,
		MessageBody: &message.Body,
	}
	if len(attrs) > 0 {
		input.MessageAttributes = attrs
	}
	if len(systemAttrs) > 0 {
		input.MessageSystemAttributes = systemAttrs
	}
	_, err := r.client.SendMessage(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to send message: %v", err)
	}
	return nil
}
