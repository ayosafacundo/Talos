SHELL := /bin/bash

GO_BIN := $(shell go env GOPATH)/bin
PROTO_FILE := api/proto/talos/hub/v1/hub.proto

.PHONY: help install-tools deps proto test build verify verify-core production-gate launchpad-test sdk-ts-test example-go-app-build example-go-app-clean example-rust-app-build example-rust-app-clean example-ts-app-build example-ts-app-clean osrs-ge-ui-build osrs-ge-app-build osrs-ge-app-clean app-build dev talos-sync-css integration-hub

help:
	@echo "Available targets:"
	@echo "  install-tools  Install protoc Go plugins"
	@echo "  proto          Regenerate protobuf + gRPC stubs"
	@echo "  deps           npm install Launchpad + isolate npm for Go + go mod tidy"
	@echo "  test           Run Go tests"
	@echo "  build          Build Go packages"
	@echo "  verify         Run validation checks; if they fail, try frontend-build then fail"
	@echo "  production-gate  go test internal/buildmode (CI sanity for buildmode package)"
	@echo "  launchpad-test Run Launchpad (Vitest) unit tests"
	@echo "  sdk-ts-test    Run Vitest for sdk/ts"
	@echo "  integration-hub Run hub gRPC integration tests"
	@echo "  frontend-build Build Launchpad (Vite) + sync sdk/talos CSS"
	@echo "  example-go-app-build Validate Example Go app source build"
	@echo "  example-go-app-clean Remove Example Go app binary"
	@echo "  example-rust-app-build Validate Example Rust app source build"
	@echo "  example-rust-app-clean Remove Example Rust app binary"
	@echo "  example-ts-app-build Build Example TypeScript app web assets"
	@echo "  example-ts-app-clean Remove Example TypeScript app generated assets"
	@echo "  osrs-ge-ui-build   Build OSRS GE Vite bundle only (updates dist/ for /talos-pkg/)"
	@echo "  osrs-ge-app-build  Build OSRS GE mini-app (Go binary + Vite UI)"
	@echo "  osrs-ge-app-clean  Remove OSRS GE app binary"
	@echo "  app-build      Build full Talos app and demos (wails build -tags=production)"
	@echo "  dev            Run Talos in development mode (proto + frontend-build + OSRS GE Go binary + UI + wails dev)"
	@echo "  talos-sync-css Copy Talos UI CSS from Launchpad to sdk/talos/"
	@echo ""
	@echo "Build modes:"
	@echo "  Release: make app-build produces one binary; use Launchpad Settings → Developer to enable manifest dev commands, or TALOS_DEV_MODE=1."
	@echo "  Dev:     make dev (frontend + wails dev); optional TALOS_DEV_MODE=1 matches production dev behavior."

install-tools:
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

deps:
	npm --prefix Packages/Launchpad install
	bash scripts/ensure-npm-go-modules.sh
	go mod tidy

proto:
	PATH="$(GO_BIN):$$PATH" protoc -I . \
		--go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		$(PROTO_FILE)

test:
	go test ./...

build:
	go build ./...

verify-core: test build launchpad-test sdk-ts-test production-gate

verify:
	@set -e; \
	if $(MAKE) verify-core; then \
	  exit 0; \
	fi; \
	echo "verify failed; attempting frontend-build before exiting..."; \
	$(MAKE) frontend-build || true; \
	exit 1

sdk-ts-test:
	npm --prefix sdk/ts install
	npm --prefix sdk/ts test

# Ensures release-tagged buildmode always disables development surfaces.
production-gate:
	go test -tags=production ./internal/buildmode -count=1

frontend-build: proto Packages/Launchpad
	npm --prefix Packages/Launchpad install
	bash scripts/ensure-npm-go-modules.sh
	npm --prefix Packages/Launchpad run build
	$(MAKE) talos-sync-css
	go mod tidy

talos-sync-css:
	@mkdir -p sdk/talos
	cp -f Packages/Launchpad/src/talos/tokens.css Packages/Launchpad/src/talos/legacy-alias.css Packages/Launchpad/src/talos/utilities.css sdk/talos/

app-build: proto frontend-build verify
	make example-go-app-build
	make example-rust-app-build
	make example-ts-app-build
	make osrs-ge-app-build
	wails build -tags=production

launchpad-test:
	npm --prefix Packages/Launchpad install
	bash scripts/ensure-npm-go-modules.sh
	npm --prefix Packages/Launchpad test -- --run

integration-hub:
	go test ./internal/hub/ -tags=integration -count=1 -v

# Example mini apps (one per SDK language)
EXAMPLE_GO_SRC := Packages/Example Go App/src
EXAMPLE_RUST_SRC := Packages/Example Rust App
EXAMPLE_TS_SRC := Packages/Example TS App
OSRS_GE_SRC := Packages/OSRS-GE-App/src
OSRS_GE_UI := Packages/OSRS-GE-App/ui

example-go-app-build:
	@if [ -d "$(EXAMPLE_GO_SRC)" ]; then \
	  mkdir -p "Packages/Example Go App/bin" && \
	  rm -f "Packages/Example Go App/bin/example-go-app" && \
	  ( cd "$(EXAMPLE_GO_SRC)" && go build -trimpath -o ../bin/example-go-app . ) && \
	  echo "Validated Example Go app source build"; \
	else \
	  echo "make: $(EXAMPLE_GO_SRC) not in tree — skip example-go-app-build"; \
	fi

example-go-app-clean:
	rm -f "Packages/Example Go App/bin/example-go-app"

example-rust-app-build:
	@if [ -f "$(EXAMPLE_RUST_SRC)/Cargo.toml" ]; then \
	  cargo build --release --manifest-path "$(EXAMPLE_RUST_SRC)/Cargo.toml" && \
	  mkdir -p "$(EXAMPLE_RUST_SRC)/bin" && \
	  rm -f "$(EXAMPLE_RUST_SRC)/bin/example-rust-app" && \
	  if [ -f "$(EXAMPLE_RUST_SRC)/target/release/example-rust-app.exe" ]; then \
	    cp -f "$(EXAMPLE_RUST_SRC)/target/release/example-rust-app.exe" "$(EXAMPLE_RUST_SRC)/bin/example-rust-app"; \
	  else \
	    cp -f "$(EXAMPLE_RUST_SRC)/target/release/example-rust-app" "$(EXAMPLE_RUST_SRC)/bin/example-rust-app"; \
	  fi && \
	  echo "Validated Example Rust app source build"; \
	else \
	  echo "make: $(EXAMPLE_RUST_SRC) not in tree — skip example-rust-app-build"; \
	fi

example-rust-app-clean:
	rm -f "Packages/Example Rust App/bin/example-rust-app"

example-ts-app-build:
	@if [ -f "$(EXAMPLE_TS_SRC)/package.json" ]; then \
	  npm --prefix "$(EXAMPLE_TS_SRC)" install && npm --prefix "$(EXAMPLE_TS_SRC)" run build; \
	  echo "Built Example TS app under Packages/Example TS App/dist"; \
	else \
	  echo "make: $(EXAMPLE_TS_SRC) not in tree — skip example-ts-app-build"; \
	fi

example-ts-app-clean:
	rm -f "Packages/Example TS App/dist/index.html" "Packages/Example TS App/dist/app.js" "Packages/Example TS App/dist/app.css"

# Vite production bundle only (updates Packages/OSRS-GE-App/dist for /talos-pkg/ iframe fallback).
osrs-ge-ui-build:
	@if [ -f "$(OSRS_GE_UI)/package.json" ]; then \
	  npm --prefix "$(OSRS_GE_UI)" install && npm --prefix "$(OSRS_GE_UI)" run build && \
	  echo "Built OSRS GE UI dist for Talos iframe fallback"; \
	else \
	  echo "make: $(OSRS_GE_UI) not in tree — skip osrs-ge-ui-build"; \
	fi

osrs-ge-app-build:
	@if [ -d "$(OSRS_GE_SRC)" ]; then \
	  mkdir -p "Packages/OSRS-GE-App/bin" && \
	  rm -f "Packages/OSRS-GE-App/bin/osrs-ge-app" && \
	  ( cd "$(OSRS_GE_SRC)" && go build -trimpath -o ../bin/osrs-ge-app . ) && \
	  echo "Built OSRS GE Go binary"; \
	else \
	  echo "make: $(OSRS_GE_SRC) not in tree — skip osrs-ge-app-build"; \
	fi
	@$(MAKE) osrs-ge-ui-build

osrs-ge-app-clean:
	rm -f "Packages/OSRS-GE-App/bin/osrs-ge-app"
	rm -f "Packages/OSRS-GE-App/dist/index.html" "Packages/OSRS-GE-App/dist/app.js" "Packages/OSRS-GE-App/dist/app.css"

dev: proto frontend-build osrs-ge-app-build
	rm -rf Temp/logs
	mkdir -p Temp/logs/packages
	TALOS_DEV_MODE=1 wails dev
