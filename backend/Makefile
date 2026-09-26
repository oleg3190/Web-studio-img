DB_URL ?= postgres://webstudio:webstudio@localhost:5432/webstudio?sslmode=disable
SQLC_VERSION ?= v1.29.0
.PHONY: sqlc test race fmt vet build migrate-up migrate-down
sqlc:
	sqlc generate
test:
	go test ./...
race:
	go test -race ./...
fmt:
	gofmt -w ./internal ./cmd
vet:
	go vet ./...
build:
	go build ./...
migrate-up:
	migrate -path migrations -database "$(DB_URL)" up
migrate-down:
	migrate -path migrations -database "$(DB_URL)" down 1
