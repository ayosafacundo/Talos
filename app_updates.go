package main

import (
	"context"
	"os"
	"strings"

	"Talos/internal/updates"
)

// UpdateEntryView is one row from an update channel (Wails JSON).
type UpdateEntryView struct {
	AppID          string `json:"app_id"`
	Version        string `json:"version"`
	ArtifactURL    string `json:"artifact_url"`
	MinHostVersion string `json:"min_host_version,omitempty"`
	SignatureURL   string `json:"signature_url,omitempty"`
	Name           string `json:"name,omitempty"`
}

// CheckForUpdates fetches a JSON channel (array of entries). Use an HTTPS URL to a static JSON file.
func (a *App) CheckForUpdates(channelURL string) ([]UpdateEntryView, error) {
	channelURL = strings.TrimSpace(channelURL)
	if channelURL == "" {
		return nil, nil
	}
	ctx := context.Background()
	if a.ctx != nil {
		ctx = a.ctx
	}
	rows, err := updates.FetchChannel(ctx, channelURL)
	if err != nil {
		return nil, err
	}
	out := make([]UpdateEntryView, 0, len(rows))
	for _, r := range rows {
		out = append(out, UpdateEntryView{
			AppID:          r.AppID,
			Version:        r.Version,
			ArtifactURL:    r.ArtifactURL,
			MinHostVersion: r.MinHostVersion,
			SignatureURL:   r.SignatureURL,
			Name:           r.Name,
		})
	}
	return out, nil
}

// ApplyUpdateFromArtifactURL downloads and installs a package zip from artifactURL (same pipeline as InstallPackageFromURL).
func (a *App) ApplyUpdateFromArtifactURL(artifactURL string) (string, error) {
	return a.InstallPackageFromURL(artifactURL)
}

// DefaultUpdateChannelURL returns TALOS_UPDATE_CHANNEL env if set (optional operator default).
func (a *App) DefaultUpdateChannelURL() string {
	return strings.TrimSpace(os.Getenv("TALOS_UPDATE_CHANNEL"))
}
