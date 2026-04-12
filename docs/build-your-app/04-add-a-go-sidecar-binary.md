# 04 - Add a Go Sidecar Binary

Use this when your app needs background logic, IPC, local file processing, or long-running jobs.

## Step 1: Add Binary to Manifest

In `manifest.yaml`:

```yaml
binary: bin/my-app
```

## Step 2: Build Binary into Package

```bash
mkdir -p "Packages/My App/bin"
go build -o "Packages/My App/bin/my-app" ./examples/tinyapps/my-app
chmod +x "Packages/My App/bin/my-app"
```

## Runtime Environment Variables

Talos injects these when starting your binary:

- `TALOS_APP_ID`
- `TALOS_APP_DATA_DIR`
- `TALOS_HUB_SOCKET`

## Go Bootstrap Skeleton

```go
appID := os.Getenv("TALOS_APP_ID")
socket := os.Getenv("TALOS_HUB_SOCKET")
dataDir := os.Getenv("TALOS_APP_DATA_DIR")
_ = appID
_ = socket
_ = dataDir
```

Use the Go SDK at `sdk/go/talos` for host communication.

