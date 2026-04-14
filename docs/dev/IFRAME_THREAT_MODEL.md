# Iframe Bridge Threat Model (Phase 4)

This note captures the current iframe bridge security posture and the next isolation backlog.

## Assets

- Host-bound APIs exposed through Launchpad bridge handlers.
- Package-scoped state and permissioned host paths.
- User trust decisions and audit history.

## Trust boundaries

- Tiny app iframe content is untrusted by default.
- Host runtime (`window.go.main.App`) is trusted and must not be directly exposed to iframe code.
- Bridge requests are only accepted through the v1 envelope and trusted sender checks.

## Current controls

- Per-instance bridge token (`_talos_bt`) and trusted sender resolution in Launchpad bridge flow.
- `allowed_origins` enforcement for postMessage origin checks.
- Method allowlist for bridge RPC calls.
- Permission grants/denies persisted and auditable.
- Scoped path resolution with explicit permission gates.

## Threats and mitigations

- Token replay across instances: mitigated by instance-bound token checks and sender window matching.
- Origin spoofing: mitigated by strict `allowed_origins` and loopback origin expansion only for equivalent local hosts.
- Overbroad API surface: mitigated by explicit method allowlist and per-method validation.
- Package tampering: mitigated by hash/signature trust statuses and optional paranoid trust gate.

## MVP isolation backlog

1. Add rate limits for bridge requests per app instance.
2. Add structured security telemetry for rejected bridge calls.
3. Add optional stricter iframe sandbox flags by app capability profile.

## Stretch backlog

1. Explore OS-level process sandboxing profiles per package binary.
2. Evaluate stronger origin pinning with signed manifest metadata.
3. Add host policy profiles for environment-variable exposure per package.
