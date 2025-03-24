package handler

import (
	"net/http"

	"github.com/gorilla/mux"
)

func (h *Handler) RetryAllMessages(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	queueName := vars["queueName"]
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
	for _, redriveQueue := range redriveQueues {
		for _, msg := range allMessages {
			err = h.repo.PublishMessage(ctx, redriveQueue.URL, msg)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
	}
	for _, msg := range allMessages {
		if err := h.repo.DeleteMessage(ctx, queueName, msg.ReceiptHandle); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	// Not implemented: SQS doesn't provide a bulk retry. Custom implementation required.
	w.WriteHeader(http.StatusNotImplemented)
}
