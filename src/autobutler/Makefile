SHELL := /bin/bash
.SHELLFLAGS = -e -c
.DEFAULT_GOAL := help
.ONESHELL:

.PHONY: $(MAKECMDGOALS)

run: ## Run all components
	@echo "Running all components..."
	$(MAKE) -C ./ui run
	$(MAKE) -C ./router run
	$(MAKE) -C ./llm run

kill:
	$(MAKE) -C ./llm kill
	$(MAKE) -C ./router kill
	$(MAKE) -C ./ui kill

.PHONY: help
help: ## Displays help info
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m\033[0m\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)
