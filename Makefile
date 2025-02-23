SHELL := /bin/bash
.SHELLFLAGS = -e -c
.DEFAULT_GOAL := help
.ONESHELL:

.PHONY: setup
setup: ## [all] Setup
	@$(MAKE) setup/infra

.PHONY: setup/infra
setup/infra: ## [infra] Setup
	@$(MAKE) -C infra setup

.PHONY: lint
lint: ## [all] Lint
	@npm run lint

.PHONY: help
help: ## Displays help info
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m\033[0m\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)
