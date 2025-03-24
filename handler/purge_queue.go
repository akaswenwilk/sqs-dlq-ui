package handler

import (
	"net/http"

	"github.com/gorilla/mux"
)

func (h *Handler) PurgeQueue(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	queueName := vars["queueName"]
	err := h.repo.PurgeQueue(ctx, queueName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}
