.PHONY: dev build test lint clean migrate-up migrate-down docker-build docker-up docker-down

dev:
	go run ./cmd/api

build:
	CGO_ENABLED=0 go build -ldflags="-s -w" -o bin/api ./cmd/api

test:
	go test ./...

lint:
	go vet ./...

migrate-up:
	psql "$(DATABASE_URL)" -f migrations/000001_init_schema.up.sql

migrate-down:
	psql "$(DATABASE_URL)" -f migrations/000001_init_schema.down.sql

clean:
	rm -rf bin/

docker-build:
	docker build -t tanok-web-api .

docker-up:
	docker compose up -d

docker-down:
	docker compose down
