SHELL := /bin/bash
.DEFAULT_GOAL := all
.SHELLFLAGS = -e -c
.ONESHELL:
.SILENT:

export PORT ?= 8000
MIN_PYTHON_VERSION := 3.13

venv/bin/pip: check/python-version
	python3 -m venv venv

venv/lib/python$(MIN_PYTHON_VERSION)/site-packages: venv/bin/pip
	venv/bin/pip install -e ../../../libs/flowtools
	venv/bin/pip install -e .

.PHONY: venv
venv: venv/lib/python$(MIN_PYTHON_VERSION)/site-packages ## Create the virtual environment and install dependencies

export LLM_MAX_TOKENS ?= 1024

.PHONY: run
run: venv/lib/python$(MIN_PYTHON_VERSION)/site-packages ## Run the butlerapi command
	. ./venv/bin/activate
	butlerapi

.PHONY: kill
kill: ## Kill the butlerapi command
	killall $$(lsof -i :$(PORT) | tail -n 1 | awk '{print $1}') || true

PYTEST_ARGS ?= -vv --tb=short

.PHONY: test
test: venv/lib/python$(MIN_PYTHON_VERSION)/site-packages ## Run the tests
	$(MAKE) -C ../../../libs/flowtools test

.PHONY: check/python-version
check/python-version: ## Check the version of python
	test $$(python3 --version | cut -d' ' -f2 | cut -d'.' -f1,2) = $(MIN_PYTHON_VERSION) || (echo "Python $(MIN_PYTHON_VERSION) is required." && exit 1)

.PHONY: help
help: ## Displays help info
	awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m\033[0m\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)
