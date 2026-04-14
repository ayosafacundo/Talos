use anyhow::Context;
use std::env;
use std::time::{SystemTime, UNIX_EPOCH};
use talos_sdk::Client;

#[tokio::main]
async fn main() -> anyhow::Result<()> {
    let app_id = env::var("TALOS_APP_ID").context("missing TALOS_APP_ID")?;
    let socket = env::var("TALOS_HUB_SOCKET").context("missing TALOS_HUB_SOCKET")?;

    let mut client = Client::dial(&socket).await?;

    let (state, found) = client.load_state(&app_id).await?;
    if found {
        println!("example-rust-app: previous state={}", String::from_utf8_lossy(&state));
    }

    let now = SystemTime::now().duration_since(UNIX_EPOCH)?.as_secs();
    client
        .save_state(&app_id, format!("last_run={now}").as_bytes())
        .await?;

    let (granted, msg) = client
        .request_permission(&app_id, "net:internet", "Example Rust app network check")
        .await?;
    println!("example-rust-app: permission granted={granted} message={msg}");

    let resolved = client
        .resolve_path(&app_id, "example-rust-app/heartbeat.txt")
        .await?;
    std::fs::create_dir_all(
        std::path::Path::new(&resolved)
            .parent()
            .unwrap_or_else(|| std::path::Path::new(".")),
    )?;
    std::fs::write(&resolved, b"ok\n")?;

    let recipients = client
        .broadcast(&app_id, "app:example:hello", b"hello from rust")
        .await?;
    println!("example-rust-app: broadcast recipients={recipients}");
    Ok(())
}
