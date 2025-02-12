SHELL := /bin/bash
.SHELLFLAGS = -e -c
.DEFAULT_GOAL := help
.ONESHELL:

UNAME_S := $(shell uname -s)
UNAME_S_LOWER := $(shell echo '${UNAME_S}' | tr '[:upper:]' '[:lower:]')
UNAME_M := $(shell uname -m)

OPENTOFU_VERSION ?= 1.9.0

OPENTOFU_URL := https://github.com/opentofu/opentofu/releases/download/v${OPENTOFU_VERSION}/tofu_${OPENTOFU_VERSION}_${UNAME_S_LOWER}_${UNAME_M}.tar.gz
BIN_DIR := ${HOME}/.local/bin

TF := tofu

.PHONY: setup
setup:
ifeq ($(UNAME_S),Linux)
	@echo "Setting up Linux"
endif
ifeq ($(UNAME_S),Darwin)
	@echo "Setting up Darwin"
endif
	@echo "Common platform setup"
	@curl --fail -L ${OPENTOFU_URL} | tar -xz - ${TF}
	@mkdir -p ${BIN_DIR}
	@mv -v ${TF} ${BIN_DIR}/${TF}

.PHONY: help
help: ## Displays help info
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m\033[0m\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)
