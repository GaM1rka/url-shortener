APP_NAME := url-shortener
IMAGE_NAME := $(APP_NAME):local
COMPOSE := docker compose -p deploy --env-file .env -f deploy/docker-compose.yml
SWAGGER_COMPOSE := docker compose -p url-shortener-swagger -f deploy/docker-compose.swagger.yml

.DEFAULT_GOAL := help

.PHONY: help env run run-memory build test test-race vet fmt check docker-build up down logs ps swagger-up swagger-down clean

help:
	@echo "Доступные команды:"
	@echo "  make run          - запустить приложение с настройками из .env"
	@echo "  make run-memory   - запустить приложение с in-memory хранилищем"
	@echo "  make up           - собрать и запустить приложение с PostgreSQL"
	@echo "  make down         - остановить Docker Compose"
	@echo "  make logs         - показать логи приложения"
	@echo "  make test         - запустить тесты"
	@echo "  make test-race    - запустить тесты с race detector"
	@echo "  make check        - форматирование, vet и тесты"
	@echo "  make swagger-up   - запустить Swagger UI"

env:
	@test -f .env || cp .env.example .env

run: env
	go run ./cmd/url-shortener

run-memory:
	STORAGE_TYPE=memory go run ./cmd/url-shortener

build:
	mkdir -p bin
	go build -trimpath -o bin/$(APP_NAME) ./cmd/url-shortener

test:
	go test ./...

test-race:
	go test -race ./...

vet:
	go vet ./...

fmt:
	go fmt ./...

check: fmt vet test

docker-build:
	docker build -f build/Dockerfile -t $(IMAGE_NAME) .

up: env
	$(COMPOSE) up --build -d

down: env
	$(COMPOSE) down

logs: env
	$(COMPOSE) logs -f app

ps: env
	$(COMPOSE) ps

swagger-up:
	$(SWAGGER_COMPOSE) up -d

swagger-down:
	$(SWAGGER_COMPOSE) down

clean:
	rm -rf bin
	go clean
