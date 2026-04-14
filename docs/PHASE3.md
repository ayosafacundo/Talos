# Phase 3 — distribution, trust, and updates

Operator-focused notes for package integrity, signing, catalogs, and updates.

## Environment variables

| Variable | Purpose |
|----------|---------|
| `TALOS_CATALOG_URL` | HTTPS URL returning JSON array of catalog entries (`id`, `name`, optional `install_url`, `source`). Powers **Repositories** in Launchpad when set. |
| `TALOS_UPDATE_CHANNEL` | Default JSON URL for **Settings → About → Check for updates** (array of `{app_id, version, artifact_url, ...}`). |
| `TALOS_PARANOID_TRUST` | Set to `1` to block opening packages whose on-disk files fail hash verification (`trust_status=tampered`). |

## Package hashes

After install, SHA-256 manifests are stored under `Temp/package_hashes/<app_id>.json`. Launchpad shows a **trust** hint per app (`ok`, `unsigned`, `tampered`, `signed_ok`, …).

## Ed25519 signatures (optional)

1. Place publisher **public** keys as `Temp/trusted_keys/<name>.pub` (32 raw bytes or 64-char hex). Use **ImportTrustedPublisherKey** from the host API or copy files manually.
2. Add a detached signature file beside the package: **`.talos-signature`** (64-byte raw or 128-char hex) signing the canonical JSON of the hash manifest (same content as `Temp/package_hashes/<id>.json`).

If a signature is present but no trusted key verifies it, status is `signed_invalid`. If hashes match but there is no signature file, status is `unsigned`.

## Catalog JSON example

```json
[
  {
    "id": "app.demo",
    "name": "Demo",
    "install_url": "https://releases.example.com/demo.zip",
    "source": "example"
  }
]
```

## Update channel JSON example

```json
[
  {
    "app_id": "app.demo",
    "version": "1.1.0",
    "artifact_url": "https://releases.example.com/demo-1.1.0.zip",
    "min_host_version": "0.1.0"
  }
]
```

Use **ApplyUpdateFromArtifactURL** / **InstallPackageFromURL** in the UI or host to apply (same secure install path as manual zip URL).
