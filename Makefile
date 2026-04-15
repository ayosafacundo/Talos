SHELL := /bin/bash

GO_BIN := $(shell go env GOPATH)/bin
PROTO_FILE := api/proto/talos/hub/v1/hub.proto

.PHONY: help install-tools deps proto test build verify verify-core production-gate launchpad-test sdk-ts-test app-build dev talos-sync-css integration-hub

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
	@echo "  app-build      Build Talos (wails build -tags=production); no Packages/* demos"
	@echo "  dev            proto + frontend-build + wails dev (TALOS_DEV_MODE=1)"
	@echo "  talos-sync-css Copy Talos UI CSS from Launchpad to sdk/talos/"
	@echo ""
	@echo "Build modes:"
	@echo "  Release: make app-build produces one binary; use Launchpad Settings → Development mode (per package folder) for manifest dev commands."
	@echo "  Dev:     make dev sets TALOS_DEV_MODE=1 for source-tree SDK/backend diagnostics (ignored in release builds)."
	@echo "  Other packages under Packages/ are built by their authors (see docs)."

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
	wails build -tags=production

launchpad-test:
	npm --prefix Packages/Launchpad install
	bash scripts/ensure-npm-go-modules.sh
	npm --prefix Packages/Launchpad test -- --run

integration-hub:
	go test ./internal/hub/ -tags=integration -count=1 -v

dev: proto frontend-build
	rm -rf Temp/logs
	mkdir -p Temp/logs/packages
	TALOS_DEV_MODE=1 wails dev
