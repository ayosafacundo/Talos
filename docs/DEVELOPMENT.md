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

**CI parity:** `make verify` runs Go tests, `go build ./...`, Launchpad Vitest, **`npm --prefix sdk/ts test`**, and the `internal/buildmode` test (see `make production-gate`). Hub socket integration tests are `bash scripts/run_integration_hub.sh` (also run in CI after verify).

**Development mode vs release binary:** `make app-build` ships a `-tags=production` binary. Manifest `development` behavior is **off by default** per `Packages/<dir>` folder. Enable it only for specific folders in **Launchpad → Settings → Development mode**. Release binaries **ignore** `TALOS_DEV_MODE`; from a source tree, `make dev` sets `TALOS_DEV_MODE=1` so non-production builds honor that env for SDK/backend diagnostics (all packages).

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

## Bundled example apps (optional)

Packages under `Packages/` are built by their authors; the root `Makefile` only builds Talos + Launchpad. To compile the **in-repo** examples manually (from repo root):

**Important (npm and scoped data)**

- Run the commands below from the **repository root**, or use the **in-package** variants that only use `--prefix ui` / `--prefix .` so npm does not create a spurious `Packages/Example … App/` tree inside an example folder.
- Talos stores app files under `Packages/<Example>/data/` via the SDK. Do not prefix paths with `data/` in API calls (that would nest `data/data/…`).

**Example Go App**

```bash
npm --prefix "Packages/Example Go App/ui" install
npm --prefix "Packages/Example Go App/ui" run build
mkdir -p "Packages/Example Go App/bin"
rm -f "Packages/Example Go App/bin/example-go-app"
( cd "Packages/Example Go App/src" && go build -trimpath -o ../bin/example-go-app . )
```

*(From `Packages/Example Go App/`: `npm --prefix ui install && npm --prefix ui run build`.)*

**Example Rust App**

```bash
npm --prefix "Packages/Example Rust App/ui" install
npm --prefix "Packages/Example Rust App/ui" run build
cargo build --release --manifest-path "Packages/Example Rust App/Cargo.toml"
mkdir -p "Packages/Example Rust App/bin"
# Copy from target/release/ to bin/ (adjust for .exe on Windows)
cp -f "Packages/Example Rust App/target/release/example-rust-app" "Packages/Example Rust App/bin/example-rust-app"
```

*(From `Packages/Example Rust App/`: `npm --prefix ui install && npm --prefix ui run build`.)*

**Example TS App**

```bash
npm --prefix "Packages/Example TS App" install
npm --prefix "Packages/Example TS App" run build
```

*(From `Packages/Example TS App/`: `npm install && npm run build`.)*

## Build full Talos app

Proto, frontend, verify, and production Wails binary (no other `Packages/*` builds):

```bash
make app-build
```

## Run Talos in Dev Mode

From repo root:

```bash
make dev
```
