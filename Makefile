SHELL := bash
.SHELLFLAGS = -e -c
.DEFAULT_GOAL := help
.ONESHELL:
.SILENT:

# .ONESHELL needs GNU Make 3.82+. macOS ships 3.81, where it is silently ignored and
# every recipe line runs in its own shell -- multi-line `if` blocks then die with
# "syntax error: unexpected end of file", which points nowhere near the real problem.
MIN_MAKE := 3.82
ifneq ($(firstword $(sort $(MAKE_VERSION) $(MIN_MAKE))),$(MIN_MAKE))
$(error GNU Make $(MAKE_VERSION) is too old; this Makefile needs $(MIN_MAKE)+. \
On macOS run `brew install make` and use `gmake`, or put \
"$$(brew --prefix)/opt/make/libexec/gnubin" first on PATH.)
endif

ifneq (,$(wildcard ./.env))
    include .env
    export
endif

GO := $(shell which go)
AIR := $(shell which air)
# Only ask Go for its defaults when Go is actually installed. On a macOS CI runner
# without it, $(GO) is empty and the shell call becomes `env GOOS`, which floods the
# log with "env: GOOS: No such file or directory" on every recipe.
ifneq ($(GO),)
export GOOS ?= $(shell $(GO) env GOOS)
export GOARCH ?= $(shell $(GO) env GOARCH)
endif
export GOPROXY ?= https://proxy.golang.org,direct

ENTRYPOINT := ./cmd/quark
EXE := ./build/quark

# The app version comes from the most recent git tag, so a build can never claim a
# version that was never released. pubspec.yaml deliberately has no `version:` field --
# it was a second place to edit and drifted from the tags it was meant to track.
# Override with BUILD_NAME=X.Y.Z.
BUILD_NAME ?= $(shell git describe --tags --abbrev=0 2>/dev/null | sed -E 's/^v//')

# Dev runs get no tag: `flutter run` stamps no --build-name, on purpose -- a dirty
# working tree must not report itself as a released version. The commit identifies it
# instead, passed as a Dart compile-time constant so it reaches web the same as mobile
# (version.json is a release-build artifact; --build-number is an integer on Android).
# Seven characters to match what the Quark's own commit is shortened to in Settings.
GIT_SHA ?= $(shell git rev-parse --short=7 HEAD 2>/dev/null)
FLUTTER_RUN_DEFINES := $(if $(GIT_SHA),--dart-define=GIT_SHA=$(GIT_SHA),)

# AS_ROOT=1 runs the backend targets under sudo. Needed for USB device mounting
# on Linux, and for binding the privileged :443 port that the secure targets use.
# The env assignment is placed after sudo (via `env`) rather than before it, so
# it survives sudo's environment scrubbing without depending on `sudo -E` being
# permitted by sudoers.
ifeq ($(AS_ROOT),1)
SUDO := sudo
else
SUDO :=
endif

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

# Only the commit, never Semver: NOSEMVER is what makes CompareVersions return 2,
# which is the guard that stops autoupdate installing over a dev binary. Stamping a
# version here would arm it (see internal/server/autoupdate.go).
#
# Every way of starting the backend needs this, `go run` most of all: it records no
# vcs.revision at all, so a `serve/backend` process has nothing to fall back on and
# reports NOCOMMIT without it.
GO_LDFLAGS := $(if $(GIT_SHA),-ldflags "-X github.com/autobutler-org/quark/pkg/util/versionutil.GitCommit=$(GIT_SHA)",)

.PHONY: build/backend
build/backend: internal/server/public/stub.txt generate/backend ## Build backend
	mkdir -p ./build
	$(GO) build $(GO_LDFLAGS) -o $(EXE) $(ENTRYPOINT)

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
build/frontend/android: generate/frontend/sbom ## Build Android app
	flutter build apk --$(FLUTTER_BUILD_MODE) $(if $(BUILD_NAME),--build-name=$(BUILD_NAME),)

.PHONY: build/frontend/ios
build/frontend/ios: generate/frontend/sbom ## Build iOS app
	flutter build ios --$(FLUTTER_BUILD_MODE) --no-codesign \
		$(if $(BUILD_NAME),--build-name=$(BUILD_NAME),)

# App Store Connect distribution. A build number must be unique and strictly increasing
# within a marketing version, and App Store Connect never frees one once consumed. So
# ask it what it already has rather than guess: any counter or clock can collide or run
# backwards between trigger paths, and a consumed number can never be reclaimed.
#
# Set IOS_BUILD_NUMBER=N to skip the query -- builds that never upload (CI verification,
# local testing) should, since the query needs App Store Connect credentials.
IOS_EXPORT_OPTIONS ?= ios/ExportOptions.plist
IOS_IPA_DIR ?= build/ios/ipa

# The bare `export` at the top of this file (active whenever .env exists) pushes every
# make variable into recipe environments, FLUTTER_BUILD_MODE ?= debug included. Flutter's
# xcode_backend reads FLUTTER_BUILD_MODE before falling back to CONFIGURATION, so an
# archive would emit debug artifacts while xcodebuild really did run -configuration
# Release -- producing an IPA that installs from TestFlight and then refuses to launch.
.PHONY: build/frontend/ios/ipa
build/frontend/ios/ipa: FLUTTER_BUILD_MODE := release
build/frontend/ios/ipa: check/frontend/ios/release ## Build a signed iOS IPA for the App Store
	# Xcode open on this workspace can contend with the CLI archive over DerivedData and
	# produce "accessing build database ... disk I/O error". Warn, do not block.
	if pgrep -qx Xcode >/dev/null; then
		echo "Warning: Xcode is running. Quit it if the archive fails on DerivedData I/O."
	fi
	# Start from a clean slate so no stale intermediate can be reused.
	# Set IOS_SKIP_CLEAN=1 when iterating on signing or export settings.
	if [ -z "$(IOS_SKIP_CLEAN)" ]; then
		flutter clean
	fi
	flutter pub get
	# assets/sbom_flutter.json is declared in pubspec.yaml but generated, not committed
	# (see .gitignore). Without it the archive dies late in
	# release_ios_bundle_flutter_assets with "Failed to bundle asset files".
	$(MAKE) generate/frontend/sbom
	if [ -z "$(BUILD_NAME)" ]; then
		echo "Error: no git tag found to derive the app version from."
		echo "  The version comes from the most recent tag, for example v0.31.1."
		echo "  Fix: git fetch --tags   (CI needs fetch-tags on actions/checkout)"
		echo "       or pass BUILD_NAME=X.Y.Z explicitly."
		exit 1
	fi
	if [ -n "$(IOS_BUILD_NUMBER)" ]; then
		build_number="$(IOS_BUILD_NUMBER)"
	else
		# Diagnostics go to stderr, so this captures just the number.
		build_number="$$(scripts/ios-next-build-number.bash --version "$(BUILD_NAME)")"
	fi
	echo "Building $(BUILD_NAME) ($$build_number)"
	flutter build ipa \
		--release \
		--export-options-plist=$(IOS_EXPORT_OPTIONS) \
		--build-name=$(BUILD_NAME) \
		--build-number="$$build_number"
	ipa="$$(ls -t $(IOS_IPA_DIR)/*.ipa 2>/dev/null | head -1)"
	if [ -z "$$ipa" ]; then
		echo "Error: no IPA was produced in $(IOS_IPA_DIR)."
		echo "Check the xcodebuild output above; a signing failure is the usual cause."
		echo "Run 'make check/frontend/ios/release' to verify signing prerequisites."
		exit 1
	fi
	$(MAKE) check/frontend/ios/ipa IOS_IPA="$$ipa"
	echo "Built $$ipa"
	echo "Next: make publish/frontend/ios"

.PHONY: check/frontend/ios/ipa
check/frontend/ios/ipa: ## Verify an IPA is a real release build (IOS_IPA=path)
	ipa="$(IOS_IPA)"
	if [ -z "$$ipa" ]; then
		ipa="$$(ls -t $(IOS_IPA_DIR)/*.ipa 2>/dev/null | head -1)"
	fi
	if [ -z "$$ipa" ] || [ ! -f "$$ipa" ]; then
		echo "Error: no IPA to check. Run 'make build/frontend/ios/ipa' first."
		exit 1
	fi
	# A debug Flutter build ships a JIT kernel blob and a stub App binary. Such a build
	# uploads and installs fine, then refuses to launch from the home screen or TestFlight.
	if unzip -l "$$ipa" | grep -q "kernel_blob.bin"; then
		echo "Error: $$ipa is a DEBUG build (contains flutter_assets/kernel_blob.bin)."
		echo "  It would install from TestFlight and then refuse to launch."
		echo "  Cause: FLUTTER_BUILD_MODE leaked into the environment as 'debug', or a"
		echo "         stale debug intermediate was reused. Check: make --eval='p:; @env |"
		echo "         grep FLUTTER_BUILD_MODE' p"
		exit 1
	fi
	if ! unzip -l "$$ipa" | grep -q "App.framework/App"; then
		echo "Error: $$ipa has no App.framework/App binary."
		exit 1
	fi
	echo "OK: $$ipa is a release build."

.PHONY: check/frontend/ios/release
check/frontend/ios/release: ## Check iOS App Store release prerequisites
	failed=0
	if [ "$(UNAME_S)" != "Darwin" ]; then
		echo "Error: iOS release builds require macOS (found $(UNAME_S))."
		exit 1
	fi
	if ! xcode-select -p >/dev/null 2>&1; then
		echo "Missing: Xcode command line tools. Run 'make setup/ios' first."
		failed=1
	fi
	if [ ! -f "$(IOS_EXPORT_OPTIONS)" ]; then
		echo "Missing: $(IOS_EXPORT_OPTIONS)."
		failed=1
	fi
	if ! security find-identity -v -p codesigning 2>/dev/null | grep -q "Apple Distribution"; then
		echo "Missing: an 'Apple Distribution' signing identity in your keychain."
		echo "  Fix: open Xcode > Settings > Accounts, sign in with the Apple Developer"
		echo "       account for team 4NK7MWUA57, then 'Manage Certificates' > '+' >"
		echo "       'Apple Distribution'."
		failed=1
	fi
	if [ ! -d "$$HOME/Library/MobileDevice/Provisioning Profiles" ]; then
		echo "Warning: no provisioning profiles installed yet. Xcode will fetch an App Store"
		echo "         profile automatically on the first archive if the bundle ID"
		echo "         org.autobutler.quark is registered in App Store Connect."
	fi
	if [ $$failed -ne 0 ]; then
		echo
		echo "iOS release prerequisites are not satisfied."
		exit 1
	fi
	echo "iOS release prerequisites OK."

.PHONY: publish/frontend/ios
publish/frontend/ios: ## Upload the iOS IPA to App Store Connect
	ipa="$$(ls -t $(IOS_IPA_DIR)/*.ipa 2>/dev/null | head -1)"
	if [ -z "$$ipa" ]; then
		echo "Error: no IPA found in $(IOS_IPA_DIR). Run 'make build/frontend/ios/ipa' first."
		exit 1
	fi
	if [ -z "$$APP_STORE_CONNECT_KEY_ID" ] || [ -z "$$APP_STORE_CONNECT_ISSUER_ID" ]; then
		echo "Error: APP_STORE_CONNECT_KEY_ID and APP_STORE_CONNECT_ISSUER_ID must be set."
		echo "  Create an API key at https://appstoreconnect.apple.com/access/integrations/api"
		echo "  with the 'App Manager' role, then place the downloaded .p8 at:"
		echo "    ~/.appstoreconnect/private_keys/AuthKey_<KEY_ID>.p8"
		echo "  and export both variables (or add them to .env)."
		exit 1
	fi
	xcrun altool --upload-app \
		--type ios \
		--file "$$ipa" \
		--apiKey "$$APP_STORE_CONNECT_KEY_ID" \
		--apiIssuer "$$APP_STORE_CONNECT_ISSUER_ID"
	echo "Uploaded $$ipa to App Store Connect."
	echo "Next: the build appears under TestFlight after processing (usually 5-30 minutes)."

.PHONY: build/frontend/web
build/frontend/web: internal/server/public/stub.txt generate/frontend/sbom ## Build web app
	flutter build web --$(FLUTTER_BUILD_MODE) $(if $(BUILD_NAME),--build-name=$(BUILD_NAME),)
	cp -R ./build/web/. ./internal/server/public/

.PHONY: build/provisioning
build/provisioning: ## Build provisioning service
	mkdir -p ./build
	$(GO) build -o ./build/quark-provisioning ./cmd/provisioning/

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
	swag init -g ./cmd/quark/main.go -o ./docs/swagger --parseInternal

.PHONY: generate/frontend
generate/frontend: generate/frontend/icons generate/frontend/quark-icons generate/frontend/sbom ## Generate frontend files

.PHONY: generate/frontend/icons
generate/frontend/icons: ## Generate app icons
	dart run flutter_launcher_icons

.PHONY: generate/frontend/quark-icons
generate/frontend/quark-icons: ## Regenerate QuarkIcons.ttf from SVGs using fantasticon
	npx fantasticon packages/quark_icons/svgs \
		--output packages/quark_icons/fonts \
		--font-types ttf \
		--name QuarkIcons \
		--config packages/quark_icons/.fantasticonrc.json \
		--normalize

.PHONY: generate/frontend/sbom
generate/frontend/sbom: ## Generate Flutter SBOM asset from pubspec.lock
	dart run scripts/generate_flutter_sbom.dart

DEPLOY_HOST ?= quark
DEPLOY_PATH ?= ~/quark

.PHONY: remote-deploy
remote-deploy: build ## Build and deploy to a remote host via scp, then run the binary
	scp $(EXE) $(DEPLOY_HOST):$(DEPLOY_PATH)
	ssh $(DEPLOY_HOST) "pkill -f '$(DEPLOY_PATH) serve' || true; nohup $(DEPLOY_PATH) serve > ~/quark.log 2>&1 &"

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
serve/backend: generate/backend ## Serve backend over plain HTTP on :8080 (insecure)
	$(SUDO) env QUARK_INSECURE=true $(GO) run $(GO_LDFLAGS) $(ENTRYPOINT) serve

.PHONY: serve/backend/secure
serve/backend/secure: generate/backend ## Serve backend over HTTPS on :443 (self-signed)
	$(SUDO) $(GO) run $(GO_LDFLAGS) $(ENTRYPOINT) serve

.PHONY: serve/frontend
serve/frontend: serve/frontend/web ## Serve frontend

.PHONY: serve/frontend/mobile
serve/frontend/mobile: generate/frontend ## Serve mobile frontend
	flutter run $(FLUTTER_RUN_DEFINES)

.PHONY: serve/frontend/web
serve/frontend/web: generate/frontend ## Serve web frontend
	flutter run \
		-d web-server \
		$(FLUTTER_RUN_DEFINES)

PRINT_COVERAGE ?= 0

.PHONY: test
test: test/unit

PERF_PORT ?= 8080
PERF_BASE_URL ?= http://127.0.0.1:$(PERF_PORT)
PERF_SUMMARY_WRK_DIRS ?= test-results/performance
PERF_FIXTURE_TARGET_DIR ?= $(HOME)/quark/data/files

.PHONY: test/perf/generate-files
test/perf/generate-files: ## Generate file fixtures under a target files directory for performance testing
	bash ./test/performance/generate_files.sh "$(PERF_FIXTURE_TARGET_DIR)"

.PHONY: test/perf/load
test/perf/load: build/backend ## Run local wrk load profile against a temporary local backend
	mkdir -p test-results/performance
	$(MAKE) test/perf/generate-files PERF_FIXTURE_TARGET_DIR="$(PERF_FIXTURE_TARGET_DIR)"
	PORT=$(PERF_PORT) QUARK_INSECURE=true ./build/quark serve > test-results/performance/server-load.log 2>&1 &
	SERVER_PID=$$!
	trap 'kill $$SERVER_PID 2>/dev/null || true' EXIT
	export QUARK_BASE_URL=$(PERF_BASE_URL)
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
	PORT=$(PERF_PORT) QUARK_INSECURE=true ./build/quark serve > test-results/performance/server-stress.log 2>&1 &
	SERVER_PID=$$!
	trap 'kill $$SERVER_PID 2>/dev/null || true' EXIT
	export QUARK_BASE_URL=$(PERF_BASE_URL)
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
	echo "Testing Quark frontend..."
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
watch/backend: build/backend ## Watch backend for changes, plain HTTP on :8080 (insecure)
	$(SUDO) env QUARK_INSECURE=true $(AIR)

.PHONY: watch/backend/secure
watch/backend/secure: build/backend ## Watch backend for changes, HTTPS on :443 (self-signed)
	$(SUDO) $(AIR)

.PHONY: watch/frontend
watch/frontend: generate/frontend ## Watch frontend on web
	echo "Defaulting to web since it supports hot reload..."
	flutter run -d chrome $(FLUTTER_RUN_DEFINES)

##@ Code quality

.PHONY: check
check: check/backend check/frontend check/spelling ## Check code

.PHONY: check/backend
check/backend: generate/backend check/format/go check/lint/go check/lint/sqlc ## Check backend code

.PHONY: check/frontend
check/frontend: check/format/flutter check/lint/flutter ## Check frontend code

.PHONY: check/spelling
check/spelling: ## Check spelling in code and docs
	npm run check:spelling

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
		--account-name quarkrelease \
		--source releases \
		--pattern "quark/$(VERSION)/*"
	echo "Marking $(VERSION) as pre-release on GitHub..."
	gh release edit $(VERSION) --prerelease --repo autobutler-org/quark
	echo "✅ $(VERSION) yanked. Ship a patch release ASAP."

##@ Helpers

.PHONY: version
version: ## Print version
	$(GO) run $(GO_LDFLAGS) $(ENTRYPOINT) version

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
## Usage: make render/headscale HEADSCALE_DOMAIN=network.quark.org ADMIN_EMAIL=admin.quark.org
## Output: deploy/azure/headscale.rendered.parameters.json (gitignored)

HEADSCALE_DOMAIN ?= network.quark.org

deploy/azure/headscale.rendered.parameters.json: env-HEADSCALE_DOMAIN ## Render ARM parameters file for headscale deployment
	bash deploy/azure/render.bash
.PHONY: render/headscale
render/headscale: deploy/azure/headscale.rendered.parameters.json ## Render ARM parameters file for headscale deployment (alias)

SSH_KEY_PATH ?= ~/.ssh/id_quark-headscale.pub

.PHONY: deploy/headscale
deploy/headscale: deploy/azure/headscale.rendered.parameters.json
	az deployment group create \
	    --resource-group quark-headscale \
	    --template-file ./deploy/azure/headscale.json \
	    --parameters ./$< \
	    --parameters adminPublicKey="$$(cat $(SSH_KEY_PATH))"
