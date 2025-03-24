package handler

import (
	"net/http"
	"regexp"

	"github.com/akaswenwilk/sqs-dlq-ui/model"
	"github.com/gorilla/mux"
	"github.com/samber/lo"
)

var dlqNameRegex = regexp.MustCompile(".*-dlq$")

func (h *Handler) RetryMessage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	queueName := vars["queueName"]
	messageID := vars["messageID"]
	redriveQueues, err := h.repo.ListDeadLetterSourceQueues(ctx, queueName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if len(redriveQueues) == 0 {
		http.Error(w, "No redrive queues found", http.StatusNotFound)
		return
	}
	allMessages, _, err := h.repo.FetchMessages(ctx, queueName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	retryMessage, found := lo.Find(allMessages, func(msg model.Message) bool {
		return msg.MessageId == messageID
	})
	if !found {
		http.Error(w, "Message not found", http.StatusNotFound)
		return
	}

	for _, redriveQueue := range redriveQueues {
		err = h.repo.PublishMessage(ctx, redriveQueue.URL, retryMessage)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	
	if err := h.repo.DeleteMessage(ctx, queueName, retryMessage.ReceiptHandle); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
