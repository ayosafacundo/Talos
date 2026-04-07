## Role
You are a Senior Systems Architect and Lead Developer specializing in **Go (Wails)**, **Svelte**, and **High-Performance IPC (gRPC)**. Your task is to architect and implement "Talos," a secure, local-first app wrapper for "Tiny Apps". The feature list and details will be in a SRS.MD file on the root of this project.

## Project Context: Talos
Talos is a fixed-size desktop dashboard (450x700px) that hosts modular, self-contained packages. It prioritizes **Hardware Sovereignty**, **Air-gapped Security**, and **Zero-Trust** execution.

### Tech Stack
- **Backend:** Go (Wails v2/v3).
- **Frontend:** Svelte (for the Host UI).
- **Isolation:** Iframes for Tiny App rendering.
- **Communication:** gRPC over Unix Domain Sockets (macOS/Linux) and Named Pipes (Windows) — No TCP/IP.
- **Package Config:** YAML manifests.

---

## Phase 1: Core Architecture & Scaffolding
1. **Project Init:** Scaffold a Wails project with a Svelte frontend.
2. **Package Discovery:** Implement a Go service using `fsnotify` to watch a `/Packages` directory. It must parse `manifest.yaml` files and emit events to the frontend.
3. **Process Manager:** Create a lifecycle manager in Go to spawn/kill self-contained binaries found in the `/Packages/[App]/bin` folder.
4. **The gRPC Hub (Option B):** - Implement a central gRPC server in the Go Host.
   - Use **Unix Domain Sockets** and **Named Pipes** for transport.
   - All inter-app communication must be routed through this Hub (Request/Response and Broadcast).

## Phase 2: The "Talos Framework" (Developer SDK)
Create a standardized "Talos SDK" that Tiny App developers will use.
1. **Go/Rust/TS Client Libraries:** Provide a lightweight gRPC client wrapper for:
   - `SaveState(data []byte)` / `LoadState()`.
   - `SendMessage(targetID, payload)`.
   - `RequestPermission(scope)`.
2. **FS Scoping:** Enforce that all binary IO is restricted to `/Packages/[AppName]/data/` unless a permission is granted.
3. **The Iframe Bridge:** Set up a Wails-to-Iframe message bus so the Svelte host can pass window events or state updates to the sandboxed frontend.

## Phase 3: Frontend & UX
1. **The Sidebar:** Build a Discord-style vertical sidebar.
   - Top icon: Static Launchpad.
   - Dynamic icons: One per active app instance.
   - **Feature:** Right-click context menu to set custom background colors per tab (persisted in local DB).
2. **The Launchpad:** A grid overlay showing all discovered apps.
3. **State Management:** Implement the "Hidden but Active" Iframe logic. Switching apps must not destroy the Iframe or the process.
4. **Global Toggle:** Implement a global hotkey listener for **`Alt Gr + Tab`** to show/hide the Talos window.

## Phase 4: Security & Persistence
1. **Permission UI:** Create a Svelte component for "First-Run" binary trust warnings and filesystem/internet permission requests.
2. **LRU Freezing:** Implement logic to monitor memory. If a threshold is hit, serialize the state of the Least Recently Used app via gRPC, kill the process, and "thaw" it only when re-focused.
3. **The "Big Save":** On Talos exit, orchestrate a global `SaveState` broadcast to all active binaries and store the returned byte arrays in a local SQLite database.

## Phase 5: Documentation Generation
Generate the following files in the `/docs` folder:
1. **`ARCHITECTURE.md`**: Explaining the gRPC Hub, the socket-based IPC, and the security model.
2. **`PLUGIN_GUIDE.md`**: A step-by-step tutorial for developers to build a "Talos-ready" app using the `manifest.yaml` and the gRPC SDK.
3. **`MANIFEST_SPEC.md`**: Detailed documentation of all YAML fields (id, name, permissions, multi_instance, etc.).

---

## Instructions for Implementation
- **Efficiency:** Minimize allocations. Use `Svelte 5` runes for state if available.
- **Security:** Ensure no app can "sniff" the socket path of another app; they must only know the Hub's path.
- **Code Style:** Clean, documented Go code with idiomatic error handling.