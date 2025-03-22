package handler

import "net/http"

func (h *Handler) RetryAllMessages(w http.ResponseWriter, r *http.Request) {
	// Not implemented: SQS doesn't provide a bulk retry. Custom implementation required.
	w.WriteHeader(http.StatusNotImplemented)
}
