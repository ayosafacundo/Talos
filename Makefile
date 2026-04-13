SHELL := /bin/bash

GO_BIN := $(shell go env GOPATH)/bin
PROTO_FILE := api/proto/talos/hub/v1/hub.proto

.PHONY: help install-tools deps proto test build verify production-gate launchpad-test tiny-demo-build tiny-demo-clean tiny-ts-demo-build tiny-ts-demo-clean app-build dev talos-sync-css integration-hub

help:
	@echo "Available targets:"
	@echo "  install-tools  Install protoc Go plugins"
	@echo "  proto          Regenerate protobuf + gRPC stubs"
	@echo "  deps           npm install Launchpad + isolate npm for Go + go mod tidy"
	@echo "  test           Run Go tests"
	@echo "  build          Build Go packages"
	@echo "  verify         Run all validation checks (Go + Launchpad + production gate)"
	@echo "  production-gate  go test internal/buildmode with -tags=production"
	@echo "  launchpad-test Run Launchpad (Vitest) unit tests"
	@echo "  integration-hub Run hub gRPC integration tests"
	@echo "  frontend-build Build Launchpad (Vite) + sync sdk/talos CSS"
	@echo "  tiny-demo-build Build Go Tiny Demo binary (needs examples/tinyapps/go-demo)"
	@echo "  tiny-demo-clean Remove Go Tiny Demo built binary"
	@echo "  tiny-ts-demo-build Build TS Tiny Demo web assets (needs examples/tinyapps/ts-demo)"
	@echo "  tiny-ts-demo-clean Remove TypeScript Tiny Demo generated app.js"
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

verify: test build launchpad-test production-gate

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

# Tiny app demos (sources live under examples/tinyapps/* when present; see docs/TINY_APP_INIT.md)
TINY_GO_SRC := examples/tinyapps/go-demo
TINY_GO_OUT := Packages/Tiny Go Demo/bin/tiny-go-demo
TINY_TS_SRC := examples/tinyapps/ts-demo

tiny-demo-build:
	@if [ -d "$(TINY_GO_SRC)" ]; then \
	  mkdir -p "Packages/Tiny Go Demo/bin" && \
	  ( cd "$(TINY_GO_SRC)" && go build -trimpath -o "$(CURDIR)/$(TINY_GO_OUT)" . ); \
	  echo "Built $(TINY_GO_OUT)"; \
	else \
	  echo "make: $(TINY_GO_SRC) not in tree — skip tiny-demo-build"; \
	fi

tiny-demo-clean:
	rm -f "$(TINY_GO_OUT)"

tiny-ts-demo-build:
	@if [ -f "$(TINY_TS_SRC)/package.json" ]; then \
	  npm --prefix "$(TINY_TS_SRC)" install && npm --prefix "$(TINY_TS_SRC)" run build; \
	  echo "Built Tiny TS Demo under Packages/Tiny TS Demo/dist"; \
	else \
	  echo "make: $(TINY_TS_SRC) not in tree — skip tiny-ts-demo-build"; \
	fi

tiny-ts-demo-clean:
	rm -f "Packages/Tiny TS Demo/dist/app.js"

dev: proto frontend-build
	rm -rf Temp/logs
	mkdir -p Temp/logs/packages
	TALOS_DEV_MODE=1 wails dev
