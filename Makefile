SHELL := /bin/bash

GO_BIN := $(shell go env GOPATH)/bin
PROTO_FILE := api/proto/talos/hub/v1/hub.proto

.PHONY: help install-tools deps proto test build frontend-build verify tiny-demo-build tiny-demo-clean tiny-ts-demo-build tiny-ts-demo-clean app-build dev

help:
	@echo "Available targets:"
	@echo "  install-tools  Install protoc Go plugins"
	@echo "  proto          Regenerate protobuf + gRPC stubs"
	@echo "  test           Run Go tests"
	@echo "  build          Build Go packages"
	@echo "  frontend-build Build frontend assets"
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
	npm --prefix frontend install
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

frontend-build:
	npm --prefix frontend run build

verify: test build frontend-build

tiny-demo-build:
	mkdir -p "Packages/Tiny Go Demo/bin"
	go build -o "Packages/Tiny Go Demo/bin/tiny-go-demo" ./examples/tinyapps/go-demo
	chmod +x "Packages/Tiny Go Demo/bin/tiny-go-demo"

tiny-demo-clean:
	rm -f "Packages/Tiny Go Demo/bin/tiny-go-demo"

tiny-ts-demo-build:
	npm --prefix examples/tinyapps/ts-demo install
	npm --prefix examples/tinyapps/ts-demo run build

tiny-ts-demo-clean:
	rm -f "Packages/Tiny TS Demo/web/main.js"

app-build: proto tiny-demo-build tiny-ts-demo-build verify
	wails build

dev: proto tiny-demo-build tiny-ts-demo-build
	wails dev
