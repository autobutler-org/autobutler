SHELL := /bin/bash
.DEFAULT_GOAL := all
.SHELLFLAGS = -e -c
.ONESHELL:

venv/bin/pip:
	@python3 -m venv venv

venv/bin/butlerctl: venv/bin/pip
	@venv/bin/pip install -e .

.PHONY: all
all: venv venv/bin/butlerctl
	@venv/bin/butlerctl --help

.PHONY: help
help: ## Displays help info
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m\033[0m\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)
