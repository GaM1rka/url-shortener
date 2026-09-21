package http

import (
	"net/http"
)

func (h *Handler) Routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /links", h.CreateLink)
	mux.HandleFunc("GET /{short_code}", h.GetOriginalURL)

	return h.loggingMiddleware(enableCORS(mux))
}
