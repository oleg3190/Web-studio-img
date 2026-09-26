# Web Studio IMG Backend

Go REST API foundation for Web Studio IMG.

## Run

From backend:

go run ./cmd/api

The API listens on APP_PORT (default 8080). Health endpoints:
- GET /healthz
- GET /readyz
- GET /api/v1/health

## Checks

go test ./...
go test -race ./...
go vet ./...
gofmt -w ./cmd ./internal

See docs/TECHNICAL_SPECIFICATION.md for the full architecture and environment contract.
