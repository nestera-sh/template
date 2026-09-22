include .env
export

.PHONY: migrate migrate-down run generate test

migrate:
	go tool goose -dir migrations postgres "$(DATABASE_URL)" up

migrate-down:
	go tool goose -dir migrations postgres "$(DATABASE_URL)" down

run:
	go run ./cmd/trip-service

generate:
	mkdir -p internal/generated
	go tool oapi-codegen -generate types,chi-server -package api \
	  -o internal/generated/api.gen.go contracts/openapi/trip-service.openapi.yaml

test:
	go test -race ./...