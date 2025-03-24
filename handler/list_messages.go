package handler

import (
	"encoding/json"
	"net/http"

	"github.com/akaswenwilk/sqs-dlq-ui/model"
	"github.com/gorilla/mux"
)

type ListMessagesResponse struct {
	Messages []model.Message `json:"messages"`
	Total    int             `json:"total"`
}

func (h *Handler) ListMessages(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	queueName := vars["queueName"]
	messages, total, err := h.repo.FetchMessages(ctx, queueName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(ListMessagesResponse{
		Messages: messages,
		Total:    total,
	})
}
