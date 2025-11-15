SHELL := /bin/bash
.SHELLFLAGS = -e -c
.DEFAULT_GOAL := help
.ONESHELL:
.SILENT:

.PHONY: $(MAKECMDGOALS)

ifneq (,$(wildcard ./.env))
    include .env
    export
endif

clean: clean/build clean/tests

clean/build:
	rm -rf ./build

clean/tests:
	rm -rf playwright-report/
	rm -rf test-results/

setup: setup/gotools setup/sqlc setup/templ ## Setup development environment

setup/gotools: ## Install go tools
	go install golang.org/x/tools/gopls@latest
	go install github.com/cweill/gotests/gotests@v1.6.0
	go install github.com/josharian/impl@v1.4.0
	go install github.com/haya14busa/goplay/cmd/goplay@v1.0.0
	go install github.com/go-delve/delve/cmd/dlv@latest
	go install honnef.co/go/tools/cmd/staticcheck@latest

setup/sqlc: ## Install sqlc tool
	go install github.com/sqlc-dev/sqlc/cmd/sqlc@v1.30.0

setup/templ: ## Install templ tool
	go install github.com/a-h/templ/cmd/templ@latest

export INSTALL_VERSION?=$(shell git describe --tags --abbrev=0)
export GOPROXY ?= https://proxy.golang.org,direct

install/linux: env-INSTALL_VERSION ## Install startup service on Linux
	if ! [[ -f /usr/local/bin/autobutler ]]; then \
		curl \
			--fail \
			-L \
			https://github.com/autobutler-org/autobutler.org/releases/download/$(INSTALL_VERSION)/autobutler_darwin_arm64.tar.gz | sudo tar -x -C /usr/local/bin ; \
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
			https://github.com/autobutler-org/autobutler.org/releases/download/$(INSTALL_VERSION)/autobutler_darwin_arm64.tar.gz | tar -x -C /Applications/ ; \
	fi
	sudo launchctl unload /Library/LaunchDaemons/com.autobutler.autobutler.plist > /dev/null 2>&1 || true
	sudo cp -f \
		./deployments/com.autobutler.autobutler.plist \
		/Library/LaunchDaemons/com.autobutler.autobutler.plist
	sudo launchctl load /Library/LaunchDaemons/com.autobutler.autobutler.plist
	echo "Installed autobutler successfully. Will run at startup."

generate: generate/sqlc generate/templ ## Generate files

generate/sqlc: ## Generate templ files
	sqlc generate

generate/templ: ## Generate templ files
	templ generate

build: generate ## Build backend
	mkdir -p ./build
	go build -o ./build/autobutler

build/all: build/linux build/mac ## Build all backends

build/linux: build/linux/amd64 build/linux/arm64 ## Build linux backends
build/linux/amd64: ## Build linux backends
	GOOS=linux GOARCH=amd64 go build -o ./build/autobutler-linux-amd64 main.go
build/linux/arm64: ## Build linux backends
	GOOS=linux GOARCH=arm64 go build -o ./build/autobutler-linux-arm64 main.go

build/mac: build/mac/amd64 build/mac/arm64 ## Build macOS backends
build/mac/arm64: ## Build macOS backends
	GOOS=darwin GOARCH=arm64 go build -o ./build/autobutler-mac-arm64 main.go

test: test/e2e
test/e2e:
	npm run test/e2e

format: format/go format/templ format/js format/ts format/css ## Format code

format/go: ## Format Go code
	go fmt ./...

format/templ: ## Format templ files
	templ fmt .

format/js: ## Format JavaScript files
	npm run format:js

format/ts: ## Format TypeScript files
	npm run format:ts

format/css: ## Format CSS files
	npm run format:css

lint: lint/go lint/sqlc lint/templ lint/js lint/ts lint/css lint/yaml ## Lint code

lint/go: ## Lint Go code
	gofmt -s -w .
	go vet ./...

lint/sqlc: ## Lint sqlc
	sqlc vet

lint/templ: ## Lint templ files
	templ fmt -fail .

lint/js: ## Lint JavaScript files
	npm run lint:js

lint/ts: ## Lint TypeScript files
	npm run lint:ts

lint/css: ## Lint CSS files
	npm run lint:css

lint/yaml: ## Lint YAML files
	npm run lint:yaml

fix: fix/go fix/js fix/ts fix/css ## Fix code issues

fix/go: ## Fix Go code issues
	go mod tidy
	go fmt ./...
	templ fmt .

fix/js: ## Fix JavaScript code issues
	npm run format:js

fix/ts: ## Fix TypeScript code issues
	npm run format:ts

fix/css: ## Fix CSS code issues
	npm run format:css

upgrade: upgrade/go upgrade/js ## Upgrade dependencies

upgrade/go: generate ## Upgrade dependencies (go)
	go get -u ./...
	$(MAKE) tidy

upgrade/js: ## Upgrade dependencies (js)
	npm run check-updates
	npm install

tidy: ## Tidy go mod
	go mod tidy

serve: generate ## Serve backend
	go run main.go serve

watch: ## Watch backend for changes
	templ generate \
		-watch \
		-watch-pattern='(.+\.go$$)|(.+\.templ$$)|(.+_templ\.txt$$)|(.+\.js$$)|(.+\.css$$)' \
		-proxy="http://localhost:8080" \
		-cmd="go run . serve"

version: ## Print version
	go run main.go version

env-%: ## Check for env var
	if [ -z "$($*)" ]; then \
		echo "Error: Environment variable '$*' is not set."; \
		exit 1; \
	fi

help: ## Displays help info
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m\033[0m\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)
