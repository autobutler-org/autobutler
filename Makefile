SHELL := /bin/bash
.SHELLFLAGS = -e -c
.DEFAULT_GOAL := help
.ONESHELL:
.SILENT:

ifneq (,$(wildcard ./.env))
    include .env
    export
endif

export GOTOOLCHAIN=go1.25.0+auto
AS_ROOT ?= 0

GO := $(shell which go)
AIR := $(shell which air)
ifeq ($(AS_ROOT), 1)
	GO := sudo $(GO)
	AIR := sudo $(AIR)
endif
export GOOS ?= $(shell $(GO) env GOOS)
export GOARCH ?= $(shell $(GO) env GOARCH)
export GOPROXY ?= https://proxy.golang.org,direct

MAIN := ./cmd/autobutler/main.go
EXE := ./build/autobutler

UNAME_S := $(shell uname -s)
FLUTTER_VERSION=$(shell grep -Eo 'flutter: (.+)' pubspec.yaml | sed -E 's/^flutter: (.+)$$/\1/')

.PHONY: clean
clean: clean/go clean/flutter ## Clean all build and test artifacts

.PHONY: clean/go
clean/go: ## Clean Go build artifacts
	rm -rf ./build

.PHONY: clean/flutter
clean/flutter: ## Clean flutter project
	flutter clean

.PHONY: setup
setup: setup/gotools setup/air setup/sqlc setup/swag setup/flutter ## Setup development environment

.PHONY: setup/air
setup/air: ## Install air tool
	$(GO) install github.com/air-verse/air@latest

.PHONY: setup/cocoapods
setup/cocoapods: ## Setup CocoaPods for iOS development
ifeq ($(UNAME_S),Darwin)
	brew install ruby
	$$(brew --prefix)/opt/ruby/bin/gem install cocoapods
	echo "Make sure to put $$(gem env gemdir)/bin in your PATH to use the installed cocoapods"
else
	$(error "CocoaPods setup is only supported on macOS")
endif

.PHONY: setup/flutter
setup/flutter: ## Install Flutter tools
	if [ -d "${HOME}/flutter" ]; then
		echo "Flutter already installed at ${HOME}/flutter"
		exit 0
	fi
	if [ -z "$(FLUTTER_VERSION)" ]; then
		echo "Error: Could not determine Flutter version from pubspec.yaml"
		exit 1
	fi
	echo "Installing Flutter version $(FLUTTER_VERSION)"
ifeq ($(UNAME_S),Linux)
	sudo apt-get update -y
	sudo apt-get install -y \
		curl \
		git \
		libglu1-mesa \
		unzip \
		xz-utils \
		zip
	curl --fail -L \
		"https://storage.googleapis.com/flutter_infra_release/releases/stable/linux/flutter_linux_$(FLUTTER_VERSION)-stable.tar.xz" | \
			tar \
				-xf \
				-C "${HOME}"
else ifeq ($(UNAME_S),Darwin)
	rm -f flutter.zip
	set -v
	curl --fail -L \
		"https://storage.googleapis.com/flutter_infra_release/releases/stable/macos/flutter_macos_arm64_$(FLUTTER_VERSION)-stable.zip" \
		-o flutter.zip
	unzip flutter.zip -d "${HOME}"
	rm -f flutter.zip

	echo "Since on Mac, setting up iOS development environment..."
	$(MAKE) setup/ios
else
	$(error "Unsupported OS: $(UNAME_S)")
endif

.PHONY: setup/gotools
setup/gotools: ## Install go tools
	$(GO) install golang.org/x/tools/gopls@latest
	$(GO) install github.com/cweill/gotests/gotests@v1.6.0
	$(GO) install github.com/josharian/impl@v1.4.0
	$(GO) install github.com/haya14busa/goplay/cmd/goplay@v1.0.0
	$(GO) install github.com/go-delve/delve/cmd/dlv@latest
	$(GO) install honnef.co/go/tools/cmd/staticcheck@latest

.PHONY: setup/ios
setup/ios: setup/cocoapods ## Setup iOS development environment
ifeq ($(UNAME_S),Darwin)
	sudo sh -c 'xcode-select -s /Applications/Xcode.app/Contents/Developer && xcodebuild -runFirstLaunch'
	sudo xcodebuild -license
	xcodebuild -downloadPlatform iOS
	sudo softwareupdate --install-rosetta --agree-to-license
else
	$(error "iOS development environment setup is only supported on macOS")
endif

.PHONY: setup/sqlc
setup/sqlc: ## Install sqlc tool
	$(GO) install github.com/sqlc-dev/sqlc/cmd/sqlc@v1.30.0

.PHONY: setup/swag
setup/swag: ## Install swag tool
	$(GO) install github.com/swaggo/swag/cmd/swag@latest

##@ Development

.PHONY: build
build: ## Build web frontend and backend
	$(MAKE) build/frontend/web
	$(MAKE) build/backend

.PHONY: build/backend
build/backend: internal/server/public/stub.txt generate/backend ## Build backend
	mkdir -p ./build
	$(GO) build -o $(EXE) $(MAIN)

internal/server/public/stub.txt: ## Ensure public directory exists for embedding
	mkdir -p ./internal/server/public
	touch ./internal/server/public/stub.txt

.PHONY: build/frontend
build/frontend: ## Build mobile app
ifeq ($(UNAME_S),Linux)
	make build/frontend/android
else ifeq ($(UNAME_S),Darwin)
	make build/frontend/ios
else
	$(error "Unsupported OS: $(UNAME_S)")
endif

.PHONY: build/frontend/android
build/frontend/android: ## Build Android app
	flutter build apk --debug

.PHONY: build/frontend/ios
build/frontend/ios: ## Build iOS app
	flutter build ios --debug --no-codesign

.PHONY: build/frontend/web
build/frontend/web: ## Build web app
	flutter build web --debug

.PHONY: build/lsusb
build/lsusb: ## Build lsusb utility
	$(GO) build -o ./build/lsusb ./cmd/lsusb/main.go

.PHONY: emulate
emulate: ## Emulate mobile device
ifeq ($(UNAME_S),Linux)
	make emulate/android
else ifeq ($(UNAME_S),Darwin)
	make emulate/ios
else
	$(error "Unsupported OS: $(UNAME_S)")
endif

ANDROID_DEVICE_ID ?= Pixel_6
IOS_DEVICE_ID ?= apple_ios_simulator

.PHONY: emulate/android
emulate/android: ## Emulate Android device
	flutter emulators --launch $(ANDROID_DEVICE_ID)

.PHONY: emulate/ios
emulate/ios: ## Emulate iOS device
	flutter emulators --launch $(IOS_DEVICE_ID)

.PHONY: generate
generate: generate/backend generate/frontend ## Generate files

.PHONY: generate/backend
generate/backend: generate/backend/sqlc generate/backend/swagger ## Generate backend files

.PHONY: generate/backend/sqlc
generate/backend/sqlc: ## Generate sqlc files
	sqlc generate

.PHONY: generate/backend/swagger
generate/backend/swagger: ## Generate Swagger docs
	swag init -g ./cmd/autobutler/main.go -o ./docs/swagger --parseInternal

.PHONY: generate/frontend
generate/frontend: generate/frontend/icons ## Generate frontend files

.PHONY: generate/frontend/icons
generate/frontend/icons: ## Generate app icons
	dart run flutter_launcher_icons

.PHONY: serve/backend
serve/backend: generate/backend ## Serve backend
	$(GO) run $(MAIN) serve

.PHONY: serve/frontend
serve/frontend: generate/frontend ## Serve frontend
	echo "Will run app on connected device or emulator..."
	flutter run

PRINT_COVERAGE ?= 0

.PHONY: test
test: test/unit

.PHONY: test/unit
test/unit: test/unit/backend test/unit/frontend ## Run unit tests

.PHONY: test/unit/backend
test/unit/backend: ## Run unit tests for backend
	$(MAKE) _ensure/public
	# Generate coverage report for unit tests
	$(GO) test -v ./... \
		-coverprofile=coverage.out \
		-covermode=atomic
	# Apply coverage ignore directives
	./scripts/apply-coverage-ignore.bash \
		coverage.out
	# Generate coverage report as HTMl
	$(GO) tool cover \
		-html=coverage.out.ignored \
		-o coverage.html
	# Display coverage summary in terminal
	if [[ "$(PRINT_COVERAGE)" = "1" || "$(PRINT_COVERAGE)" = "true" ]] ; then
		$(GO) tool cover \
			-func=coverage.out.ignored
	fi

.PHONY: test/unit/frontend
test/unit/frontend: ## Run unit tests for frontend
	flutter test

.PHONY: tidy
tidy: tidy/flutter tidy/go ## Tidy dependencies

.PHONY: tidy/flutter
tidy/flutter: ## Tidy Flutter dependencies
	flutter pub get

.PHONY: tidy/go
tidy/go: ## Tidy go mod
	$(GO) mod tidy

.PHONY: upgrade
upgrade: upgrade/flutter upgrade/go ## Upgrade dependencies

.PHONY: upgrade/flutter
upgrade/flutter: ## Upgrade Flutter dependencies
	flutter pub upgrade

.PHONY: upgrade/flutter/go
upgrade/go: generate ## Upgrade dependencies (go)
	$(GO) get -u ./...
	$(MAKE) tidy

.PHONY: watch/backend
watch/backend: build/backend ## Watch backend for changes
ifeq ($(AS_ROOT), 1)
	$(AIR) \
		--build.cmd "sudo $(MAKE) build/backend" \
		--build.entrypoint "$(EXE)" \
		--build.args_bin "serve" \
		--build.exclude_dir ".dart_tool,.idea,.ralph,app,build,cd,datalinks,docs,internal/db,scripts,sql"
else
	$(AIR) \
		--build.cmd "$(MAKE) build/backend" \
		--build.entrypoint "$(EXE)" \
		--build.args_bin "serve" \
		--build.exclude_dir ".dart_tool,.idea,.ralph,app,build,cd,datalinks,docs,internal/db,scripts,sql"
endif

.PHONY: watch/frontend
watch/frontend: ## Watch frontend for changes
	echo 'Flutter does not have a built-in watch mode, but you can use "flutter run" to achieve a similar effect. This will run the app and allow you to reload on changes.'
	$(MAKE) serve/frontend

##@ Code quality

.PHONY: check
check: check/format check/lint ## Check code

.PHONY: check/backend
check/backend: check/format/go check/lint/go check/lint/sqlc ## Check backend code

.PHONY: check/frontend
check/frontend: check/format/flutter check/lint/flutter ## Check frontend code

.PHONY: check/flutter
check/flutter: check/format/flutter check/lint/flutter ## Check Flutter/Dart code

.PHONY: check/go
check/go: check/format/go check/lint/go ## Check Go code

.PHONY: check/sqlc
check/sqlc: check/lint/sqlc ## Check sqlc

.PHONY: check/format
check/format: check/format/flutter check/format/go ## Check code formatting

.PHONY: check/format/flutter
check/format/flutter: ## Check Flutter/Dart code formatting
	dart format --set-exit-if-changed .

.PHONY: check/format/go
check/format/go: ## Check Go code formatting
	if [[ -n $$($(GO) fmt ./...) ]]; then
		exit 1
	fi

.PHONY: check/lint/flutter
check/lint/flutter: ## Lint Flutter/Dart code
	flutter analyze

.PHONY: check/lint/go
check/lint/go: internal/server/public/stub.txt ## Check Go code
	$(GO) vet ./...

.PHONY: check/lint/sqlc
check/lint/sqlc: ## Check sqlc
	sqlc vet

.PHONY: fix
fix: fix/flutter fix/go ## Fix code issues

.PHONY: fix/flutter
fix/flutter: format/flutter ## Fix Flutter code issues

.PHONY: fix/go
fix/go: tidy/go format/go ## Fix Go code issues

.PHONY: format
format: format/flutter format/go ## Format code

.PHONY: format/flutter
format/flutter: ## Format Flutter/Dart code
	dart format .

.PHONY: format/go
format/go: ## Format Go code
	gofmt -s -w .

##@ Helpers

.PHONY: version
version: ## Print version
	$(GO) run $(MAIN) version

.PHONY: help
help: ## Displays help info
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m\033[0m\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

env-%: ## Check for env var
	if [ -z "$($*)" ]; then \
		echo "Error: Environment variable '$*' is not set."; \
		exit 1; \
	fi
