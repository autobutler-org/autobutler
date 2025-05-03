SHELL := /bin/bash
.SHELLFLAGS = -e -c
.DEFAULT_GOAL := help
.ONESHELL:

.PHONY: $(MAKECMDGOALS)

JS_EXEC ?= bun
JS_INSTALL ?= install
export PORT ?= 8082

run:
	if ! [[ -d ./node_modules ]]; then \
		$(JS_EXEC) $(JS_INSTALL); \
	fi; \
	$(JS_EXEC) run dev &

kill:
	killall $$(lsof -i :$(PORT) | tail -n 1 | awk '{print $1}') || true

.PHONY: help
help: ## Displays help info
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m\033[0m\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)
