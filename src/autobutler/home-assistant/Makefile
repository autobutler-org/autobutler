SHELL := /bin/bash
.SHELLFLAGS = -e -c
.DEFAULT_GOAL := help
.ONESHELL:

.PHONY: $(MAKECMDGOALS)

PWD := $(shell pwd)

UNAME_S := $(shell uname -s)

run: ## Run home assistant
	@mkdir -p $(PWD)/config
	@if command -v rsync >/dev/null 2>&1; then \
		rsync -ar --delete $(PWD)/integrations/ $(PWD)/config/custom_components; \
	else \
		rm -rf $(PWD)/config/custom_components
		cp -r $(PWD)/integrations $(PWD)/config/custom_components; \
	fi
	@docker compose up -d
	@sleep 5
ifeq ($(UNAME_S),Linux)
	@xdg-open http://localhost:8123 &
else
	@open http://localhost:8123 &
endif
	@docker compose logs -f

kill: ## Kill home assistant
	docker compose down

clean: ## Clean home assistant
	docker compose down --volumes --remove-orphans
	sudo rm -rf $(PWD)/config

.PHONY: help
help: ## Displays help info
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m\033[0m\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)
