package handler

import (
	"net/http"

	"github.com/akaswenwilk/sqs-dlq-ui/model"
	"github.com/gorilla/mux"
	"github.com/samber/lo"
)

func (h *Handler) DeleteMessage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	queueName := vars["queueName"]
	messageID := vars["messageID"]
	messages, _, err := h.repo.FetchMessages(ctx, queueName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	messageToDelete, found := lo.Find(messages, func(msg model.Message) bool {
		return msg.MessageId == messageID
	})
	if !found {
		http.Error(w, "Message not found", http.StatusNotFound)
		return
	}
	err = h.repo.DeleteMessage(ctx, queueName, messageToDelete.ReceiptHandle)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}
