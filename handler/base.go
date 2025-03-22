package handler

import (
	"context"
	"net/http"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/gorilla/mux"
)

type Handler struct {
	sqsClient *sqs.Client
	r         *mux.Router
}

func New(r *mux.Router, sqsClient *sqs.Client) *Handler {
	return &Handler{
		sqsClient: sqsClient,
		r:         r,
	}
}

func (h *Handler) getQueueURLByName(ctx context.Context, name string) (string, error) {
	result, err := h.sqsClient.ListQueues(ctx, &sqs.ListQueuesInput{})
	if err != nil {
		return "", err
	}
	for _, url := range result.QueueUrls {
		segments := strings.Split(url, "/")
		if segments[len(segments)-1] == name {
			return url, nil
		}
	}
	return "", http.ErrNoLocation
}
