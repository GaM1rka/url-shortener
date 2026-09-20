package http

type CreateLinkRequest struct {
	OriginalURL string `json:"original_url"`
}

type ShortLinkResponse struct {
	ShortCode string `json:"short_code"`
	ShortURL  string `json:"short_url"`
}

type OriginalURLResponse struct {
	OriginalURL string `json:"original_url"`
}

type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}