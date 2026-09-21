package http

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/GaM1rka/url-shortener/internal/domain"
	"github.com/GaM1rka/url-shortener/internal/repository"
	"github.com/GaM1rka/url-shortener/internal/repository/memory"
	"github.com/GaM1rka/url-shortener/internal/service"
)

const (
	testBaseURL  = "http://short.test"
	testCode     = "Ab3_dE9xY0"
	testOriginal = "https://example.com/articles/42?source=test"
)

type fixedGenerator struct {
	code string
	err  error
}

func (g fixedGenerator) Generate() (string, error) {
	return g.code, g.err
}

type repositoryStub struct {
	getByShort    func(context.Context, string) (domain.Link, error)
	getByOriginal func(context.Context, string) (domain.Link, error)
	create        func(context.Context, domain.Link) error
}

var _ repository.LinkRepository = (*repositoryStub)(nil)

func (r *repositoryStub) GetByShort(ctx context.Context, shortCode string) (domain.Link, error) {
	if r.getByShort == nil {
		return domain.Link{}, domain.ErrNotFound
	}
	return r.getByShort(ctx, shortCode)
}

func (r *repositoryStub) GetByOriginal(ctx context.Context, originalURL string) (domain.Link, error) {
	if r.getByOriginal == nil {
		return domain.Link{}, domain.ErrNotFound
	}
	return r.getByOriginal(ctx, originalURL)
}

func (r *repositoryStub) Create(ctx context.Context, link domain.Link) error {
	if r.create == nil {
		return nil
	}
	return r.create(ctx, link)
}

func TestHandlerCreateLink(t *testing.T) {
	t.Parallel()

	handler := newTestHandler(memory.New(), fixedGenerator{code: testCode})
	requestBody := `{"original_url":"` + testOriginal + `"}`

	created := performRequest(
		t,
		handler.Routes(),
		http.MethodPost,
		"/links",
		requestBody,
		"application/json; charset=utf-8",
	)
	if created.Code != http.StatusCreated {
		t.Fatalf("first POST status = %d, want %d; body = %s", created.Code, http.StatusCreated, created.Body.String())
	}

	want := ShortLinkResponse{
		ShortCode: testCode,
		ShortURL:  testBaseURL + "/" + testCode,
	}
	if got := decodeResponse[ShortLinkResponse](t, created); got != want {
		t.Fatalf("first POST response = %+v, want %+v", got, want)
	}
	if location := created.Header().Get("Location"); location != want.ShortURL {
		t.Fatalf("Location = %q, want %q", location, want.ShortURL)
	}
	assertJSONContentType(t, created)
	if origin := created.Header().Get("Access-Control-Allow-Origin"); origin != "*" {
		t.Fatalf("Access-Control-Allow-Origin = %q, want %q", origin, "*")
	}

	existing := performRequest(
		t,
		handler.Routes(),
		http.MethodPost,
		"/links",
		requestBody,
		"application/json",
	)
	if existing.Code != http.StatusOK {
		t.Fatalf("second POST status = %d, want %d; body = %s", existing.Code, http.StatusOK, existing.Body.String())
	}
	if got := decodeResponse[ShortLinkResponse](t, existing); got != want {
		t.Fatalf("second POST response = %+v, want %+v", got, want)
	}
	if location := existing.Header().Get("Location"); location != "" {
		t.Fatalf("existing link Location = %q, want empty", location)
	}
}

func TestHandlerCreateLinkValidation(t *testing.T) {
	t.Parallel()

	handler := newTestHandler(memory.New(), fixedGenerator{code: testCode})
	oversizedBody := `{"original_url":"https://example.com/` +
		strings.Repeat("a", int(maxCreateLinkRequestSize)) + `"}`

	tests := []struct {
		name        string
		body        string
		contentType string
		wantStatus  int
		wantCode    string
	}{
		{
			name:       "missing content type",
			body:       `{"original_url":"https://example.com"}`,
			wantStatus: http.StatusUnsupportedMediaType,
			wantCode:   "unsupported_media_type",
		},
		{
			name:        "wrong content type",
			body:        `{"original_url":"https://example.com"}`,
			contentType: "text/plain",
			wantStatus:  http.StatusUnsupportedMediaType,
			wantCode:    "unsupported_media_type",
		},
		{
			name:        "malformed JSON",
			body:        `{"original_url":`,
			contentType: "application/json",
			wantStatus:  http.StatusBadRequest,
			wantCode:    "invalid_request",
		},
		{
			name:        "unknown field",
			body:        `{"original_url":"https://example.com","unexpected":true}`,
			contentType: "application/json",
			wantStatus:  http.StatusBadRequest,
			wantCode:    "invalid_request",
		},
		{
			name:        "multiple JSON values",
			body:        `{"original_url":"https://example.com"} {}`,
			contentType: "application/json",
			wantStatus:  http.StatusBadRequest,
			wantCode:    "invalid_request",
		},
		{
			name:        "missing original URL",
			body:        `{}`,
			contentType: "application/json",
			wantStatus:  http.StatusBadRequest,
			wantCode:    "invalid_request",
		},
		{
			name:        "relative URL",
			body:        `{"original_url":"example.com/path"}`,
			contentType: "application/json",
			wantStatus:  http.StatusBadRequest,
			wantCode:    "invalid_request",
		},
		{
			name:        "unsupported URL scheme",
			body:        `{"original_url":"ftp://example.com/file"}`,
			contentType: "application/json",
			wantStatus:  http.StatusBadRequest,
			wantCode:    "invalid_request",
		},
		{
			name:        "oversized body",
			body:        oversizedBody,
			contentType: "application/json",
			wantStatus:  http.StatusBadRequest,
			wantCode:    "invalid_request",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response := performRequest(
				t,
				handler.Routes(),
				http.MethodPost,
				"/links",
				test.body,
				test.contentType,
			)
			assertErrorResponse(t, response, test.wantStatus, test.wantCode)
		})
	}
}

func TestHandlerCreateLinkInternalError(t *testing.T) {
	t.Parallel()

	repositoryError := errors.New("repository unavailable")
	repo := &repositoryStub{
		getByOriginal: func(context.Context, string) (domain.Link, error) {
			return domain.Link{}, repositoryError
		},
	}
	handler := newTestHandler(repo, fixedGenerator{code: testCode})

	response := performRequest(
		t,
		handler.Routes(),
		http.MethodPost,
		"/links",
		`{"original_url":"https://example.com"}`,
		"application/json",
	)
	assertErrorResponse(t, response, http.StatusInternalServerError, "internal_error")
	if strings.Contains(response.Body.String(), repositoryError.Error()) {
		t.Fatal("internal error details leaked to the response")
	}
}

func TestHandlerGetOriginalURL(t *testing.T) {
	t.Parallel()

	repo := memory.New()
	want := domain.Link{ShortCode: testCode, OriginalURL: testOriginal}
	if err := repo.Create(context.Background(), want); err != nil {
		t.Fatalf("repository Create() error = %v", err)
	}
	handler := newTestHandler(repo, fixedGenerator{code: testCode})

	response := performRequest(
		t,
		handler.Routes(),
		http.MethodGet,
		"/"+testCode,
		"",
		"",
	)
	if response.Code != http.StatusOK {
		t.Fatalf("GET status = %d, want %d; body = %s", response.Code, http.StatusOK, response.Body.String())
	}
	wantResponse := OriginalURLResponse{OriginalURL: testOriginal}
	if got := decodeResponse[OriginalURLResponse](t, response); got != wantResponse {
		t.Fatalf("GET response = %+v, want %+v", got, wantResponse)
	}
	assertJSONContentType(t, response)
}

func TestHandlerGetOriginalURLValidationAndNotFound(t *testing.T) {
	t.Parallel()

	handler := newTestHandler(memory.New(), fixedGenerator{code: testCode})
	tests := []struct {
		name       string
		shortCode  string
		wantStatus int
		wantCode   string
	}{
		{
			name:       "too short",
			shortCode:  "short",
			wantStatus: http.StatusBadRequest,
			wantCode:   "invalid_request",
		},
		{
			name:       "invalid character",
			shortCode:  "123456789-",
			wantStatus: http.StatusBadRequest,
			wantCode:   "invalid_request",
		},
		{
			name:       "not found",
			shortCode:  "1234567890",
			wantStatus: http.StatusNotFound,
			wantCode:   "link_not_found",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response := performRequest(
				t,
				handler.Routes(),
				http.MethodGet,
				"/"+test.shortCode,
				"",
				"",
			)
			assertErrorResponse(t, response, test.wantStatus, test.wantCode)
		})
	}
}

func TestHandlerGetOriginalURLInternalError(t *testing.T) {
	t.Parallel()

	repositoryError := errors.New("repository unavailable")
	repo := &repositoryStub{
		getByShort: func(context.Context, string) (domain.Link, error) {
			return domain.Link{}, repositoryError
		},
	}
	handler := newTestHandler(repo, fixedGenerator{code: testCode})

	response := performRequest(
		t,
		handler.Routes(),
		http.MethodGet,
		"/"+testCode,
		"",
		"",
	)
	assertErrorResponse(t, response, http.StatusInternalServerError, "internal_error")
	if strings.Contains(response.Body.String(), repositoryError.Error()) {
		t.Fatal("internal error details leaked to the response")
	}
}

func TestHandlerWithoutServiceReturnsInternalError(t *testing.T) {
	t.Parallel()

	handler := NewHandler(nil, testBaseURL, discardLogger())
	tests := []struct {
		name        string
		method      string
		target      string
		body        string
		contentType string
	}{
		{
			name:        "create",
			method:      http.MethodPost,
			target:      "/links",
			body:        `{"original_url":"https://example.com"}`,
			contentType: "application/json",
		},
		{
			name:   "get",
			method: http.MethodGet,
			target: "/" + testCode,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response := performRequest(
				t,
				handler.Routes(),
				test.method,
				test.target,
				test.body,
				test.contentType,
			)
			assertErrorResponse(t, response, http.StatusInternalServerError, "internal_error")
		})
	}
}

func TestHandlerCORSPreflight(t *testing.T) {
	t.Parallel()

	handler := newTestHandler(memory.New(), fixedGenerator{code: testCode})
	response := performRequest(
		t,
		handler.Routes(),
		http.MethodOptions,
		"/links",
		"",
		"",
	)

	if response.Code != http.StatusNoContent {
		t.Fatalf("OPTIONS status = %d, want %d", response.Code, http.StatusNoContent)
	}
	if origin := response.Header().Get("Access-Control-Allow-Origin"); origin != "*" {
		t.Fatalf("Access-Control-Allow-Origin = %q, want %q", origin, "*")
	}
	if methods := response.Header().Get("Access-Control-Allow-Methods"); methods != "GET, POST, OPTIONS" {
		t.Fatalf("Access-Control-Allow-Methods = %q, want %q", methods, "GET, POST, OPTIONS")
	}
}

func newTestHandler(repo repository.LinkRepository, generator service.CodeGenerator) *Handler {
	shortener := service.NewShortenerService(repo, generator)
	return NewHandler(shortener, testBaseURL+"/", discardLogger())
}

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func performRequest(
	t *testing.T,
	handler http.Handler,
	method string,
	target string,
	body string,
	contentType string,
) *httptest.ResponseRecorder {
	t.Helper()

	request := httptest.NewRequest(method, target, strings.NewReader(body))
	if contentType != "" {
		request.Header.Set("Content-Type", contentType)
	}

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}

func decodeResponse[T any](t *testing.T, response *httptest.ResponseRecorder) T {
	t.Helper()

	var decoded T
	decoder := json.NewDecoder(response.Body)
	if err := decoder.Decode(&decoded); err != nil {
		t.Fatalf("decode response: %v; body = %s", err, response.Body.String())
	}
	return decoded
}

func assertErrorResponse(
	t *testing.T,
	response *httptest.ResponseRecorder,
	wantStatus int,
	wantCode string,
) {
	t.Helper()

	if response.Code != wantStatus {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, wantStatus, response.Body.String())
	}
	decoded := decodeResponse[ErrorResponse](t, response)
	if decoded.Code != wantCode {
		t.Fatalf("error code = %q, want %q", decoded.Code, wantCode)
	}
	if decoded.Message == "" {
		t.Fatal("error message is empty")
	}
	assertJSONContentType(t, response)
}

func assertJSONContentType(t *testing.T, response *httptest.ResponseRecorder) {
	t.Helper()

	if got := response.Header().Get("Content-Type"); got != "application/json; charset=utf-8" {
		t.Fatalf("Content-Type = %q, want application/json; charset=utf-8", got)
	}
}
