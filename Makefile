.PHONY: build test lint run clean listening-test test-integration test-coverage \
       docker docker-run release-snapshot help

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -s -w -X main.version=$(VERSION)

## Build

build:
	go build -ldflags="$(LDFLAGS)" -o bin/llmidi-gen ./cmd/llmidi-gen/

build-all: ## Cross-compile for all platforms
	GOOS=linux   GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o bin/llmidi-gen-linux-amd64 ./cmd/llmidi-gen/
	GOOS=linux   GOARCH=arm64 CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o bin/llmidi-gen-linux-arm64 ./cmd/llmidi-gen/
	GOOS=darwin  GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o bin/llmidi-gen-darwin-amd64 ./cmd/llmidi-gen/
	GOOS=darwin  GOARCH=arm64 CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o bin/llmidi-gen-darwin-arm64 ./cmd/llmidi-gen/
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o bin/llmidi-gen-windows-amd64.exe ./cmd/llmidi-gen/
	GOOS=windows GOARCH=arm64 CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o bin/llmidi-gen-windows-arm64.exe ./cmd/llmidi-gen/

## Test

test:
	go test ./... -v -count=1

test-race:
	go test ./... -race -count=1

test-integration:
	go test ./... -v -tags=integration -count=1

test-coverage:
	go test ./... -coverprofile=coverage.out -covermode=atomic
	go tool cover -func=coverage.out

test-coverage-html: test-coverage
	go tool cover -html=coverage.out -o coverage.html

## Lint

lint:
	golangci-lint run ./...

vet:
	go vet ./...

fmt:
	gofmt -w .
	goimports -w .

## Run

run:
	go run ./cmd/llmidi-gen/ --bpm 122 --key Am

run-offline:
	go run ./cmd/llmidi-gen/ --bpm 122 --key Am --no-llm

run-claude:
	go run ./cmd/llmidi-gen/ --bpm 122 --key Am --provider claude

run-ollama:
	go run ./cmd/llmidi-gen/ --bpm 126 --key Em --provider ollama --model qwen2.5:7b

## Listening test

listening-test:
	@mkdir -p output/listening
	@echo "Generating patterns for listening test..."
	@for key in Am Em Fm Dm; do \
		for bpm in 122 126 128; do \
			go run ./cmd/llmidi-gen/ --bpm $$bpm --key $$key --no-llm --output output/listening; \
		done; \
	done
	@for key in D C G; do \
		for bpm in 120 124 128; do \
			go run ./cmd/llmidi-gen/ --bpm $$bpm --key $$key --no-llm --output output/listening; \
		done; \
	done
	@echo "Done. Load output/listening/ files in DAW for A/B comparison."

## Docker

docker:
	docker build -t llmidi-gen:$(VERSION) .
	docker tag llmidi-gen:$(VERSION) llmidi-gen:latest

docker-run:
	docker run --rm -v $(PWD)/output:/app/output llmidi-gen:latest --bpm 122 --key Am --no-llm

docker-compose-up:
	docker compose up llmidi-gen

## Clean

clean:
	rm -rf bin/ output/ debug/ coverage.out coverage.html

## Help

help:
	@echo "LLMIDI-Gen — AI-powered MIDI generator"
	@echo ""
	@echo "Build:"
	@echo "  make build          Build binary"
	@echo "  make build-all      Cross-compile all platforms"
	@echo "  make docker         Build Docker image"
	@echo ""
	@echo "Test:"
	@echo "  make test           Run unit tests"
	@echo "  make test-race      Run tests with race detector"
	@echo "  make test-coverage  Generate coverage report"
	@echo "  make lint           Run golangci-lint"
	@echo ""
	@echo "Run:"
	@echo "  make run            Run with Claude (Am, 122 BPM)"
	@echo "  make run-offline    Run without LLM"
	@echo "  make run-ollama     Run with Ollama"
	@echo "  make listening-test Generate patterns for DAW comparison"
	@echo ""
	@echo "Docker:"
	@echo "  make docker-run     Run in container (offline mode)"
