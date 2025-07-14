SHELL := /bin/bash
.SHELLFLAGS = -e -c
.DEFAULT_GOAL := help
.ONESHELL:
.SILENT:

.PHONY: $(MAKECMDGOALS)

setup: setup/gotools setup/templ ## Setup development environment

setup/gotools: ## Install go tools
	go install golang.org/x/tools/gopls@latest
	go install github.com/cweill/gotests/gotests@v1.6.0
	go install github.com/josharian/impl@v1.4.0
	go install github.com/haya14busa/goplay/cmd/goplay@v1.0.0
	go install github.com/go-delve/delve/cmd/dlv@latest
	go install honnef.co/go/tools/cmd/staticcheck@latest

setup/templ: ## Install templ tool
	go install github.com/a-h/templ/cmd/templ@latest

export INSTALL_VERSION?=$(shell git describe --tags --abbrev=0)

install/linux: env-INSTALL_VERSION ## Install startup service on Linux
	if ! [[ -f /usr/local/bin/autobutler ]]; then \
		curl \
			--fail \
			-L \
			https://github.com/autobutler-ai/autobutler.ai/releases/download/$(INSTALL_VERSION)/autobutler_darwin_arm64.tar.gz | sudo tar -x -C /usr/local/bin ; \
	fi
	sudo cp -f \
		./deployments/autobutler.service \
		/etc/systemd/system/
	sudo systemctl restart autobutler
	echo "Installed autobutler successfully. Will run at startup."

install/mac: env-INSTALL_VERSION ## Install startup service on Mac
	if ! [[ -f /Applications/autobutler ]]; then \
		curl \
			--fail \
			-L \
			https://github.com/autobutler-ai/autobutler.ai/releases/download/$(INSTALL_VERSION)/autobutler_darwin_arm64.tar.gz | tar -x -C /Applications/ ; \
	fi
	sudo launchctl unload /Library/LaunchDaemons/com.autobutler.autobutler.plist > /dev/null 2>&1 || true
	sudo cp -f \
		./deployments/com.autobutler.autobutler.plist \
		/Library/LaunchDaemons/com.autobutler.autobutler.plist
	sudo launchctl load /Library/LaunchDaemons/com.autobutler.autobutler.plist
	echo "Installed autobutler successfully. Will run at startup."

generate: ## Generate templ files
	templ generate

build: generate ## Build backend
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
	templ fmt .

lint: ## Lint code
	gofmt -s -w .
	go vet ./...
	templ fmt -fail .

upgrade: ## Upgrade dependencies
	go get -u ./...
	go mod tidy

fix: ## Fix code issues
	go mod tidy
	$(MAKE) format

serve: generate env-LLM_AZURE_API_KEY ## Serve backend
	go run main.go serve

watch: env-LLM_AZURE_API_KEY ## Watch backend for changes
	templ generate \
		--watch \
		--proxy="http://localhost:8080" \
		--cmd="go run . serve"

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
