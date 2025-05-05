SHELL := /bin/bash
.SHELLFLAGS = -e -c
.DEFAULT_GOAL := help
.ONESHELL:

YAML_FILES := $(shell find . -not -path "*/node_modules/*" -not -path "**/helm-templates/**" -not -path "**/cluster-nodes/**/templates/**" -type f -name '*.yml')
JS_DIRS := $(shell find . -not -path "*/node_modules/*" -not -path "*/.*" -type f -name 'package.json' -exec dirname {} \;)

JS_EXEC ?= bun
JS_INSTALL ?= install

.PHONY: $(MAKECMDGOALS)

fix: fix/js fix/md fix/yaml ## [all] Fix format and lint errors

format: fix/js/format fix/md/format fix/python/format fix/yaml/format ## [all] Format

fix/js: fix/js/format fix/js/eslint ## [js] Fix
fix/js/format:
	@echo "[fix/format/js] begin"
	@if ! [[ -d ./node_modules ]]; then \
		$(JS_EXEC) $(JS_INSTALL); \
	fi
	@$(JS_EXEC) run fix:prettier
	@echo "[fix/format/js] end"
fix/js/eslint:
	@echo "[fix/js/eslint] begin"
	for dir in $(JS_DIRS); do \
		cd $${dir}; \
		if ! [[ -d ./node_modules ]]; then \
			$(JS_EXEC) $(JS_INSTALL); \
		fi; \
		$(JS_EXEC) run build; \
		$(JS_EXEC) run fix:eslint; \
		cd -; \
	done
	@echo "[fix/js/eslint] end"

fix/md: fix/md/format ## [md] Fix

fix/md/format:
	@echo "[fix/format/md] begin"
	@if ! [[ -d ./node_modules ]]; then \
		$(JS_EXEC) $(JS_INSTALL); \
	fi
	@$(JS_EXEC) run fix:md
	@echo "[fix/format/md] end"

fix/python: fix/python/format ## [python] Fix
fix/python/format:
	@SHOULD_INSTALL=0
	@if ! [[ -d ./venv ]]; then \
		python3 -m venv ./venv; \
		SHOULD_INSTALL=1; \
	fi
	@. ./venv/bin/activate
	@if [[ $${SHOULD_INSTALL} -eq 1 ]]; then \
		pip install --upgrade pip; \
		pip install -r ./requirements.dev.txt; \
	fi
	@black .

fix/yaml: fix/yaml/format ## [yaml] Format
fix/yaml/format:
	@echo "[fix/format/yaml] begin"
	@for file in $(YAML_FILES); do \
		yq -i -P $${file}; \
	done
	@echo "[fix/format/yaml] end"

lint: lint/md lint/js lint/python lint/yaml ## [all] Lint
lint/md: ## [all] Lint MD
	@if ! [[ -d ./node_modules ]]; then \
		$(JS_EXEC) $(JS_INSTALL); \
	fi
	@$(JS_EXEC) run lint:md

lint/js: lint/js/format lint/js/eslint ## [all] Lint JS
	@if ! [[ -d ./node_modules ]]; then \
		$(JS_EXEC) $(JS_INSTALL); \
	fi
	@$(JS_EXEC) run lint
lint/js/eslint:
	@echo "[lint/eslint/js] begin"
	@for dir in $(JS_DIRS); do \
		cd $${dir}; \
		if ! [[ -d ./node_modules ]]; then \
			$(JS_EXEC) $(JS_INSTALL); \
		fi; \
		$(JS_EXEC) run build; \
		$(JS_EXEC) run lint:eslint; \
		cd -; \
	done
	@echo "[lint/check/js] end"
lint/js/format:
	@echo "[lint/format/js] begin"
	@if ! [[ -d ./node_modules ]]; then \
		$(JS_EXEC) $(JS_INSTALL); \
	fi
	@$(JS_EXEC) run lint:prettier
	@echo "[lint/format/js] end"

lint/python: lint/python/format ## [all] Lint Python
lint/python/format:
	@SHOULD_INSTALL=0
	@if ! [[ -d ./venv ]]; then \
		python3 -m venv ./venv; \
		SHOULD_INSTALL=1; \
	fi
	@. ./venv/bin/activate
	@if [[ $${SHOULD_INSTALL} -eq 1 ]]; then \
		pip install --upgrade pip; \
		pip install -r ./requirements.dev.txt; \
	fi
	@black --check .

lint/yaml: lint/yaml/format ## [all] Lint YAML
lint/yaml/format:
	@echo "[lint/format/yaml] begin"
	@for file in $(YAML_FILES); do \
		yq -P $${file} > /dev/null; \
	done
	@echo "[lint/format/yaml] end"

setup: setup/js

setup/js : ## [js] Setup JS
	@if command -v bun &> /dev/null; then \
		exit 0; \
	fi
	curl -fsSL https://bun.sh/install | bash

.PHONY: help
help: ## Displays help info
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m\033[0m\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)
