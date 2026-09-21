package http

import (
	"net/http"
)

func (h *Handler) Routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /links", enableCORS(h.loggingMiddleware(h.CreateLink)))
	mux.HandleFunc("GET /{short_code}", enableCORS(h.loggingMiddleware(h.GetOriginalURL)))

	return mux
}
