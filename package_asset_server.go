package main

import (
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

const talosPackageURLPrefix = "/talos-pkg/"

// Longer-term (bridge audit): optionally reverse-proxy mini-app dev servers through this same
// Wails asset origin so embedded WebViews never open a second loopback origin (avoids HMR
// WebSockets + strict postMessage target skew in dev). Production already uses /talos-pkg/.

// talosPackageMiddleware serves files from packagesRoot for GET /talos-pkg/<package-dir>/...
// so mini-app iframes share the Wails asset origin (file:// iframes are blocked from wails://).
func talosPackageMiddleware(packagesRoot string) assetserver.Middleware {
	absRoot, err := filepath.Abs(packagesRoot)
	if err != nil {
		absRoot = packagesRoot
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet || !strings.HasPrefix(r.URL.Path, talosPackageURLPrefix) {
				next.ServeHTTP(w, r)
				return
			}
			rel := strings.TrimPrefix(r.URL.Path, talosPackageURLPrefix)
			rel = strings.TrimPrefix(path.Clean("/"+rel), "/")
			if rel == ".." || strings.HasPrefix(rel, "../") {
				log.Printf("talos-pkg: invalid relative path url=%q rel=%q", r.URL.Path, rel)
				http.Error(w, "invalid path", http.StatusBadRequest)
				return
			}
			fsPath := filepath.Join(absRoot, filepath.FromSlash(rel))
			absFile, err := filepath.Abs(fsPath)
			if err != nil {
				log.Printf("talos-pkg: abs failed url=%q err=%v", r.URL.Path, err)
				http.Error(w, "invalid path", http.StatusBadRequest)
				return
			}
			if !strings.HasPrefix(absFile, absRoot+string(filepath.Separator)) && absFile != absRoot {
				log.Printf("talos-pkg: path outside root url=%q absFile=%q root=%q", r.URL.Path, absFile, absRoot)
				http.Error(w, "invalid path", http.StatusForbidden)
				return
			}
			fi, err := os.Stat(absFile)
			if err != nil {
				if os.IsNotExist(err) {
					log.Printf("talos-pkg: not found url=%q rel=%q", r.URL.Path, rel)
					http.NotFound(w, r)
					return
				}
				log.Printf("talos-pkg: stat error url=%q err=%v", r.URL.Path, err)
				http.Error(w, "internal error", http.StatusInternalServerError)
				return
			}
			if fi.IsDir() {
				log.Printf("talos-pkg: directory listing denied url=%q", r.URL.Path)
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}
			// ServeContent avoids net/http.ServeFile's redirect when the path ends in index.html
			// (301 → …/dist/), which embedded WebKit iframes often cannot follow.
			f, err := os.Open(absFile)
			if err != nil {
				log.Printf("talos-pkg: open failed url=%q err=%v", r.URL.Path, err)
				http.Error(w, "internal error", http.StatusInternalServerError)
				return
			}
			defer f.Close()
			http.ServeContent(w, r, filepath.Base(absFile), fi.ModTime(), f)
		})
	}
}
