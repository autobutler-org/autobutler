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

MAIN := ./cmd/autobutler/main.go

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
	$(MAKE) lint/go

build: ## Build backend and frontend
	$(MAKE) -j2 build/backend build/frontend

build/backend: generate ## Build backend
	mkdir -p ./build
	go build -o ./build/autobutler $(MAIN)

build/frontend: ## Build frontend
	npm run build --prefix ./app

build/all: build/linux build/mac ## Build all backends

build/linux: build/linux/amd64 build/linux/arm64 ## Build linux backends
build/linux/amd64: ## Build linux backends
	GOOS=linux GOARCH=amd64 go build -o ./build/autobutler-linux-amd64 $(MAIN)
build/linux/arm64: ## Build linux backends
	GOOS=linux GOARCH=arm64 go build -o ./build/autobutler-linux-arm64 $(MAIN)

build/mac: build/mac/amd64 build/mac/arm64 ## Build macOS backends
build/mac/arm64: ## Build macOS backends
	GOOS=darwin GOARCH=arm64 go build -o ./build/autobutler-mac-arm64 $(MAIN)

PRINT_COVERAGE ?= 0

test: test/unit test/e2e
test/unit: ## Run unit tests
	# Generate coverage report for unit tests
	go test -v ./... \
		-coverprofile=coverage.out \
		-covermode=atomic
	# Apply coverage ignore directives
	./scripts/apply-coverage-ignore.bash \
		coverage.out
	# Generate coverage report as HTMl
	go tool cover \
		-html=coverage.out.ignored \
		-o coverage.html
	# Display coverage summary in terminal
	if [[ "$(PRINT_COVERAGE)" = "1" || "$(PRINT_COVERAGE)" = "true" ]] ; then
		go tool cover \
			-func=coverage.out.ignored
	fi
test/e2e:
	npm run test/e2e

format: format/go format/templ format/ts format/css ## Format code

format/go: ## Format Go code
	gofmt -s -w .

format/templ: ## Format templ files
	templ fmt .

format/ts: ## Format TypeScript files
	npm run format --prefix ./app

format/css: ## Format CSS files
	npm run format:css

lint: lint/go lint/sqlc lint/templ lint/ts lint/css lint/yaml ## Lint code

lint/go: ## Lint Go code
	go vet ./...

lint/sqlc: ## Lint sqlc
	sqlc vet

lint/templ: ## Lint templ files
	templ fmt -fail .

lint/ts: ## Lint TypeScript files
	npm run lint --prefix ./app

lint/css: ## Lint CSS files
	npm run lint:css

lint/yaml: ## Lint YAML files
	npm run lint:yaml

fix: fix/go fix/ts fix/css ## Fix code issues

fix/go: ## Fix Go code issues
	go mod tidy
	go fmt ./...
	templ fmt .

fix/ts: ## Fix TypeScript code issues
	npm run format:ts

fix/css: ## Fix CSS code issues
	npm run format:css

upgrade: upgrade/go upgrade/ts ## Upgrade dependencies

upgrade/go: generate ## Upgrade dependencies (go)
	go get -u ./...
	$(MAKE) tidy

upgrade/ts: ## Upgrade dependencies (ts)
	npm run check-updates
	npm install
	npm run check-updates --prefix ./app
	npm install --prefix ./app

tidy: ## Tidy go mod
	go mod tidy

serve:
	$(MAKE) -j2 serve/backend serve/frontend

watch:
	$(MAKE) -j2 watch/backend watch/frontend

serve/backend: generate ## Serve backend
	go run $(MAIN) serve

watch/backend: ## Watch backend for changes
	templ generate \
		-watch \
		-watch-pattern='(.+\.go$$)|(.+\.templ$$)|(.+_templ\.txt$$)' \
		-proxy="http://localhost:8080" \
		-cmd="go run $(MAIN) serve"

serve/frontend: ## Serve frontend
	npm run dev --prefix ./app

watch/frontend: serve/frontend ## Watch frontend

version: ## Print version
	go run $(MAIN) version

env-%: ## Check for env var
	if [ -z "$($*)" ]; then \
		echo "Error: Environment variable '$*' is not set."; \
		exit 1; \
	fi

help: ## Displays help info
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m\033[0m\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)
