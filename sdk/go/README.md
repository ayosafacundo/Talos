# Talos Go SDK

Go tiny apps can use `sdk/go/talos` to connect to the Talos Hub socket and call:

- `SaveState`
- `LoadState`
- `SendMessage`
- `Broadcast`
- `RequestPermission`

Connection endpoint should come from `TALOS_HUB_SOCKET`.
