SHELL := /bin/bash

GO_BIN := $(shell go env GOPATH)/bin
PROTO_FILE := api/proto/talos/hub/v1/hub.proto

.PHONY: help install-tools deps proto test build verify production-gate launchpad-test sdk-ts-test example-go-app-build example-go-app-clean example-rust-app-build example-rust-app-clean example-ts-app-build example-ts-app-clean app-build dev talos-sync-css integration-hub

help:
	@echo "Available targets:"
	@echo "  install-tools  Install protoc Go plugins"
	@echo "  proto          Regenerate protobuf + gRPC stubs"
	@echo "  deps           npm install Launchpad + isolate npm for Go + go mod tidy"
	@echo "  test           Run Go tests"
	@echo "  build          Build Go packages"
	@echo "  verify         Run all validation checks (Go + Launchpad + TS SDK + production gate)"
	@echo "  production-gate  go test internal/buildmode with -tags=production"
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
	@echo "  app-build        Build full Talos app and demos (wails build -tags=production)"
	@echo "  dev              Run Talos in development mode (proto + frontend-build + wails dev)"
	@echo "  talos-sync-css   Copy Talos UI CSS from Launchpad to sdk/talos/"
	@echo ""
	@echo "Build modes:"
	@echo "  Release: app-build uses -tags=production so development manifest URLs/commands are disabled."
	@echo "  Dev:     run 'make dev' (sets TALOS_DEV_MODE=1) or 'TALOS_DEV_MODE=1 wails dev' for dev iframe URLs."

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

verify: test build launchpad-test sdk-ts-test production-gate

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

example-go-app-build:
	@if [ -d "$(EXAMPLE_GO_SRC)" ]; then \
	  ( cd "$(EXAMPLE_GO_SRC)" && go build -trimpath . ) && \
	  echo "Validated Example Go app source build"; \
	else \
	  echo "make: $(EXAMPLE_GO_SRC) not in tree — skip example-go-app-build"; \
	fi

example-go-app-clean:
	@echo "No generated Go artifacts to clean (wrapper-based runtime)."

example-rust-app-build:
	@if [ -f "$(EXAMPLE_RUST_SRC)/Cargo.toml" ]; then \
	  cargo build --release --manifest-path "$(EXAMPLE_RUST_SRC)/Cargo.toml" && \
	  echo "Validated Example Rust app source build"; \
	else \
	  echo "make: $(EXAMPLE_RUST_SRC) not in tree — skip example-rust-app-build"; \
	fi

example-rust-app-clean:
	@echo "No generated Rust artifacts to clean (wrapper-based runtime)."

example-ts-app-build:
	@if [ -f "$(EXAMPLE_TS_SRC)/package.json" ]; then \
	  npm --prefix "$(EXAMPLE_TS_SRC)" install && npm --prefix "$(EXAMPLE_TS_SRC)" run build; \
	  echo "Built Example TS app under Packages/Example TS App/dist"; \
	else \
	  echo "make: $(EXAMPLE_TS_SRC) not in tree — skip example-ts-app-build"; \
	fi

example-ts-app-clean:
	rm -f "Packages/Example TS App/dist/index.html" "Packages/Example TS App/dist/app.js" "Packages/Example TS App/dist/app.css"

dev: proto frontend-build
	rm -rf Temp/logs
	mkdir -p Temp/logs/packages
	TALOS_DEV_MODE=1 wails dev
