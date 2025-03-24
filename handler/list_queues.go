package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/akaswenwilk/sqs-dlq-ui/model"
)

type QueueInfoResponse struct {
	Queues []model.QueueInfo `json:"queues"`
	Total  int               `json:"total"`
}

func (h *Handler) ListQueues(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	size, _ := strconv.Atoi(r.URL.Query().Get("size"))
	if size < 1 {
		size = 10
	}
	search := r.URL.Query().Get("search")
	queues, total, err := h.repo.ListQueues(ctx, page, size, search)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	response := QueueInfoResponse{Queues: queues, Total: total}
	json.NewEncoder(w).Encode(response)
}
