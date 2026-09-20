package http

import (
	"net/http"
)

func (h *Handler) Routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /links", enableCORS(loggingMiddleware(h.CreateLink)))
	mux.HandleFunc("Get /{short_code}", enableCORS(loggingMiddleware(h.GetOriginalUrl)))

	return mux
}