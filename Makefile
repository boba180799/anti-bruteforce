.PHONY: build run test lint proto docker-build up down clean

APP_NAME=anti-bruteforce
BIN_DIR=bin

build:
	@echo "Building..."
	@mkdir -p $(BIN_DIR)
	go build -o $(BIN_DIR)/$(APP_NAME) ./cmd/antibruteforce
	go build -o $(BIN_DIR)/cli ./cmd/cli

run:
	docker compose -f deploy/docker-compose.yml up --build

test:
	go test -race -count 1 ./...

test-full:
	go test -race -count 100 ./...

lint:
	golangci-lint run ./...

proto:
	protoc --proto_path=api/proto \
		--go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		--grpc-gateway_out=. --grpc-gateway_opt=paths=source_relative \
		api/proto/v1/*.proto

up:
	docker compose -f deploy/docker-compose.yml up -d

down:
	docker compose -f deploy/docker-compose.yml down

clean:
	rm -rf $(BIN_DIR)