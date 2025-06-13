SHELL := /bin/bash
.SHELLFLAGS = -e -c
.DEFAULT_GOAL := help
.ONESHELL:
.SILENT:

.PHONY: $(MAKECMDGOALS)

# If the first argument is "run"...
ifeq (run,$(firstword $(MAKECMDGOALS)))
  # use the rest as arguments for "run"
  RUN_ARGS := $(wordlist 2,$(words $(MAKECMDGOALS)),$(MAKECMDGOALS))
  # ...and turn them into do-nothing targets
  $(eval $(RUN_ARGS):;@:)
endif
ifeq (run/build,$(firstword $(MAKECMDGOALS)))
  # use the rest as arguments for "run"
  RUN_ARGS := $(wordlist 2,$(words $(MAKECMDGOALS)),$(MAKECMDGOALS))
  # ...and turn them into do-nothing targets
  $(eval $(RUN_ARGS):;@:)
endif

build: ## Build backend as `ab`
	mkdir -p ./build
	go build -o ./build/ab github.com/exokomodo/exoflow/autobutler/backend

format: ## Format code
	go fmt ./...

lint: ## Lint code
	gofmt -s -w .
	go vet ./...

serve: ## Serve backend
	go run main.go serve

version: ## Print version
	go run main.go version

exercise: ## Exercise the backend chat feature
	killall main || true
	$(MAKE) serve &
	sleep 5
	echo ""
	curl \
		--silent \
		-X GET \
		"http://localhost:8080/chat?prompt=How+much+milk+is+in+the+house" | tee ./e2e.json
	killall main || true

help: ## Displays help info
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m\033[0m\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)
