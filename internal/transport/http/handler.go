package http

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"net/url"
	"strings"

	"github.com/GaM1rka/url-shortener/internal/service"
)

const maxCreateLinkRequestSize int64 = 1 << 20

type Handler struct {
	service *service.ShortenerService
	baseURL string
	logger  *slog.Logger
}

func NewHandler(service *service.ShortenerService, baseURL string, logger *slog.Logger) *Handler {
	return &Handler{
		service: service,
		baseURL: strings.TrimSuffix(baseURL, "/"),
		logger:  logger,
	}
}

func (h *Handler) CreateLink(w http.ResponseWriter, r *http.Request) {
	if !isJSONContentType(r.Header.Get("Content-Type")) {
		writeJSON(w, http.StatusUnsupportedMediaType, ErrorResponse{
			Code:    "unsupported_media_type",
			Message: "Ожидается Content-Type application/json.",
		})
		return
	}

	if r.Body == nil {
		writeInvalidJSON(w)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxCreateLinkRequestSize)
	defer r.Body.Close()

	var request CreateLinkRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&request); err != nil {
		writeInvalidJSON(w)
		return
	}

	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		writeInvalidJSON(w)
		return
	}

	if !isValidOriginalURL(request.OriginalURL) {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{
			Code:    "invalid_request",
			Message: "Поле original_url должно содержать абсолютный HTTP/HTTPS URL.",
		})
		return
	}

	if h.service == nil {
		h.writeInternalError(w, errors.New("shortener service is nil"))
		return
	}

	link, created, err := h.service.Create(r.Context(), request.OriginalURL)
	if err != nil {
		h.writeInternalError(w, err)
		return
	}

	shortURL := h.baseURL + "/" + link.ShortCode
	response := ShortLinkResponse{
		ShortCode: link.ShortCode,
		ShortURL:  shortURL,
	}

	if created {
		w.Header().Set("Location", shortURL)
		writeJSON(w, http.StatusCreated, response)
		return
	}

	writeJSON(w, http.StatusOK, response)
}

func (h *Handler) GetOriginalURL(w http.ResponseWriter, r *http.Request) {
	shortCode := r.PathValue("short_code")
	if !isValidShortCode(shortCode) {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{
			Code:    "invalid_request",
			Message: "Код должен содержать ровно 10 символов из набора A-Z, a-z, 0-9, _.",
		})
		return
	}

	if h.service == nil {
		h.writeInternalError(w, errors.New("shortener service is nil"))
		return
	}

	link, err := h.service.GetByShort(r.Context(), shortCode)
	if err != nil {
		if isNotFoundError(err) {
			writeJSON(w, http.StatusNotFound, ErrorResponse{
				Code:    "link_not_found",
				Message: "Сокращённая ссылка не найдена.",
			})
			return
		}

		h.writeInternalError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, OriginalURLResponse{
		OriginalURL: link.OriginalURL,
	})
}

func isJSONContentType(value string) bool {
	mediaType, _, err := mime.ParseMediaType(value)
	return err == nil && mediaType == "application/json"
}

func isValidOriginalURL(value string) bool {
	parsed, err := url.Parse(value)
	if err != nil {
		return false
	}

	scheme := strings.ToLower(parsed.Scheme)
	return (scheme == "http" || scheme == "https") &&
		parsed.IsAbs() && parsed.Host != "" && parsed.Hostname() != ""
}

func isValidShortCode(value string) bool {
	if len(value) != 10 {
		return false
	}

	for i := 0; i < len(value); i++ {
		char := value[i]
		if (char < 'a' || char > 'z') &&
			(char < 'A' || char > 'Z') &&
			(char < '0' || char > '9') &&
			char != '_' {
			return false
		}
	}

	return true
}

func isNotFoundError(err error) bool {
	type notFoundMarker interface {
		NotFound() bool
	}
	type errorCoder interface {
		Code() string
	}
	type statusCoder interface {
		StatusCode() int
	}

	var marker notFoundMarker
	if errors.As(err, &marker) && marker.NotFound() {
		return true
	}

	var coder errorCoder
	if errors.As(err, &coder) {
		code := coder.Code()
		if code == "not_found" || code == "link_not_found" {
			return true
		}
	}

	var status statusCoder
	if errors.As(err, &status) && status.StatusCode() == http.StatusNotFound {
		return true
	}

	for current := err; current != nil; current = errors.Unwrap(current) {
		switch strings.ToLower(strings.TrimSpace(current.Error())) {
		case "not found", "link not found", "link_not_found":
			return true
		}
	}

	return false
}

func writeInvalidJSON(w http.ResponseWriter) {
	writeJSON(w, http.StatusBadRequest, ErrorResponse{
		Code:    "invalid_request",
		Message: "Тело запроса должно содержать один корректный JSON-объект.",
	})
}

func (h *Handler) writeInternalError(w http.ResponseWriter, err error) {
	if h.logger != nil {
		h.logger.Error("HTTP handler error", "error", err)
	}

	writeJSON(w, http.StatusInternalServerError, ErrorResponse{
		Code:    "internal_error",
		Message: "Внутренняя ошибка сервиса.",
	})
}

func writeJSON(w http.ResponseWriter, status int, response any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(response)
}
