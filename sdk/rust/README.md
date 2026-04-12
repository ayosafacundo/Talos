# Talos Rust SDK (Baseline)

This crate is the Phase 2 baseline wrapper for Rust tiny apps.

Current shape:

- `TalosTransport` trait abstracts transport details
- `TalosClient` exposes:
  - `save_state`
  - `load_state`
  - `send_message`
  - `request_permission`

Next step is wiring this trait to tonic-based gRPC over Unix sockets / named pipes.
