package manifest

import (
	"errors"
	"fmt"
	"net/url"
	"path/filepath"
	"strings"
)

// Development configures optional iframe dev-server loading (honored only when host allows development mode).
type Development struct {
	// Command is argv for the dev server (e.g. ["npm", "run", "dev"]). Executed with Dir=package root.
	Command []string `yaml:"command,omitempty" json:"command,omitempty"`
	// URL is the dev server base URL the iframe loads (e.g. http://127.0.0.1:5174/).
	URL string `yaml:"url,omitempty" json:"url,omitempty"`
	// AllowedOrigins limits postMessage targets for the bridge; if empty and URL is set, defaults to URL's origin.
	AllowedOrigins []string `yaml:"allowed_origins,omitempty" json:"allowed_origins,omitempty"`
}

func (d *Development) validate() error {
	if d == nil {
		return nil
	}
	d.URL = strings.TrimSpace(d.URL)
	for i := range d.Command {
		d.Command[i] = strings.TrimSpace(d.Command[i])
	}
	if len(d.Command) > 0 {
		for _, a := range d.Command {
			if a == "" {
				return errors.New("manifest: development.command entries must be non-empty")
			}
		}
		if filepath.IsAbs(d.Command[0]) {
			return errors.New("manifest: development.command[0] must not be an absolute path")
		}
		if strings.Contains(d.Command[0], "..") {
			return errors.New("manifest: development.command[0] must not contain ..")
		}
	}
	if d.URL != "" {
		u, err := url.Parse(d.URL)
		if err != nil {
			return fmt.Errorf("manifest: development.url: %w", err)
		}
		if u.Scheme != "http" && u.Scheme != "https" {
			return errors.New("manifest: development.url must use http or https")
		}
		host := strings.ToLower(strings.Trim(u.Hostname(), "[]"))
		if host != "localhost" && host != "127.0.0.1" && host != "::1" && !strings.HasPrefix(host, "127.") {
			return errors.New("manifest: development.url host must be localhost, 127.0.0.1, or ::1")
		}
	}
	origins, err := d.normalizedAllowedOrigins()
	if err != nil {
		return err
	}
	if d.URL != "" {
		u, _ := url.Parse(d.URL)
		origin := (&url.URL{Scheme: u.Scheme, Host: u.Host}).String()
		found := false
		for _, o := range origins {
			if o == origin {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("manifest: development.url origin %q must appear in allowed_origins", origin)
		}
	}
	d.AllowedOrigins = origins
	return nil
}

func (d *Development) normalizedAllowedOrigins() ([]string, error) {
	if d == nil {
		return nil, nil
	}
	if len(d.AllowedOrigins) == 0 && d.URL != "" {
		u, err := url.Parse(d.URL)
		if err != nil {
			return nil, err
		}
		origin := (&url.URL{Scheme: u.Scheme, Host: u.Host}).String()
		return []string{origin}, nil
	}
	out := make([]string, 0, len(d.AllowedOrigins))
	for _, raw := range d.AllowedOrigins {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		u, err := url.Parse(raw)
		if err != nil {
			return nil, fmt.Errorf("manifest: allowed_origins: %w", err)
		}
		if u.Scheme != "http" && u.Scheme != "https" {
			return nil, errors.New("manifest: allowed_origins must be http(s) origins")
		}
		if u.Path != "" && u.Path != "/" {
			return nil, errors.New("manifest: allowed_origins must be origin only (no path)")
		}
		origin := (&url.URL{Scheme: u.Scheme, Host: u.Host}).String()
		out = append(out, origin)
	}
	if len(d.AllowedOrigins) > 0 && len(out) == 0 {
		return nil, errors.New("manifest: allowed_origins has no valid entries")
	}
	return out, nil
}

// EffectiveAllowedOrigins returns normalized origins for runtime (after validate).
func (d *Development) EffectiveAllowedOrigins() []string {
	if d == nil {
		return nil
	}
	o, _ := d.normalizedAllowedOrigins()
	return o
}
