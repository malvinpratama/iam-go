GOBIN := $(shell go env GOPATH)/bin
export PATH := $(GOBIN):$(PATH)

.PHONY: tools proto sqlc build test vet run up down logs smoke

## Install codegen tools (buf, protoc plugins, sqlc)
tools:
	go install github.com/bufbuild/buf/cmd/buf@latest
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest

## Generate gRPC stubs from proto
proto:
	buf generate

## Generate sqlc db code for each service
sqlc:
	cd services/auth && sqlc generate
	cd services/user && sqlc generate

build:
	go build ./...

vet:
	go vet ./...

test:
	go test ./...

## Run the full stack via docker-compose
up:
	cd deploy && [ -f .env ] || cp .env.example .env
	cd deploy && docker compose up --build -d

down:
	cd deploy && docker compose down -v

logs:
	cd deploy && docker compose logs -f

## End-to-end smoke test against the running gateway
smoke:
	./scripts/smoke.sh http://localhost:8080
