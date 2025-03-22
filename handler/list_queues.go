package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
)

type QueueInfo struct {
	URL          string `json:"url"`
	Name         string `json:"name"`
	MessageCount int    `json:"messageCount"`
}

func (h *Handler) ListQueues(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	size, _ := strconv.Atoi(r.URL.Query().Get("size"))
	if size < 1 {
		size = 10
	}
	prefix := r.URL.Query().Get("prefix")
	var queueURLs []string
	var prefixInput *string
	if prefix != "" {
		prefixInput = aws.String(prefix)
	}
	result, err := h.sqsClient.ListQueues(context.TODO(), &sqs.ListQueuesInput{
		QueueNamePrefix: prefixInput,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	queueURLs = append(queueURLs, result.QueueUrls...)
	for {
		if result.NextToken == nil || len(queueURLs) >= page*size {
			break
		}
		result, err = h.sqsClient.ListQueues(context.TODO(), &sqs.ListQueuesInput{
			NextToken: result.NextToken,
			QueueNamePrefix: prefixInput,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		queueURLs = append(queueURLs, result.QueueUrls...)
	}
	var queues []QueueInfo
	for i := page*size - size; i < page*size; i++ {
		if i >= len(queueURLs) {
			break
		}
		
		url := queueURLs[i]
		attrs, err := h.sqsClient.GetQueueAttributes(context.TODO(), &sqs.GetQueueAttributesInput{
			QueueUrl:       aws.String(url),
			AttributeNames: []types.QueueAttributeName{types.QueueAttributeNameApproximateNumberOfMessages},
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		count := 0
		if err == nil {
			count, _ = strconv.Atoi(attrs.Attributes[string(types.QueueAttributeNameApproximateNumberOfMessages)])
		}
		segments := strings.Split(url, "/")
		name := segments[len(segments)-1]
		queues = append(queues, QueueInfo{
			URL:          url,
			Name:         name,
			MessageCount: count,
		})
	}
	json.NewEncoder(w).Encode(queues)
}
