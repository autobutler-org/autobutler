SHELL := /bin/bash
.SHELLFLAGS = -e -c
.DEFAULT_GOAL := help
.ONESHELL:
.SILENT:

.PHONY: $(MAKECMDGOALS)

YAML_FILES := $(shell find . -not -path "*/node_modules/*" -not -path "**/helm-templates/**" -not -path "**/cluster-nodes/**/templates/**" -type f -name '*.yml')
JS_DIRS := $(shell find . -not -path "*/node_modules/*" -not -path "*/.*" -type f -name 'package.json' -exec dirname {} \;)

JS_EXEC ?= bun
JS_INSTALL ?= install

UNAME_S := $(shell uname -s)

.PHONY: $(MAKECMDGOALS)

fix: fix/js fix/md fix/yaml ## [all] Fix format and lint errors

format: format/go format/js format/md format/python format/yaml ## [all] Format

format/go: ## [golang] Format
	go fmt ./...

fix/js: format/js fix/js/eslint ## [js] Fix
format/js:
	echo "[fix/format/js] begin"
	if ! [[ -d ./node_modules ]]; then \
		$(JS_EXEC) $(JS_INSTALL); \
	fi
	$(JS_EXEC) run fix:prettier
	echo "[fix/format/js] end"
fix/js/eslint:
	echo "[fix/js/eslint] begin"
	for dir in $(JS_DIRS); do \
		cd $${dir}; \
		if ! [[ -d ./node_modules ]]; then \
			$(JS_EXEC) $(JS_INSTALL); \
		fi; \
		$(JS_EXEC) run build; \
		$(JS_EXEC) run fix:eslint; \
		cd -; \
	done
	echo "[fix/js/eslint] end"

fix/md: format/md ## [md] Fix

format/md:
	echo "[fix/format/md] begin"
	if ! [[ -d ./node_modules ]]; then \
		$(JS_EXEC) $(JS_INSTALL); \
	fi
	$(JS_EXEC) run fix:md
	echo "[fix/format/md] end"

fix/python: format/python ## [python] Fix
format/python:
	SHOULD_INSTALL=0
	if ! [[ -d ./venv ]]; then \
		python3 -m venv ./venv; \
		SHOULD_INSTALL=1; \
	fi
	. ./venv/bin/activate
	if [[ $${SHOULD_INSTALL} -eq 1 ]]; then \
		pip install --upgrade pip; \
		pip install -r ./requirements.dev.txt; \
	fi
	black .
	isort --profile black .

fix/yaml: format/yaml ## [yaml] Format
format/yaml:
	echo "[fix/format/yaml] begin"
	for file in $(YAML_FILES); do \
		yq -i -P $${file}; \
	done
	echo "[fix/format/yaml] end"

lint: lint/go lint/md lint/js lint/python lint/yaml ## [all] Lint
lint/md: ## [all] Lint MD
	if ! [[ -d ./node_modules ]]; then \
		$(JS_EXEC) $(JS_INSTALL); \
	fi
	$(JS_EXEC) run lint:md

lint/go: lint/go/format lint/go/vet ## [all] Lint Golang

lint/go/format:
	gofmt -s -w .

lint/go/vet:
	# iterate over al folders with go.mod
	echo "[lint/vet/go] begin"
	for dir in $(shell find . -type f -name 'go.mod' -exec dirname {} \;); do \
		pushd $${dir}; \
		echo "[lint/vet/go] running go vet in $${dir}"; \
		go vet ./...; \
		popd; \
	done
	echo "[lint/vet/go] end"

lint/js: lint/js/format lint/js/eslint ## [all] Lint JS
	if ! [[ -d ./node_modules ]]; then \
		$(JS_EXEC) $(JS_INSTALL); \
	fi
	$(JS_EXEC) run lint
lint/js/eslint:
	echo "[lint/eslint/js] begin"
	for dir in $(JS_DIRS); do \
		cd $${dir}; \
		if ! [[ -d ./node_modules ]]; then \
			$(JS_EXEC) $(JS_INSTALL); \
		fi; \
		$(JS_EXEC) run build; \
		$(JS_EXEC) run lint:eslint; \
		cd -; \
	done
	echo "[lint/check/js] end"
lint/js/format:
	echo "[lint/format/js] begin"
	if ! [[ -d ./node_modules ]]; then \
		$(JS_EXEC) $(JS_INSTALL); \
	fi
	$(JS_EXEC) run lint:prettier
	echo "[lint/format/js] end"

lint/python: lint/python/format ## [all] Lint Python
lint/python/format:
	SHOULD_INSTALL=0
	if ! [[ -d ./venv ]]; then \
		python3 -m venv ./venv; \
		SHOULD_INSTALL=1; \
	fi
	. ./venv/bin/activate
	if [[ $${SHOULD_INSTALL} -eq 1 ]]; then \
		pip install --upgrade pip; \
		pip install -r ./requirements.dev.txt; \
	fi
	black --check .
	isort --profile black --check-only .

lint/yaml: lint/yaml/format ## [all] Lint YAML
lint/yaml/format:
	echo "[lint/format/yaml] begin"
	for file in $(YAML_FILES); do \
		yq -P $${file} > /dev/null; \
	done
	echo "[lint/format/yaml] end"

setup: setup/js setup/db

setup/js : ## [js] Setup JS
	if command -v bun &> /dev/null; then \
		exit 0; \
	fi
	curl -fsSL https://bun.sh/install | bash

setup/db: ## [db] Setup DB
ifeq ($(UNAME_S),Linux)
	if ! command -v sqlite3 &> /dev/null; then \
		sudo apt-get install -y sqlite3; \
	fi
else ifeq ($(UNAME_S),Darwin)
	if ! command -v sqlite3 &> /dev/null; then \
		brew install sqlite; \
	fi
endif

export LLM_URL ?= https://autobutler-eus2.services.ai.azure.com/models/chat/completions
export LLM_SYSTEM_PROMPT_FILE ?= system.prompt
LLM_ARGS := api-version=2024-05-01-preview
LLM_MODEL := autobutler_Ministral-3B
export LLM_TOP_P ?= 0.1
export LLM_TEMP ?= 0.8
export LLM_MAX_TOKENS ?= 2048
llm: env-LLM_AZURE_API_KEY env-LLM_SYSTEM_PROMPT_FILE env-LLM_PROMPT env-LLM_URL env-LLM_TOP_P env-LLM_TEMP env-LLM_MAX_TOKENS ## Call LLM
	curl --silent -X POST "$(LLM_URL)?$(LLM_ARGS)" \
	    -H "Content-Type: application/json" \
	    -H "Authorization: Bearer $(LLM_AZURE_API_KEY)" \
	    -d "{ \
	            \"messages\": [ \
	                { \
	                    \"role\": \"system\", \
	                    \"content\": \"$(shell cat $(LLM_SYSTEM_PROMPT_FILE))\" \
	                }, \
	                { \
	                    \"role\": \"user\", \
	                    \"content\": \"$(LLM_PROMPT)\" \
	                } \
	            ], \
	            \"max_tokens\": $(LLM_MAX_TOKENS), \
	            \"temperature\": $(LLM_TEMP), \
	            \"top_p\": $(LLM_TOP_P), \
	            \"model\": \"$(LLM_MODEL)\" \
	    }"


env-%: ## Check for env var
	if [ -z "$($*)" ]; then \
		echo "Error: Environment variable '$*' is not set."; \
		exit 1; \
	fi

.PHONY: help
help: ## Displays help info
	awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m\033[0m\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)
