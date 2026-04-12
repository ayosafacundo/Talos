SHELL := /bin/bash

GO_BIN := $(shell go env GOPATH)/bin
PROTO_FILE := api/proto/talos/hub/v1/hub.proto

.PHONY: help install-tools deps proto test build verify tiny-demo-build tiny-demo-clean tiny-ts-demo-build tiny-ts-demo-clean app-build dev

help:
	@echo "Available targets:"
	@echo "  install-tools  Install protoc Go plugins"
	@echo "  proto          Regenerate protobuf + gRPC stubs"
	@echo "  test           Run Go tests"
	@echo "  build          Build Go packages"
	@echo "  verify         Run all validation checks"
	@echo "  tiny-demo-build Build Go Tiny Demo package binary"
	@echo "  tiny-demo-clean Remove Go Tiny Demo built binary"
	@echo "  tiny-ts-demo-build Build TypeScript Tiny Demo web assets"
	@echo "  tiny-ts-demo-clean Remove TypeScript Tiny Demo generated app.js"
	@echo "  app-build        Build full Talos app and demos"
	@echo "  dev              Run Talos in development mode"

install-tools:
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

deps:
	go mod tidy
	npm --prefix Packages/Launchpad install
	npm --prefix examples/tinyapps/ts-demo install

proto:
	PATH="$(GO_BIN):$$PATH" protoc -I . \
		--go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		$(PROTO_FILE)

test:
	go test ./...

build:
	go build ./...

verify: test build

frontend-build: proto Packages/Launchpad
	npm --prefix Packages/Launchpad install
	npm --prefix Packages/Launchpad run build


app-build: proto frontend-build verify
	wails build

dev: proto frontend-build
	rm -rf Temp/logs
	mkdir -p Temp/logs/packages
	TALOS_DEV_MODE=1 wails dev
