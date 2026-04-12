package security

import "sync"

const (
	ScopeFSData = "fs:data"
)

type RequestHandler func(appID, scope, reason string) (bool, string)

// Permissions tracks per-app permission grants.
type Permissions struct {
	mu       sync.RWMutex
	grants   map[string]map[string]bool
	onPrompt RequestHandler
}

func NewPermissions(onPrompt RequestHandler) *Permissions {
	return &Permissions{
		grants:   make(map[string]map[string]bool),
		onPrompt: onPrompt,
	}
}

func (p *Permissions) IsGranted(appID, scope string) bool {
	if scope == ScopeFSData {
		return true
	}

	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.grants[appID]["*"] {
		return true
	}
	return p.grants[appID][scope]
}

func (p *Permissions) Set(appID, scope string, granted bool) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if _, ok := p.grants[appID]; !ok {
		p.grants[appID] = make(map[string]bool)
	}
	p.grants[appID][scope] = granted
}

func (p *Permissions) Export() map[string]map[string]bool {
	p.mu.RLock()
	defer p.mu.RUnlock()

	out := make(map[string]map[string]bool, len(p.grants))
	for appID, scopes := range p.grants {
		out[appID] = make(map[string]bool, len(scopes))
		for scope, granted := range scopes {
			out[appID][scope] = granted
		}
	}
	return out
}

func (p *Permissions) Import(grants map[string]map[string]bool) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.grants = make(map[string]map[string]bool, len(grants))
	for appID, scopes := range grants {
		p.grants[appID] = make(map[string]bool, len(scopes))
		for scope, granted := range scopes {
			p.grants[appID][scope] = granted
		}
	}
}

func (p *Permissions) Request(appID, scope, reason string) (bool, string) {
	if p.IsGranted(appID, scope) {
		return true, "already granted"
	}

	if p.onPrompt == nil {
		return false, "permission requires explicit host grant"
	}

	granted, msg := p.onPrompt(appID, scope, reason)
	if granted {
		p.Set(appID, scope, true)
	}
	return granted, msg
}
