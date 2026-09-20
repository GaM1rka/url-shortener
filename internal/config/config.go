package config

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

const (
	StorageMemory   = "memory"
	StoragePostgres = "postgres"
)

type Config struct {
	HTTP        HTTPConfig
	StorageType string
	PostgresDSN string
}

type HTTPConfig struct {
	Address         string
	BaseURL         string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	ShutdownTimeout time.Duration
}

func MustLoad() (*Config, error) {
	values, err := godotenv.Read(".env")
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, errors.New("config: cannot read or parse .env")
	}

	env := func(key, fallback string) string {
		if value, ok := os.LookupEnv(key); ok {
			return value
		}
		if value, ok := values[key]; ok {
			return value
		}
		return fallback
	}

	cfg := &Config{
		HTTP: HTTPConfig{
			Address: env("HTTP_ADDRESS", ":8081"),
			BaseURL: env("HTTP_BASE_URL", "http://localhost:8081"),
		},
		StorageType: env("STORAGE_TYPE", StorageMemory),
		PostgresDSN: env("POSTGRES_DSN", ""),
	}

	timeouts := []struct {
		key      string
		fallback string
		target   *time.Duration
	}{
		{"HTTP_READ_TIMEOUT", "5s", &cfg.HTTP.ReadTimeout},
		{"HTTP_WRITE_TIMEOUT", "10s", &cfg.HTTP.WriteTimeout},
		{"HTTP_IDLE_TIMEOUT", "60s", &cfg.HTTP.IdleTimeout},
		{"HTTP_SHUTDOWN_TIMEOUT", "10s", &cfg.HTTP.ShutdownTimeout},
	}
	for _, timeout := range timeouts {
		value, err := time.ParseDuration(env(timeout.key, timeout.fallback))
		if err != nil || value <= 0 {
			return nil, fmt.Errorf("config: %s must be a positive duration, for example 5s or 1m", timeout.key)
		}
		*timeout.target = value
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}
	cfg.HTTP.BaseURL = strings.TrimSuffix(cfg.HTTP.BaseURL, "/")
	return cfg, nil
}

func (cfg *Config) validate() error {
	_, port, err := net.SplitHostPort(cfg.HTTP.Address)
	if err != nil || !validPort(port) {
		return errors.New("config: HTTP_ADDRESS must have the form host:port or :port, with port between 1 and 65535")
	}

	baseURL, err := url.Parse(cfg.HTTP.BaseURL)
	if err != nil || (baseURL.Scheme != "http" && baseURL.Scheme != "https") ||
		baseURL.Hostname() == "" || baseURL.User != nil ||
		(baseURL.EscapedPath() != "" && baseURL.EscapedPath() != "/") ||
		baseURL.RawQuery != "" || baseURL.ForceQuery || strings.Contains(cfg.HTTP.BaseURL, "#") {
		return errors.New("config: HTTP_BASE_URL must be an absolute HTTP/HTTPS URL without credentials, path, query or fragment")
	}
	if port := baseURL.Port(); port != "" && !validPort(port) {
		return errors.New("config: HTTP_BASE_URL port must be between 1 and 65535")
	}

	switch cfg.StorageType {
	case StorageMemory:
	case StoragePostgres:
		if cfg.PostgresDSN == "" {
			return errors.New("config: POSTGRES_DSN is required when STORAGE_TYPE=postgres")
		}
		dsn, err := url.Parse(cfg.PostgresDSN)
		if err != nil || (dsn.Scheme != "postgres" && dsn.Scheme != "postgresql") ||
			dsn.Hostname() == "" || strings.Trim(dsn.Path, "/") == "" || dsn.Fragment != "" {
			return errors.New("config: POSTGRES_DSN must be a postgres:// or postgresql:// URL with a host and database name")
		}
		if port := dsn.Port(); port != "" && !validPort(port) {
			return errors.New("config: POSTGRES_DSN port must be between 1 and 65535")
		}
		if _, err := url.ParseQuery(dsn.RawQuery); err != nil {
			return errors.New("config: POSTGRES_DSN contains invalid query parameters")
		}
	default:
		return errors.New("config: STORAGE_TYPE must be memory or postgres")
	}
	return nil
}

func validPort(value string) bool {
	port, err := strconv.ParseUint(value, 10, 16)
	return err == nil && port > 0
}
