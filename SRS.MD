# SRS: Talos App Wrapper

## 1. Project Overview
**Talos** is a high-performance, security-hardened desktop utility dashboard. It serves as a host for "Tiny Apps"—modular, self-contained plugins that provide specialized functionality (calculators, color pickers, etc.) without the overhead or privacy risks of cloud-based solutions.

### Core Principles
* **Hardware Sovereignty:** Everything runs locally; no telemetry or forced cloud sync.
* **Zero-Trust:** Apps are sandboxed by default and require explicit user consent for elevated permissions.
* **Efficiency:** Built with **Go (Wails)** and **Svelte** for minimal RAM/CPU footprint.

---

## 2. Technical Architecture


| Component | Technology | Description |
| :--- | :--- | :--- |
| **Host Core** | Go (Wails) | Manages lifecycle, windowing, and the gRPC hub. |
| **Frontend UI** | Svelte | Powers the Sidebar, Launchpad, and Host overlays. |
| **Plugin IPC** | gRPC (UDS / Named Pipes) | Fast, local-only communication between Host and Tiny App binaries. |
| **Isolation** | Iframes | Ensures 100% CSS/JS separation between apps and the Host. |
| **Persistence** | SQLite / JSON | Host-managed state storage for all plugins. |

---

## 3. UI & Experience (UX)

### 3.1 The "Fixed-Canvas" Layout
* **Fixed Window Size:** Talos maintains a set dimension (e.g., 450x700px) that plugins cannot override.
* **The Launchpad:** A grid-based "App Store" overlay where users discover and launch installed packages from `/Packages`.
* **The Global Toggle:** Configurable in the host settings, defaulting to **`Alt Gr + Tab`**.

### 3.2 The Sidebar (The "Command Center")
* **Permanent Launchpad Icon:** Located at the top to trigger the grid view.
* **Instance Tabs:** Icon-only tabs for every running/hidden app instance.
* **Visual Indicators:** A Discord-style "Active" pill indicates the currently focused app.
* **Customization:** * Users can right-click any tab to assign a **specific background color** (Preset or HEX).
    * This color is saved per-instance (e.g., a "Work" calculator vs. a "Personal" calculator).

---

## 4. Security & Sandboxing

### 4.1 Networking & Communication
* **Air-Gap by Default:** Tiny apps have **zero** internet access unless the user grants an exception via a "First-Run" or "Manifest-Request" prompt.
* **Central Hub Routing:**
    * **1-to-1:** App A sends a message to the Host, which routes it to App B.
    * **Broadcast:** Apps can emit a "Global" signal (e.g., "System Sleep" or "Theme Update") that the Host pushes to all active instances.

### 4.2 Filesystem & Trust
* **Scoped IO:** Apps can only read/write to `/Packages/[AppName]/data/`.
* **Permission Gate:** Any attempt to access external directories triggers a native Talos dialog.
* **Binary Trust:** On the first launch of a new or updated binary, Talos presents a **security warning** requiring user confirmation before execution.

---

## 5. Plugin Lifecycle & Persistence

### 5.1 Package Format
A valid Talos package is a folder containing:
1.  **`manifest.yaml`**: Defines ID, name, icon, binary path, and `multi_instance` support.
2.  **`bin/`**: The self-contained executable.
3.  **`web/`**: The HTML/JS/CSS assets (rendered via Iframe).

### 5.2 State Management
* **Switching:** When a user clicks a different sidebar icon, the previous app's Iframe is hidden but remains active in the background. No data is lost.
* **Session Save:** On closing Talos, the Host sends a gRPC `Save` request to all binaries. Binaries return their current state as a byte-array, which Talos stores locally.
* **Resource Throttling:** If memory is low, Talos serializes the **Least Recently Used (LRU)** app to disk and kills its process. It "thaws" and restores the app automatically when the user clicks its icon again.