# Makefile
.PHONY: build run test lint migrate seed docker-up docker-down

# Binary output
BINARY_NAME=server
BUILD_DIR=./bin

# Go build flags
GO_BUILD_FLAGS=-v

# Go test flags
GO_TEST_FLAGS=-v

build:
	go build $(GO_BUILD_FLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/server

run:
	go run ./cmd/server

test:
	go test $(GO_TEST_FLAGS) ./...

lint:
	golangci-lint run

migrate-up:
	./migrate -path ./migrations -database "postgres://postgres:postgres@localhost:5432/restaurant?sslmode=disable" up

migrate-down:
	./migrate -path ./migrations -database "postgres://postgres:postgres@localhost:5432/restaurant?sslmode=disable" down

migrate-create:
	@read -p "Enter migration name: " name; \
	./migrate create -ext sql -dir ./migrations -seq $$name

seed:
	go run ./cmd/tools/seeder

docker-up:
	docker-compose up -d

docker-down:
	docker-compose down

clean:
	rm -rf $(BUILD_DIR)