# Talos Rust SDK

gRPC hub client over **Unix domain sockets** (same contract as `sdk/go/talos`).

**Windows:** `Client::dial` is not implemented for named pipes yet; use the Go SDK or TS iframe bridge on Windows until pipe transport is added.

```rust
#[tokio::main]
async fn main() -> anyhow::Result<()> {
    let mut c = talos_sdk::Client::dial("unix:///path/to/talos.sock").await?;
    c.save_state("app.my.app", b"hello").await?;
    let (data, found) = c.load_state("app.my.app").await?;
    assert!(found);
    Ok(())
}
```

Use `std::env::var("TALOS_HUB_SOCKET")` or your manifest wiring to obtain the socket URL the host injects for binaries.

Integration test (ignored by default): `tests/dial_smoke.rs` — set `TALOS_TEST_SOCKET` and run with `cargo test -- --ignored`.
