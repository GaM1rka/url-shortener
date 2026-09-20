package http

import (
	"net/http"

	"github.com/GaM1rka/url-shortener"
)

type Handler struct {
	service *service.ShortenerService
}

func NewHandler(service *service.ShortenerService) *Handler {
	return &Handler{
		service: service,
	}
}

func (h *Handler) CreateLink(w http.ResponseWriter, r *http.Request){}

func (h *Handler) GetOriginalUrl(w http.ResponseWriter, r *http.Request){}
