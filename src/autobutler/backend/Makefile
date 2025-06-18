SHELL := /bin/bash
.SHELLFLAGS = -e -c
.DEFAULT_GOAL := help
.ONESHELL:
.SILENT:

.PHONY: $(MAKECMDGOALS)

build: ## Build backend
	mkdir -p ./build
	go build -o ./build/autobutler

build/all: build/linux build/mac build/windows ## Build all backends

build/linux: build/linux/amd64 build/linux/arm64 ## Build linux backends
build/linux/amd64: ## Build linux backends
	GOOS=linux GOARCH=amd64 go build -o ./build/autobutler-linux-amd64 main.go
build/linux/arm64: ## Build linux backends
	GOOS=linux GOARCH=arm64 go build -o ./build/autobutler-linux-arm64 main.go

build/mac: build/mac/amd64 build/mac/arm64 ## Build macOS backends
build/mac/arm64: ## Build macOS backends
	GOOS=darwin GOARCH=arm64 go build -o ./build/autobutler-mac-arm64 main.go

build/windows: build/windows/amd64 build/windows/arm64 ## Build windows backends
build/windows/amd64: ## Build windows backends
	GOOS=windows GOARCH=amd64 go build -o ./build/autobutler-windows-amd64.exe main.go
build/windows/arm64: ## Build windows backends
	GOOS=windows GOARCH=arm64 go build -o ./build/autobutler-windows-arm64.exe main.go

format: ## Format code
	go fmt ./...

lint: ## Lint code
	gofmt -s -w .
	go vet ./...

serve: env-LLM_AZURE_API_KEY ## Serve backend
	go run main.go serve

version: ## Print version
	go run main.go version

exercise: env-LLM_AZURE_API_KEY ## Exercise the backend chat feature
	killall main || true
	$(MAKE) serve &
	sleep 5
	echo ""
	curl \
		--silent \
		-X GET \
		"http://localhost:8080/chat?prompt=How+much+milk+is+in+the+house" | tee ./exercise.json
	killall main || true

env-%: ## Check for env var
	if [ -z "$($*)" ]; then \
		echo "Error: Environment variable '$*' is not set."; \
		exit 1; \
	fi

help: ## Displays help info
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m\033[0m\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)
