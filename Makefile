include .env
export

.PHONY: migrate migrate-down run generate test

migrate:
	go tool goose -dir migrations postgres "$(DATABASE_URL)" up

migrate-down:
	go tool goose -dir migrations postgres "$(DATABASE_URL)" down


build:
	go build -o bin/trip-service ./cmd/trip-service

run:
	-go run ./cmd/trip-service

generate:
	mkdir -p internal/generated
	go tool oapi-codegen -generate types,chi-server -package api \
	  -o internal/generated/api.gen.go contracts/openapi/trip-service.openapi.yaml
	go generate ./...
lint:
	golangci-lint run ./...

test:
	go test -race -cover ./...

test-integration:
	go test -tags=integration -v -race ./internal/repository/...


clean:
	rm -rf bin/
	rm -f internal/generated/api.gen.go
	find . -name "mocks_test.go" -delete