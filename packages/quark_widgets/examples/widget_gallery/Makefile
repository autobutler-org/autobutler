SHELL := /usr/bin/env
.SHELLFLAGS = bash -e -c
.DEFAULT_GOAL := help
.ONESHELL:
.SILENT:

##@ Development

PLATFORM ?= web
FLUTTER_BUILD_MODE ?= debug
SERVE_PLATFORM ?= web-server

.PHONY: build
build: ## Build the gallery
	flutter build $(PLATFORM) --$(FLUTTER_BUILD_MODE)

.PHONY: serve
serve: ## Serve the gallery on a local web server
	flutter run -d $(SERVE_PLATFORM)

.PHONY: watch
watch: ## Run the gallery in Chrome with hot reload
	flutter run -d chrome

.PHONY: generate/docs
generate/docs: ## Regenerate lib/docs.g.dart from the package's /// class docs
	$(MAKE) -C ../.. generate/docs

##@ Helpers

.PHONY: help
help: ## Displays help info
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m\033[0m\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)
