package security

import "sync"

const (
	ScopeFSData = "fs:data"

	// MsgPendingHostApproval is returned by the UI prompt hook when the host must
	// collect a decision; Request then blocks until CompletePendingDecision runs.
	MsgPendingHostApproval = "pending host approval"
)

type permissionWaitOutcome struct {
	granted bool
	msg     string
}

type RequestHandler func(appID, scope, reason string) (bool, string)

// Permissions tracks per-app permission grants.
type Permissions struct {
	mu       sync.RWMutex
	grants   map[string]map[string]bool
	onPrompt RequestHandler

	waitMu  sync.Mutex
	waiters map[string][]chan permissionWaitOutcome
}

func NewPermissions(onPrompt RequestHandler) *Permissions {
	return &Permissions{
		grants:   make(map[string]map[string]bool),
		onPrompt: onPrompt,
		waiters:  make(map[string][]chan permissionWaitOutcome),
	}
}

func waitKey(appID, scope string) string {
	return appID + "\x00" + scope
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

// Clear removes a scope entry so the next Request can prompt again (revocation).
func (p *Permissions) Clear(appID, scope string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	scopes, ok := p.grants[appID]
	if !ok {
		return
	}
	delete(scopes, scope)
	if len(scopes) == 0 {
		delete(p.grants, appID)
	}
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
		return true, msg
	}
	if msg != MsgPendingHostApproval {
		return granted, msg
	}

	ch := make(chan permissionWaitOutcome, 1)
	key := waitKey(appID, scope)
	p.waitMu.Lock()
	p.waiters[key] = append(p.waiters[key], ch)
	p.waitMu.Unlock()

	out := <-ch
	if out.granted {
		p.Set(appID, scope, true)
	}
	return out.granted, out.msg
}

// CompletePendingDecision unblocks Request callers waiting on (appID, scope).
func (p *Permissions) CompletePendingDecision(appID, scope string, granted bool, msg string) {
	key := waitKey(appID, scope)
	p.waitMu.Lock()
	waiters := p.waiters[key]
	delete(p.waiters, key)
	p.waitMu.Unlock()

	out := permissionWaitOutcome{granted: granted, msg: msg}
	for _, ch := range waiters {
		select {
		case ch <- out:
		default:
		}
	}
}
