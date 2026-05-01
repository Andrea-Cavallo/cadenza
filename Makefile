.PHONY: build test lint run clean listening-test test-integration test-coverage \
       docker docker-run release-snapshot release sonar vuln ci help install-tools

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -s -w -X main.version=$(VERSION)

## Build

build:
	go build -ldflags="$(LDFLAGS)" -o bin/cadenza ./cmd/cadenza/

build-all: ## Cross-compile for all platforms
	GOOS=linux   GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o bin/cadenza-linux-amd64 ./cmd/cadenza/
	GOOS=linux   GOARCH=arm64 CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o bin/cadenza-linux-arm64 ./cmd/cadenza/
	GOOS=darwin  GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o bin/cadenza-darwin-amd64 ./cmd/cadenza/
	GOOS=darwin  GOARCH=arm64 CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o bin/cadenza-darwin-arm64 ./cmd/cadenza/
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o bin/cadenza-windows-amd64.exe ./cmd/cadenza/
	GOOS=windows GOARCH=arm64 CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o bin/cadenza-windows-arm64.exe ./cmd/cadenza/

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

## Lint & Quality

lint:
	golangci-lint run ./...

vet:
	go vet ./...

fmt:
	gofmt -w .
	goimports -w .

vuln: ## Vulnerability scan via govulncheck
	@which govulncheck > /dev/null 2>&1 || go install golang.org/x/vuln/cmd/govulncheck@latest
	govulncheck ./...

sonar: test-coverage ## Run SonarScanner locally (requires sonar-scanner on PATH + SONAR_TOKEN env var)
	@which sonar-scanner > /dev/null 2>&1 || \
		(echo "sonar-scanner not found: https://docs.sonarcloud.io/advanced-setup/ci-based-analysis/sonarscanner-cli/" && exit 1)
	@test -n "$$SONAR_TOKEN" || (echo "Error: SONAR_TOKEN env var is not set" && exit 1)
	sonar-scanner \
		-Dsonar.projectKey=Andrea-Cavallo_cadenza \
		-Dsonar.organization=andrea-cavallo \
		-Dsonar.sources=. \
		-Dsonar.go.coverage.reportPaths=coverage.out \
		-Dsonar.token="$$SONAR_TOKEN"

ci: build fmt vet lint vuln test-coverage ## Full local CI pipeline: build → fmt → vet → lint → vuln → coverage
	GOOS=linux go build ./...
	@echo "CI pipeline complete."

install-tools: ## Install development tools (golangci-lint, govulncheck, goimports)
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.64.8
	go install golang.org/x/vuln/cmd/govulncheck@latest
	go install golang.org/x/tools/cmd/goimports@latest

## Run

run:
	go run ./cmd/cadenza/ --bpm 122 --key Am

run-offline:
	go run ./cmd/cadenza/ --bpm 122 --key Am --no-llm

run-claude:
	go run ./cmd/cadenza/ --bpm 122 --key Am --provider claude

run-ollama:
	go run ./cmd/cadenza/ --bpm 126 --key Em --provider ollama --model qwen2.5:7b

## Listening test

listening-test:
	@mkdir -p output/listening
	@echo "Generating patterns for listening test..."
	@for key in Am Em Fm Dm; do \
		for bpm in 122 126 128; do \
			go run ./cmd/cadenza/ --bpm $$bpm --key $$key --no-llm --output output/listening; \
		done; \
	done
	@for key in D C G; do \
		for bpm in 120 124 128; do \
			go run ./cmd/cadenza/ --bpm $$bpm --key $$key --no-llm --output output/listening; \
		done; \
	done
	@echo "Done. Load output/listening/ files in DAW for A/B comparison."

## Release

release: build-all ## Package all platform binaries as zip archives in dist/
	@mkdir -p dist
	@zip -j dist/cadenza-$(VERSION)-linux-amd64.zip   bin/cadenza-linux-amd64
	@zip -j dist/cadenza-$(VERSION)-linux-arm64.zip   bin/cadenza-linux-arm64
	@zip -j dist/cadenza-$(VERSION)-darwin-amd64.zip  bin/cadenza-darwin-amd64
	@zip -j dist/cadenza-$(VERSION)-darwin-arm64.zip  bin/cadenza-darwin-arm64
	@zip -j dist/cadenza-$(VERSION)-windows-amd64.zip bin/cadenza-windows-amd64.exe
	@zip -j dist/cadenza-$(VERSION)-windows-arm64.zip bin/cadenza-windows-arm64.exe
	@echo "Release archives in dist/:"
	@ls -lh dist/

release-snapshot: build-all ## Build all platforms without packaging (smoke test)
	@echo "Snapshot build complete. Binaries in bin/"
	@ls -lh bin/

## Docker

docker:
	docker build -t cadenza:$(VERSION) .
	docker tag cadenza:$(VERSION) cadenza:latest

docker-run:
	docker run --rm -v $(PWD)/output:/app/output cadenza:latest --bpm 122 --key Am --no-llm

docker-compose-up:
	docker compose up cadenza

## Clean

clean:
	rm -rf bin/ output/ debug/ coverage.out coverage.html

## Help

help:
	@echo "Cadenza — AI-powered MIDI generator (backend)"
	@echo ""
	@echo "Build:"
	@echo "  make build          Build binary (bin/cadenza)"
	@echo "  make build-all      Cross-compile all platforms"
	@echo "  make docker         Build Docker image"
	@echo ""
	@echo "Test & Quality:"
	@echo "  make test           Run unit tests"
	@echo "  make test-race      Run tests with race detector"
	@echo "  make test-coverage  Generate coverage report"
	@echo "  make lint           Run golangci-lint"
	@echo "  make vuln           Run govulncheck (vulnerability scan)"
	@echo "  make sonar          Run SonarScanner locally (requires SONAR_TOKEN)"
	@echo "  make ci             Run full local CI pipeline (fmt+vet+lint+vuln+coverage)"
	@echo ""
	@echo "Run:"
	@echo "  make run            Interactive session (Claude, Am, 122 BPM)"
	@echo "  make run-offline    Run without LLM"
	@echo "  make run-ollama     Run with Ollama"
	@echo "  make listening-test Generate patterns for DAW comparison"
	@echo ""
	@echo "Docker:"
	@echo "  make docker-run     Run in container (offline mode)"
	@echo ""
	@echo "Release:"
	@echo "  make release        Build + zip all platforms into dist/"
