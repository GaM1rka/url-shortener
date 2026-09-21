package domain

import "errors"

var (
	ErrNotFound          = errors.New("link not found")
	ErrShortCodeExists   = errors.New("short code already exists")
	ErrOriginalURLExists = errors.New("original URL already exists")
	ErrInvalidURL        = errors.New("invalid original URL")
	ErrInvalidShortCode  = errors.New("invalid short code")
)
