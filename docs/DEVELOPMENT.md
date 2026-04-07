# Development Workflow

This file documents repeatable commands for local development.

## Prerequisites

- Go (current project uses module `go 1.24`)
- Node.js + npm
- Wails CLI (`wails`)
- Protobuf compiler (`protoc`)
- Go protoc plugins:
  - `protoc-gen-go`
  - `protoc-gen-go-grpc`

If a required tool is missing, install it before proceeding.

## Install Dependencies

From repo root:

```bash
go mod tidy
```

From frontend:

```bash
cd frontend
npm install
```

## Regenerate gRPC Stubs

From repo root:

```bash
PATH="$(go env GOPATH)/bin:$PATH" protoc -I . \
  --go_out=. --go_opt=paths=source_relative \
  --go-grpc_out=. --go-grpc_opt=paths=source_relative \
  api/proto/talos/hub/v1/hub.proto
```

Generated files:

- `api/proto/talos/hub/v1/hub.pb.go`
- `api/proto/talos/hub/v1/hub_grpc.pb.go`

## Validate Backend

```bash
go test ./...
go build ./...
```

## Validate Frontend

```bash
cd frontend
npm run build
```

## Run Talos in Dev Mode

From repo root:

```bash
wails dev
```
