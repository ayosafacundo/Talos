# 01 - How Talos Loads Apps

Understanding this flow prevents most setup issues.

## Runtime Flow

1. Talos starts.
2. Host scans `Packages/` for package folders.
3. For each folder, Talos reads `manifest.yaml`.
4. Talos validates `web_entry` exists on disk.
5. Valid packages are added to in-memory package registry.
6. Launchpad asks host for installed apps and renders them.
7. When user launches an app, Talos opens that package `web_entry` in an iframe.
8. If package has `binary`, Talos starts it as a process and injects env vars.

## Discovery Rules

Package discovery is file-system based:

- Root path: `Packages/`
- Manifest filename: exactly `manifest.yaml`
- `web_entry` must point to an existing file
- `id` must be unique across packages

If your app is not showing in Launchpad, the cause is usually:

- missing or invalid `manifest.yaml`
- incorrect `web_entry` path
- `dist/` output not built yet

