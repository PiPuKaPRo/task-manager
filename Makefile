.PHONY: run build test test-coverage migrate-up migrate-down deps clean

run:
	go run cmd/server/main.go

build:
	go build -o bin/task-manager cmd/server/main.go

test:
	go test -v -cover ./...

test-coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

migrate-up:
	migrate -path migrations -database "postgres://$(DB_USER):$(DB_PASSWORD)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=disable" up

migrate-down:
	migrate -path migrations -database "postgres://$(DB_USER):$(DB_PASSWORD)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=disable" down

deps:
	go mod download
	go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
	go install github.com/golang/mock/mockgen@latest

clean:
	rm -rf bin/ coverage.out coverage.html

mock-gen:
	mockgen -source=internal/service/task.go -destination=internal/service/mocks/mock_task.go -package=mocks