# Talos UI CSS (tokens + utilities)

These files mirror [`Packages/Launchpad/src/talos/`](../../Packages/Launchpad/src/talos/) after a Launchpad build. Copy them into your tiny app `dist/` and link from HTML:

```html
<link rel="stylesheet" href="talos/tokens.css" />
<link rel="stylesheet" href="talos/legacy-alias.css" />
<link rel="stylesheet" href="talos/utilities.css" />
```

Iframe apps do not inherit the host theme; bundle these assets in your package. See `docs/build-your-app/07-talos-ui-and-themes.md`.
