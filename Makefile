.PHONY: build run test test-full lint proto proto-clean docker-build docker-run up down clean

APP_NAME=anti-bruteforce
BIN_DIR=bin
PROTO_DIR=api/proto
THIRD_PARTY=third_party
GO_OUT=.
GOPATH_BIN := $(shell go env GOPATH)/bin

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
	@echo "Generating proto..."
	protoc \
		-I $(PROTO_DIR) \
		-I $(THIRD_PARTY) \
		--plugin=protoc-gen-go=$(GOPATH_BIN)/protoc-gen-go \
		--plugin=protoc-gen-go-grpc=$(GOPATH_BIN)/protoc-gen-go-grpc \
		--plugin=protoc-gen-grpc-gateway=$(GOPATH_BIN)/protoc-gen-grpc-gateway \
		--go_out=. --go_opt=module=github.com/boba180799/anti-bruteforce \
		--go-grpc_out=. --go-grpc_opt=module=github.com/boba180799/anti-bruteforce \
		--grpc-gateway_out=. --grpc-gateway_opt=module=github.com/boba180799/anti-bruteforce \
		$(PROTO_DIR)/v1/*.proto

up:
	docker compose -f deploy/docker-compose.yml up -d

down:
	docker compose -f deploy/docker-compose.yml down

clean:
	rm -rf $(BIN_DIR)

docker-build:
	docker build -t abf .

docker-run:
	docker run --rm abf

proto-clean:
	@echo "Cleaning generated proto files..."
	find api/proto -name '*.pb.go' -delete
	find api/proto -name '*.pb.gw.go' -delete