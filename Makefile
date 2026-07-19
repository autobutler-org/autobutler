SHELL := /usr/bin/env
.SHELLFLAGS = bash -e -c
.DEFAULT_GOAL := help
.ONESHELL:
.SILENT:

ifneq (,$(wildcard ./.env))
    include .env
    export
endif

GO := $(shell which go)
AIR := $(shell which air)
export GOOS ?= $(shell $(GO) env GOOS)
export GOARCH ?= $(shell $(GO) env GOARCH)
export GOPROXY ?= https://proxy.golang.org,direct

ENTRYPOINT := ./cmd/autobutler
EXE := ./build/autobutler

# Auto-detect Chromium-based browser for Flutter web if CHROME_EXECUTABLE
# is not already set.  Brave is checked first because users who removed
# Chrome in favour of Brave still need a Chromium engine for Flutter.
ifndef CHROME_EXECUTABLE
  ifneq (,$(shell test -f '/Applications/Brave Browser.app/Contents/MacOS/Brave Browser' && echo found))
    export CHROME_EXECUTABLE := /Applications/Brave Browser.app/Contents/MacOS/Brave Browser
  endif
endif

UNAME_S := $(shell uname -s)
UNAME_M := $(shell uname -m)
FLUTTER_VERSION=$(shell grep -Eo 'flutter: (.+)' pubspec.yaml | sed -E 's/^flutter: (.+)$$/\1/')
GO_MOD_VERSION := $(shell awk '/^go /{print $$2; exit}' go.mod)
export GOTOOLCHAIN=go$(GO_MOD_VERSION)

.PHONY: clean
clean: clean/go clean/flutter ## Clean all build and test artifacts

.PHONY: clean/go
clean/go: ## Clean Go build artifacts
	rm -rf ./build

.PHONY: clean/flutter
clean/flutter: ## Clean flutter project
	flutter clean

.PHONY: setup
setup: setup/gotools setup/air setup/sqlc setup/swag setup/flutter setup/hooks ## Setup development environment

.PHONY: setup/wrk
setup/wrk: ## Install wrk load-testing CLI
ifeq ($(UNAME_S),Linux)
	sudo apt-get update -y
	sudo apt-get install -y wrk
else ifeq ($(UNAME_S),Darwin)
	brew install wrk
else
	$(error "Unsupported OS: $(UNAME_S)")
endif

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
ifeq ($(UNAME_M),aarch64)
	# The official Flutter Linux tarball ships with an x86_64 Dart SDK only.
	# On ARM64 (e.g. Raspberry Pi), we download the tarball for Flutter's tooling
	# and directory structure, then swap in the native ARM64 Dart SDK from apt.
	curl --fail -L \
		"https://storage.googleapis.com/flutter_infra_release/releases/stable/linux/flutter_linux_$(FLUTTER_VERSION)-stable.tar.xz" \
		-o /tmp/flutter.tar.xz
	tar -xf /tmp/flutter.tar.xz -C "${HOME}"
	rm -f /tmp/flutter.tar.xz
	# Install ARM64 Dart SDK
	curl -fsSL https://dl-ssl.google.com/linux/linux_signing_key.pub | sudo gpg --dearmor -o /usr/share/keyrings/dart.gpg
	echo 'deb [signed-by=/usr/share/keyrings/dart.gpg arch=arm64] https://storage.googleapis.com/download.dartlang.org/linux/debian stable main' | sudo tee /etc/apt/sources.list.d/dart_stable.list
	sudo apt-get update -y
	sudo apt-get install -y dart
	# Replace the x86_64 Dart SDK bundled with Flutter with the native ARM64 one
	rsync -a /usr/lib/dart/ "${HOME}/flutter/bin/cache/dart-sdk/"
	# Remove the pre-compiled x86_64 flutter_tools snapshot so Flutter rebuilds it
	# with the ARM64 Dart SDK on first run
	rm -f "${HOME}/flutter/bin/cache/flutter_tools.snapshot" \
		  "${HOME}/flutter/bin/cache/flutter_tools.stamp"
else
	curl --fail -L \
		"https://storage.googleapis.com/flutter_infra_release/releases/stable/linux/flutter_linux_$(FLUTTER_VERSION)-stable.tar.xz" | \
			tar \
				-xJf - \
				-C "${HOME}"
endif
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
	$(GO) install golang.org/x/vuln/cmd/govulncheck@latest

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

.PHONY: setup/hooks
setup/hooks: ## Install git hooks
	ln -sf "$(PWD)/git/hooks/pre-commit" .git/hooks/pre-commit
	echo "✅ Git hooks installed"

.PHONY: setup/node
setup/node: ## Install and use the Node.js version from .nvmrc via nvm
	if [ ! -s "$$HOME/.nvm/nvm.sh" ]; then
		echo "nvm is not installed. See https://github.com/nvm-sh/nvm#installing-and-updating"
		exit 1
	fi
	. "$$HOME/.nvm/nvm.sh" && nvm install && nvm use

##@ Development

.PHONY: build
build: ## Build web frontend and backend
	$(MAKE) build/frontend/web
	$(MAKE) build/backend

.PHONY: build/backend
build/backend: internal/server/public/stub.txt generate/backend ## Build backend
	mkdir -p ./build
	$(GO) build -o $(EXE) $(ENTRYPOINT)

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

FLUTTER_BUILD_MODE ?= debug

.PHONY: build/frontend/android
build/frontend/android: ## Build Android app
	flutter build apk --$(FLUTTER_BUILD_MODE)

.PHONY: build/frontend/ios
build/frontend/ios: ## Build iOS app
	flutter build ios --$(FLUTTER_BUILD_MODE) --no-codesign

.PHONY: build/frontend/web
build/frontend/web: internal/server/public/stub.txt generate/frontend/sbom ## Build web app
	flutter build web --$(FLUTTER_BUILD_MODE)
	cp -R ./build/web/. ./internal/server/public/

.PHONY: build/provisioning
build/provisioning: ## Build provisioning service
	mkdir -p ./build
	$(GO) build -o ./build/autobutler-provisioning ./cmd/provisioning/

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
generate/frontend: generate/frontend/icons generate/frontend/autobutler-icons generate/frontend/sbom ## Generate frontend files

.PHONY: generate/frontend/icons
generate/frontend/icons: ## Generate app icons
	dart run flutter_launcher_icons

.PHONY: generate/frontend/autobutler-icons
generate/frontend/autobutler-icons: ## Regenerate AutobutlerIcons.ttf from SVGs using fantasticon
	npx fantasticon packages/autobutler_icons/svgs \
		--output packages/autobutler_icons/fonts \
		--font-types ttf \
		--name AutobutlerIcons \
		--config packages/autobutler_icons/.fantasticonrc.json \
		--normalize

.PHONY: generate/frontend/sbom
generate/frontend/sbom: ## Generate Flutter SBOM asset from pubspec.lock
	dart run scripts/generate_flutter_sbom.dart

DEPLOY_HOST ?= autobutler
DEPLOY_PATH ?= ~/autobutler

.PHONY: remote-deploy
remote-deploy: build ## Build and deploy to a remote host via scp, then run the binary
	scp $(EXE) $(DEPLOY_HOST):$(DEPLOY_PATH)
	ssh $(DEPLOY_HOST) "pkill -f '$(DEPLOY_PATH) serve' || true; nohup $(DEPLOY_PATH) serve > ~/autobutler.log 2>&1 &"

.PHONY: drive
drive: ## Create and mount a new MyDrive DMG (auto-numbered: MyDrive, MyDrive2, MyDrive3, …)
	@if ! test -d /Volumes/MyDrive; then \
		N=""; \
	else \
		i=2; \
		while test -d /Volumes/MyDrive$$i; do i=$$((i+1)); done; \
		N=$$i; \
	fi; \
	NAME="MyDrive$$N"; \
	FILE="$$HOME/Desktop/mydrive$$N.dmg"; \
	echo "Creating $$FILE (volume: $$NAME)..."; \
	hdiutil create -size 100m -fs HFS+ -volname "$$NAME" "$$FILE"; \
	hdiutil attach "$$FILE"

.PHONY: unmount-drive
unmount-drive: ## Detach the highest-numbered MyDrive volume currently mounted
	@highest_num=-1; \
	highest_name=""; \
	for name in $$(ls /Volumes/ 2>/dev/null | grep -E '^MyDrive[0-9]*$$'); do \
		num=$$(echo "$$name" | sed 's/MyDrive//'); \
		if [ -z "$$num" ]; then num=0; fi; \
		if [ "$$num" -gt "$$highest_num" ]; then \
			highest_num=$$num; \
			highest_name=$$name; \
		fi; \
	done; \
	if [ -z "$$highest_name" ]; then \
		echo "No MyDrive volumes mounted."; \
		exit 1; \
	fi; \
	echo "Detaching /Volumes/$$highest_name..."; \
	hdiutil detach "/Volumes/$$highest_name"

.PHONY: serve/backend
serve/backend: generate/backend ## Serve backend
	AUTOBUTLER_INSECURE=true $(GO) run $(ENTRYPOINT) serve

.PHONY: serve/backend/secure
serve/backend/secure: generate/backend ## Serve backend with TLS enabled
	$(GO) run $(ENTRYPOINT) serve

.PHONY: serve/frontend
serve/frontend: serve/frontend/web ## Serve frontend

.PHONY: serve/frontend/mobile
serve/frontend/mobile: generate/frontend ## Serve mobile frontend
	flutter run

.PHONY: serve/frontend/web
serve/frontend/web: generate/frontend ## Serve web frontend
	flutter run \
		-d web-server

PRINT_COVERAGE ?= 0

.PHONY: test
test: test/unit

PERF_PORT ?= 8080
PERF_BASE_URL ?= http://127.0.0.1:$(PERF_PORT)
PERF_SUMMARY_WRK_DIRS ?= test-results/performance
PERF_FIXTURE_TARGET_DIR ?= $(HOME)/autobutler/data/cirrus

.PHONY: test/perf/generate-files
test/perf/generate-files: ## Generate file fixtures under a target Cirrus directory for performance testing
	bash ./test/performance/generate_files.sh "$(PERF_FIXTURE_TARGET_DIR)"

.PHONY: test/perf/load
test/perf/load: build/backend ## Run local wrk load profile against a temporary local backend
	mkdir -p test-results/performance
	$(MAKE) test/perf/generate-files PERF_FIXTURE_TARGET_DIR="$(PERF_FIXTURE_TARGET_DIR)"
	PORT=$(PERF_PORT) AUTOBUTLER_INSECURE=true ./build/autobutler serve > test-results/performance/server-load.log 2>&1 &
	SERVER_PID=$$!
	trap 'kill $$SERVER_PID 2>/dev/null || true' EXIT
	export AUTOBUTLER_BASE_URL=$(PERF_BASE_URL)
	export PERF_FIXTURE_TARGET_DIR="$(PERF_FIXTURE_TARGET_DIR)"
	export TEST_DURATION_THREADS=2
	export TEST_DURATION_CONCURRENCY=15
	export TEST_DURATION_DURATION=10s
	export TEST_UPLOAD_CONCURRENCY=4
	export TEST_UPLOAD_COUNT=8
	./test/performance/test.sh

.PHONY: test/perf/stress
test/perf/stress: build/backend ## Run local wrk stress profile against a temporary local backend
	mkdir -p test-results/performance
	$(MAKE) test/perf/generate-files PERF_FIXTURE_TARGET_DIR="$(PERF_FIXTURE_TARGET_DIR)"
	PORT=$(PERF_PORT) AUTOBUTLER_INSECURE=true ./build/autobutler serve > test-results/performance/server-stress.log 2>&1 &
	SERVER_PID=$$!
	trap 'kill $$SERVER_PID 2>/dev/null || true' EXIT
	export AUTOBUTLER_BASE_URL=$(PERF_BASE_URL)
	export PERF_FIXTURE_TARGET_DIR="$(PERF_FIXTURE_TARGET_DIR)"
	export TEST_DURATION_THREADS=4
	export TEST_DURATION_CONCURRENCY=50
	export TEST_DURATION_DURATION=30s
	export TEST_UPLOAD_CONCURRENCY=10
	export TEST_UPLOAD_COUNT=20
	./test/performance/test.sh

.PHONY: test/perf/summary
test/perf/summary: ## Render the Markdown performance summary
	python3 ./test/performance/render_summary.py \
		$(foreach dir,$(PERF_SUMMARY_WRK_DIRS),--wrk-dir $(dir))

.PHONY: test/unit
test/unit: test/unit/backend test/unit/frontend ## Run unit tests

.PHONY: test/unit/backend
test/unit/backend: internal/server/public/stub.txt ## Run unit tests for backend
	# Generate coverage report for unit tests (excludes integration test packages)
	$(GO) test -v $(shell $(GO) list ./... | grep -v '/internal/server/api/v0/') \
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
test/unit/frontend: generate/frontend ## Run unit tests for frontend
	echo "Testing Autobutler frontend..."
	flutter test
	for pkg in packages/*/; do
		if [ -f "$$pkg/pubspec.yaml" ] && [ -d "$$pkg/test" ]; then
			echo "Testing $$pkg..."
			$(MAKE) -C "$$pkg" test/unit || exit 1
		fi
	done

.PHONY: test/integration
test/integration: test/integration/backend ## Run integration tests

.PHONY: test/integration/backend
test/integration/backend: internal/server/public/stub.txt ## Run backend integration tests (requires real filesystem, spins up gin engine)
	$(GO) test -v ./internal/server/api/v0/...

.PHONY: coverage
coverage: test/unit/backend ## Run backend tests and print coverage percentage
	$(GO) tool cover \
		-func=coverage.out.ignored | tail -1

.PHONY: tidy
tidy: tidy/flutter tidy/go ## Tidy dependencies

.PHONY: tidy/flutter
tidy/flutter: ## Tidy Flutter dependencies
	flutter pub get
	for pkg in packages/*/; do
		if [ -f "$$pkg/pubspec.yaml" ] && [ -d "$$pkg/test" ]; then
			echo "Downloading packages for $$pkg..."
			$(MAKE) -C "$$pkg" tidy || exit 1
		fi
	done
ifeq ($(UNAME_S),Darwin)
	cd ios && pod install
endif

.PHONY: tidy/go
tidy/go: ## Tidy go mod
	$(GO) mod tidy

.PHONY: upgrade
upgrade: upgrade/flutter upgrade/go ## Upgrade dependencies

.PHONY: upgrade/flutter
upgrade/flutter: ## Upgrade Flutter dependencies
	flutter pub upgrade
	$(MAKE) tidy/flutter
	$(MAKE) generate/frontend

.PHONY: upgrade/go
upgrade/go: generate/backend ## Upgrade dependencies (go)
	$(GO) get -u ./...
	$(MAKE) tidy/go

.PHONY: watch/backend
watch/backend: build/backend ## Watch backend for changes
	$(AIR)

.PHONY: watch/frontend
watch/frontend: generate/frontend ## Watch frontend on web
	echo "Defaulting to web since it supports hot reload..."
	flutter run -d chrome

##@ Code quality

.PHONY: check
check: check/backend check/frontend ## Check code

.PHONY: check/backend
check/backend: generate/backend check/format/go check/lint/go check/lint/sqlc ## Check backend code

.PHONY: check/frontend
check/frontend: check/format/flutter check/lint/flutter ## Check frontend code

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

.PHONY: check/lint
check/lint: check/lint/flutter check/lint/go check/lint/sqlc ## Check code quality

.PHONY: check/lint/flutter
check/lint/flutter: generate/frontend/icons generate/frontend/sbom ## Lint Flutter/Dart code
	flutter analyze

.PHONY: check/lint/go
check/lint/go: internal/server/public/stub.txt ## Check Go code
	$(GO) vet ./...

.PHONY: check/vuln
check/vuln: check/vuln/backend ## Check for known CVEs

.PHONY: check/vuln/backend
check/vuln/backend: ## Check Go module for known CVEs (govulncheck)
	govulncheck ./...

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

##@ Release

.PHONY: release/yank
release/yank: ## Yank a release: remove from Azure + mark GitHub release as pre-release (VERSION=v0.X.Y)
	if [ -z "$(VERSION)" ]; then echo "Error: VERSION is required. Usage: make release/yank VERSION=v0.X.Y"; exit 1; fi
	echo "Yanking $(VERSION) from Azure Blob Storage..."
	az storage blob delete-batch \
		--account-name autobutlerrelease \
		--source releases \
		--pattern "autobutler/$(VERSION)/*"
	echo "Marking $(VERSION) as pre-release on GitHub..."
	gh release edit $(VERSION) --prerelease --repo autobutler-org/autobutler
	echo "✅ $(VERSION) yanked. Ship a patch release ASAP."

##@ Helpers

.PHONY: version
version: ## Print version
	$(GO) run $(ENTRYPOINT) version

.PHONY: help
help: ## Displays help info
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m\033[0m\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

env-%: ## Check for env var
	if [ -z "$($*)" ]; then \
		echo "Error: Environment variable '$*' is not set."; \
		exit 1; \
	fi

# ── Azure deployment ────────────────────────────────────────────────────────

## render/headscale: Embed setup-headscale.bash into ARM parameters file.
## Usage: make render/headscale HEADSCALE_DOMAIN=network.autobutler.org ADMIN_EMAIL=admin.autobutler.org
## Output: deploy/azure/headscale.rendered.parameters.json (gitignored)

HEADSCALE_DOMAIN ?= network.autobutler.org

deploy/azure/headscale.rendered.parameters.json: env-HEADSCALE_DOMAIN ## Render ARM parameters file for headscale deployment
	bash deploy/azure/render.bash
.PHONY: render/headscale
render/headscale: deploy/azure/headscale.rendered.parameters.json ## Render ARM parameters file for headscale deployment (alias)

SSH_KEY_PATH ?= ~/.ssh/id_autobutler-headscale.pub

.PHONY: deploy/headscale
deploy/headscale: deploy/azure/headscale.rendered.parameters.json
	az deployment group create \
	    --resource-group autobutler-headscale \
	    --template-file ./deploy/azure/headscale.json \
	    --parameters ./$< \
	    --parameters adminPublicKey="$$(cat $(SSH_KEY_PATH))"
