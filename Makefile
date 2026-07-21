APP_NAME := sharetrip
CMD_PATH := ./cmd/sharetrip
BIN_DIR := bin
BIN_PATH := $(BIN_DIR)/$(APP_NAME)
TOOLS_BIN := $(BIN_DIR)/tools
COMPOSE_FILE := deploy/docker-compose.yml
MIGRATIONS_DIR := migrations
DATABASE_URL ?= postgres://postgres:password@localhost:6544/sharetrip?sslmode=disable
E2E_BASE_URL ?= http://localhost:8080
GOOSE := $(TOOLS_BIN)/goose
GOOSE_VERSION := v3.26.0
GOOSE_DRIVER := postgres
GOLANGCI_LINT := $(TOOLS_BIN)/golangci-lint
GOLANGCI_LINT_VERSION := v2.11.3

.PHONY: deps fmt lint test coverage build run up down migrate-up migrate-down migrate-status e2e check

deps: $(GOOSE) $(GOLANGCI_LINT)
	go mod tidy
	go mod download

fmt:
	go fmt ./...

lint: $(GOLANGCI_LINT)
	$(GOLANGCI_LINT) run ./...

test:
	go test ./...

coverage:
	go test ./... -coverprofile=coverage.out
	go tool cover -func=coverage.out

build:
	mkdir -p $(BIN_DIR)
	go build -o $(BIN_PATH) $(CMD_PATH)

run:
	go run $(CMD_PATH)

up:
	docker compose -f $(COMPOSE_FILE) up -d

down:
	docker compose -f $(COMPOSE_FILE) down

migrate-up: $(GOOSE)
	$(GOOSE) -dir $(MIGRATIONS_DIR) $(GOOSE_DRIVER) "$(DATABASE_URL)" up

migrate-down: $(GOOSE)
	$(GOOSE) -dir $(MIGRATIONS_DIR) $(GOOSE_DRIVER) "$(DATABASE_URL)" down

migrate-status: $(GOOSE)
	$(GOOSE) -dir $(MIGRATIONS_DIR) $(GOOSE_DRIVER) "$(DATABASE_URL)" status

e2e:
	curl -fsS $(E2E_BASE_URL)/ready

check: deps fmt lint test build

$(GOOSE):
	mkdir -p $(TOOLS_BIN)
	GOBIN=$(abspath $(TOOLS_BIN)) go install github.com/pressly/goose/v3/cmd/goose@$(GOOSE_VERSION)

$(GOLANGCI_LINT):
	mkdir -p $(TOOLS_BIN)
	GOBIN=$(abspath $(TOOLS_BIN)) go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)
