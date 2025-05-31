SHELL := /bin/bash
.DEFAULT_GOAL := all
.SHELLFLAGS = -e -c
.ONESHELL:

export PORT ?= 8000

venv/bin/pip:
	python3 -m venv venv

venv/bin/butlerapi: venv/bin/pip
	venv/bin/pip install -e .

export LLM_MAX_TOKENS ?= 1024

.PHONY: run
run: venv/bin/butlerapi ## Run the butlerapi command
	@. ./venv/bin/activate
	@butlerapi

.PHONY: kill
kill: ## Kill the butlerapi command
	@killall $$(lsof -i :$(PORT) | tail -n 1 | awk '{print $1}') || true

PYTEST_ARGS ?= -vv --tb=short

.PHONY: test
test: venv/bin/butlerapi ## Run the tests
	@. ./venv/bin/activate
	pytest $(PYTEST_ARGS) tests/

.PHONY: help
help: ## Displays help info
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m\033[0m\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)
