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

export GOTOOLCHAIN=go1.25.0+auto

MAIN := ./cmd/autobutler/main.go
EXE := ./build/autobutler

_ensure/public:
	mkdir -p ./internal/server/public
	touch ./internal/server/public/stub.txt

clean: clean/build clean/tests

clean/build:
	rm -rf ./build

clean/tests:
	rm -rf playwright-report/
	rm -rf test-results/

setup: setup/gotools setup/sqlc setup/air setup/node setup/playwright ## Setup development environment

setup/gotools: ## Install go tools
	$(GO) install golang.org/x/tools/gopls@latest
	$(GO) install github.com/cweill/gotests/gotests@v1.6.0
	$(GO) install github.com/josharian/impl@v1.4.0
	$(GO) install github.com/haya14busa/goplay/cmd/goplay@v1.0.0
	$(GO) install github.com/go-delve/delve/cmd/dlv@latest
	$(GO) install honnef.co/go/tools/cmd/staticcheck@latest

setup/sqlc: ## Install sqlc tool
	$(GO) install github.com/sqlc-dev/sqlc/cmd/sqlc@v1.30.0

setup/air: ## Install air tool
	$(GO) install github.com/air-verse/air@latest

setup/node: ## Setup Node.js environment
	npm install --prefix ./app

setup/playwright: ## Install Playwright browsers
	npx playwright install \
		--with-deps

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

generate: generate/sqlc ## Generate files

generate/sqlc: ## Generate sqlc files
	sqlc generate

build: ## Build backend and frontend
	# Order matters: frontend must be built before backend
	$(MAKE) build/frontend
	$(MAKE) build/backend

build/backend: generate ## Build backend
	$(MAKE) _ensure/public
	mkdir -p ./build
	$(GO) build -o $(EXE) $(MAIN)

build/frontend: ## Build frontend
	npm run build --prefix ./app
	# Explanation: https://github.com/gin-gonic/gin/issues/2654#issuecomment-815823804
	cp -f ./internal/server/public/index.html ./internal/server/public/index.htm
	$(MAKE) _ensure/public

build/lsusb: ## Build lsusb utility
	$(GO) build -o ./build/lsusb ./cmd/lsusb/main.go

LSUSB_ARGS ?= -storage

lsusb: env-LSUSB_ARGS ## Run lsusb utility
	$(MAKE) build/lsusb
	sudo ./build/lsusb $(LSUSB_ARGS)

PRINT_COVERAGE ?= 0

test: test/unit test/e2e

test/unit: ## Run unit tests
	$(MAKE) test/unit/backend
	$(MAKE) test/unit/frontend

test/unit/backend: ## Run unit tests for backend
	$(MAKE) _ensure/public
	# Generate coverage report for unit tests
	$(GO) test -v ./... \
		-coverprofile=coverage.out \
		-covermode=atomic
	# Apply coverage ignore directives
	./scripts/apply-coverage-ignore.bash \
		coverage.out
	# Generate coverage report as HTMl
	$(GO) tool cover \
		-html=coverage.out.ignored \
		-o coverage.html
	# Display coverage summary in terminal
	if [[ "$(PRINT_COVERAGE)" = "1" || "$(PRINT_COVERAGE)" = "true" ]] ; then
		$(GO) tool cover \
			-func=coverage.out.ignored
	fi

test/unit/frontend: ## Run unit tests for frontend
	npm run test:unit --prefix ./app

test/e2e:
	npm run test:e2e --prefix ./app

format: format/go format/ts ## Format code

format/go: ## Format Go code
	gofmt -s -w .

format/ts: ## Format TypeScript files
	npm run format --prefix ./app

lint: lint/go lint/sqlc lint/ts lint/yaml ## Lint code

lint/go: ## Lint Go code
	$(MAKE) _ensure/public
	$(GO) vet ./...

lint/sqlc: ## Lint sqlc
	sqlc vet

lint/ts: ## Lint TypeScript files
	npm run lint:ts --prefix ./app

lint/yaml: ## Lint YAML files
	npm run lint:yaml --prefix ./app

fix: fix/go fix/ts ## Fix code issues

fix/go: ## Fix Go code issues
	$(GO) mod tidy
	$(GO) fmt ./...
	templ fmt .

fix/ts: ## Fix TypeScript code issues
	npm run fix --prefix ./app

upgrade: upgrade/go upgrade/ts ## Upgrade dependencies

upgrade/go: generate ## Upgrade dependencies (go)
	$(GO) get -u ./...
	$(MAKE) tidy

upgrade/ts: ## Upgrade dependencies (ts)
	npm run check-updates --prefix ./app
	npm install --prefix ./app

tidy: ## Tidy go mod
	$(GO) mod tidy

serve:
	$(MAKE) -j2 serve/backend serve/frontend

watch:
	$(MAKE) -j2 watch/backend watch/frontend

AS_ROOT ?= 0

GO := $(shell which go)
AIR := $(shell which air)
ifeq ($(AS_ROOT), 1)
	GO := sudo $(GO)
	AIR := sudo $(AIR)
endif

serve/backend: generate ## Serve backend
	$(GO) run $(MAIN) serve

serve/production: ## Serve backend in production mode
	$(MAKE) build
	$(EXE) serve

watch/backend: build/backend ## Watch backend for changes
ifeq ($(AS_ROOT), 1)
	$(AIR) \
		--build.cmd "sudo $(MAKE) build/backend" \
		--build.entrypoint "$(EXE)" \
		--build.args_bin "serve" \
		--build.exclude_dir "app,build,cd,datalinks,docs,internal/db,node_modules,playwright-report,scripts,sql,teststest-results"
else
	$(AIR) \
		--build.cmd "$(MAKE) build/backend" \
		--build.entrypoint "$(EXE)" \
		--build.args_bin "serve" \
		--build.exclude_dir "app,build,cd,datalinks,docs,internal/db,node_modules,playwright-report,scripts,sql,teststest-results"
endif

serve/frontend: ## Serve frontend
	npm run dev --prefix ./app

watch/frontend: serve/frontend ## Watch frontend

version: ## Print version
	$(GO) run $(MAIN) version

env-%: ## Check for env var
	if [ -z "$($*)" ]; then \
		echo "Error: Environment variable '$*' is not set."; \
		exit 1; \
	fi

help: ## Displays help info
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m\033[0m\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)
