package http

import (
	"log/slog"
	"net/http"

	"github.com/GaM1rka/url-shortener/internal/service"
)

type Handler struct {
	service *service.ShortenerService
	baseURL string
	logger  *slog.Logger
}

func NewHandler(service *service.ShortenerService, baseURL string, logger *slog.Logger) *Handler {
	return &Handler{
		service: service,
		baseURL: baseURL,
		logger:  logger,
	}
}

func (h *Handler) CreateLink(w http.ResponseWriter, r *http.Request){}

func (h *Handler) GetOriginalURL(w http.ResponseWriter, r *http.Request){}
