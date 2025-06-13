SHELL := /bin/bash
.SHELLFLAGS = -e -c
.DEFAULT_GOAL := help
.ONESHELL:
.SILENT:

.PHONY: $(MAKECMDGOALS)

build: ## Build backend
	mkdir -p ./build
	go build -o ./build/dotai

format: ## Format code
	go fmt ./...

lint: ## Lint code
	gofmt -s -w .
	go vet ./...

serve: ## Serve backend
	go run main.go

version: ## Print version
	go run main.go version # TODO: Implement version command

help: ## Displays help info
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m\033[0m\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)
