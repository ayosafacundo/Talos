# 02 - Package Layout and Manifest

## Recommended Layout

```text
Packages/My App/
├── manifest.yaml
├── dist/
│   └── index.html
├── data/
│   └── .gitkeep
└── bin/
    └── my-app-binary   (optional)
```

## Minimal `manifest.yaml`

```yaml
id: app.my.app
name: My App
web_entry: dist/index.html
multi_instance: false
```

## Full Example

```yaml
id: app.my.app
name: My App
version: "0.1.0"
icon: dist/icon.png # png, gif, jpg, webp, svg are supported
binary: bin/my-app-binary
web_entry: dist/index.html
permissions:
  - fs:external
  - net:internet
multi_instance: false
```

## Field Notes

- `id`: unique package id, stable across versions.
- `name`: display name in Launchpad.
- `web_entry`: relative path to iframe entry HTML.
- `binary`: optional relative path to executable sidecar.
- `permissions`: requested capabilities (host policy still controls grants).
- `multi_instance`: whether multiple UI instances are allowed.

## Validation Constraints

- Paths must be relative (not absolute).
- `web_entry` defaults to `dist/index.html` if omitted.
- `web_entry` file must exist or discovery rejects package.

