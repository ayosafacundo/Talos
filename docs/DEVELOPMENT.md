# Development Workflow

This file documents repeatable commands for local development.

For the complete end-to-end reference, read `docs/DEVELOPMENT_FULL.md`.

## Prerequisites

- Go (current project uses module `go 1.24`)
- Node.js + npm
- Wails CLI (`wails`)
- Protobuf compiler (`protoc`)
- Go protoc plugins:
  - `protoc-gen-go`
  - `protoc-gen-go-grpc`

If a required tool is missing, ask the user to install it before proceeding.

## Install Dependencies

From repo root, use `make deps` (runs `npm install` for Launchpad, [`scripts/ensure-npm-go-modules.sh`](../scripts/ensure-npm-go-modules.sh) to keep each `Packages/**/node_modules` tree as a nested Go module, then `go mod tidy`). That way third-party `.go` files inside npm packages are not part of the `Talos` module.

If you run `go mod tidy` yourself, run `npm --prefix Packages/Launchpad install` and `bash scripts/ensure-npm-go-modules.sh` first whenever `node_modules` changes.

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

Or use:

```bash
make proto
```

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

## One-Command Verification

From repo root:

```bash
make verify
```

## SDK Locations

- Go SDK: `sdk/go/talos`
- TS SDK baseline: `sdk/ts`
- Rust SDK baseline: `sdk/rust`
- SDK usage guide: `docs/SDK_GUIDE.md`
- Tiny app initialization guide: `docs/TINY_APP_INIT.md`

## Runtime Artifacts

- Persisted permission grants: `Temp/permissions.json`

## Demo Tiny App

Build bundled Go demo tiny app:

```bash
make tiny-demo-build
```

Build bundled TypeScript iframe demo app:

```bash
make tiny-ts-demo-build
```

Clean built binary:

```bash
make tiny-demo-clean
```

Clean TypeScript generated web file:

```bash
make tiny-ts-demo-clean
```

## Build Full App

Build everything (proto, demos, verify, Wails package):

```bash
make app-build
```

## Run Talos in Dev Mode

From repo root:

```bash
make dev
```
