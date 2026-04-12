# 03 - Build a Web App Package

This is the simplest app type: web-only app, no binary process.

## Step 1: Create Package Folder

```bash
mkdir -p "Packages/My App/dist" "Packages/My App/data"
```

## Step 2: Add Manifest

Create `Packages/My App/manifest.yaml`:

```yaml
id: app.my.app
name: My App
web_entry: dist/index.html
multi_instance: false
```

## Step 3: Add Web Entry

Create `Packages/My App/dist/index.html`:

```html
<!doctype html>
<html>
  <body>
    <h1>My App</h1>
    <script>
      console.log("My app started inside Talos iframe");
    </script>
  </body>
</html>
```

## Step 4: Run Talos

```bash
make dev
```

Your app should appear in Launchpad Installed list.

## Step 5: Iteration

Modify files in `dist/` and relaunch app from Launchpad.  
Talos package events trigger refresh behavior for changed app frontends.

