package packageinstall

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"archive/zip"
)

const githubUserAgent = "TalosPackageInstaller/1.0"

// InstallFromHTTPURL downloads a .zip and installs it (follows redirects; size-limited).
func InstallFromHTTPURL(ctx context.Context, downloadURL, packagesRoot, hashDir string) (appID string, outDir string, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
	if err != nil {
		return "", "", err
	}
	req.Header.Set("User-Agent", githubUserAgent)
	client := &http.Client{Timeout: 8 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", "", fmt.Errorf("http %d from %s", resp.StatusCode, downloadURL)
	}
	f, err := os.CreateTemp("", "talos-dl-*.zip")
	if err != nil {
		return "", "", err
	}
	tmpPath := f.Name()
	defer func() { _ = os.Remove(tmpPath) }()
	n, err := io.Copy(f, io.LimitReader(resp.Body, MaxZipBytes+1))
	if err != nil {
		_ = f.Close()
		return "", "", err
	}
	if err := f.Close(); err != nil {
		return "", "", err
	}
	if n > MaxZipBytes {
		return "", "", fmt.Errorf("download exceeds size limit")
	}
	r, err := zip.OpenReader(tmpPath)
	if err != nil {
		return "", "", err
	}
	defer r.Close()
	return InstallFromZipReader(&r.Reader, packagesRoot, hashDir)
}

// InstallFromGitHubZipball downloads the GitHub source zipball for ref (branch or tag).
func InstallFromGitHubZipball(ctx context.Context, owner, repo, ref, packagesRoot, hashDir string) (appID string, outDir string, err error) {
	owner = strings.TrimSpace(owner)
	repo = strings.TrimSpace(repo)
	ref = strings.TrimSpace(ref)
	if owner == "" || repo == "" {
		return "", "", fmt.Errorf("owner and repo required")
	}
	if ref == "" {
		ref = "main"
	}
	u := fmt.Sprintf("https://api.github.com/repos/%s/%s/zipball/%s", owner, repo, ref)
	return InstallFromHTTPURL(ctx, u, packagesRoot, hashDir)
}
