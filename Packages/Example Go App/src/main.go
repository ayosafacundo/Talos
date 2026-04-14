package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"Talos/sdk/go/talos"
)

func main() {
	appID := os.Getenv("TALOS_APP_ID")
	hubSocket := os.Getenv("TALOS_HUB_SOCKET")
	if appID == "" || hubSocket == "" {
		fmt.Println("example-go-app: TALOS_APP_ID or TALOS_HUB_SOCKET not set")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	client, err := talos.Dial(ctx, hubSocket)
	if err != nil {
		fmt.Printf("example-go-app: dial failed: %v\n", err)
		return
	}
	defer client.Close()

	if prev, found, err := client.LoadState(ctx, appID); err == nil && found {
		fmt.Printf("example-go-app: previous state=%s\n", string(prev))
	}
	state := []byte(time.Now().UTC().Format(time.RFC3339))
	if err := client.SaveState(ctx, appID, state); err != nil {
		fmt.Printf("example-go-app: save state failed: %v\n", err)
	}

	granted, message, err := client.RequestPermission(ctx, appID, "net:internet", "Example Go app network check")
	if err != nil {
		fmt.Printf("example-go-app: permission request failed: %v\n", err)
	} else {
		fmt.Printf("example-go-app: permission granted=%t message=%s\n", granted, message)
	}

	if err := client.WriteScopedFile(ctx, appID, "example-go-app/heartbeat.txt", []byte("ok\n")); err != nil {
		fmt.Printf("example-go-app: scoped write failed: %v\n", err)
	}

	if recipients, err := client.Broadcast(ctx, appID, "app:example:hello", []byte("hello from go")); err == nil {
		fmt.Printf("example-go-app: broadcast recipients=%d\n", recipients)
	}
}
