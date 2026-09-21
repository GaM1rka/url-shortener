# URL shortener

Я сделал небольшой сервис для создания коротких ссылок на Go. Для одного исходного URL всегда возвращается один и тот же короткий код длиной 10 символов.

Сервис умеет работать с двумя хранилищами:

- PostgreSQL через `pgxpool`
- память приложения с защитой от конкурентного доступа

Миграции PostgreSQL запускаются автоматически через Goose при старте приложения.

## Запуск через Docker

Нужны Docker и Docker Compose.

```bash
cp .env.example .env
make up
```

Сервис будет доступен по адресу `http://localhost:8081`. Остановить его можно командой `make down`, посмотреть логи - `make logs`.

## Запуск без Docker

Для запуска с хранилищем в памяти достаточно выполнить:

```bash
make run-memory
```

## API

Создать короткую ссылку:

```bash
curl -X POST http://localhost:8081/links \
  -H 'Content-Type: application/json' \
  -d '{"original_url":"https://example.com/article"}'
```

Получить исходный URL:

```bash
curl http://localhost:8081/{short_code} # нужно вставить short_code, который получили после вызова POST /links.
```

OpenAPI-описание находится в `api/openapi.yaml`. Swagger UI запускается командой `make swagger-up` и открывается по адресу `http://localhost:8080`.

## Проверка

```bash
make check
make test-race
```
