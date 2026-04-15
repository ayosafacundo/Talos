//! Example Rust sidecar: async hub client, structured `serde` status, and graceful shutdown via `tokio::signal`.

use anyhow::Context;
use serde::Serialize;
use std::env;
use std::time::{Duration, SystemTime, UNIX_EPOCH};
use talos_sdk::Client;
use tokio::time::{interval, MissedTickBehavior};

const STATUS_FILE: &str = "example_rust_status.json";
const HEARTBEAT_FILE: &str = "heartbeat.txt";
const GO_EXAMPLE_ID: &str = "app.example.go";

#[derive(Serialize)]
#[serde(rename_all = "snake_case")]
struct Status {
    schema_version: u32,
    app_id: String,
    /// Mirrors Launchpad **package** development (`TALOS_PACKAGE_DEVELOPMENT`): Settings toggle or source `TALOS_DEV_MODE` policy.
    talos_package_development: bool,
    boot_unix_ms: u128,
    last_tick_unix_ms: u128,
    ticks: u64,
    net_internet_granted: bool,
    #[serde(skip_serializing_if = "String::is_empty")]
    net_internet_message: String,
    #[serde(skip_serializing_if = "String::is_empty")]
    send_message_note: String,
    last_bootstrap_broadcast_recipients: i32,
    prev_hub_state_found: bool,
    #[serde(skip_serializing_if = "String::is_empty")]
    prev_hub_state_snippet: String,
    #[serde(skip_serializing_if = "String::is_empty")]
    heartbeat_file_preview: String,
    crate_version: &'static str,
}

fn unix_ms() -> u128 {
    SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .map(|d| d.as_millis())
        .unwrap_or(0)
}

fn package_dev_from_env() -> bool {
    matches!(
        env::var("TALOS_PACKAGE_DEVELOPMENT")
            .map(|v| v.trim().to_ascii_lowercase())
            .unwrap_or_default()
            .as_str(),
        "1" | "true" | "yes"
    )
}

fn clip(s: &str, max: usize) -> String {
    let t = s.trim();
    if max == 0 || t.len() <= max {
        return t.to_string();
    }
    format!("{}…", &t[..max])
}

struct Runner {
    app_id: String,
    package_dev: bool,
    boot_ms: u128,
    ticks: u64,
    net_granted: bool,
    net_message: String,
    send_note: String,
    bcast_n: i32,
    prev_found: bool,
    prev_snip: String,
}

impl Runner {
    async fn bootstrap(client: &mut Client, app_id: &str, package_dev: bool) -> anyhow::Result<Self> {
        let (prev, found) = client.load_state(app_id).await?;
        let prev_snip = if found && !prev.is_empty() {
            clip(&String::from_utf8_lossy(&prev), 96)
        } else {
            String::new()
        };

        let stamp = format!("rust_run_ms={}\n", unix_ms());
        client.save_state(app_id, stamp.as_bytes()).await?;

        let (granted, msg) = client
            .request_permission(
                app_id,
                "net:internet",
                "Example Rust app: demonstrate permission flow (declared in manifest).",
            )
            .await?;

        let send_note = match client
            .send_message(
                app_id,
                GO_EXAMPLE_ID,
                "app:example:from-rust",
                br#"{"from":"example-rust"}"#,
            )
            .await
        {
            Ok(_) => format!("Routed a message to {GO_EXAMPLE_ID}."),
            Err(e) => format!(
                "SendMessage to {GO_EXAMPLE_ID} failed (normal if that app is offline): {e:#}"
            ),
        };

        let bcast_n = client
            .broadcast(app_id, "app:example:rust-broadcast", b"hello from Rust")
            .await?;

        let hb = format!("bootstrap_ms={}\n", unix_ms());
        client.write_scoped_file(app_id, HEARTBEAT_FILE, hb.as_bytes()).await?;

        let _ = client
            .append_package_log(
                app_id,
                "INFO",
                &format!("example-rust-app ready (package_dev={package_dev}; broadcast n={bcast_n})"),
            )
            .await;

        Ok(Self {
            app_id: app_id.to_string(),
            package_dev,
            boot_ms: unix_ms(),
            ticks: 0,
            net_granted: granted,
            net_message: msg,
            send_note,
            bcast_n,
            prev_found: found,
            prev_snip,
        })
    }

    async fn tick(&mut self, client: &mut Client) -> anyhow::Result<()> {
        self.ticks += 1;

        let hb_preview = match client.read_scoped_file(&self.app_id, HEARTBEAT_FILE).await {
            Ok(bytes) => clip(&String::from_utf8_lossy(&bytes), 120),
            Err(e) => format!("(read {HEARTBEAT_FILE} failed: {e:#})"),
        };

        let st = Status {
            schema_version: 1,
            app_id: self.app_id.clone(),
            talos_package_development: self.package_dev,
            boot_unix_ms: self.boot_ms,
            last_tick_unix_ms: unix_ms(),
            ticks: self.ticks,
            net_internet_granted: self.net_granted,
            net_internet_message: self.net_message.clone(),
            send_message_note: self.send_note.clone(),
            last_bootstrap_broadcast_recipients: self.bcast_n,
            prev_hub_state_found: self.prev_found,
            prev_hub_state_snippet: self.prev_snip.clone(),
            heartbeat_file_preview: hb_preview,
            crate_version: env!("CARGO_PKG_VERSION"),
        };

        let json = serde_json::to_vec_pretty(&st)?;
        client
            .write_scoped_file(&self.app_id, STATUS_FILE, &json)
            .await?;

        if self.ticks.is_multiple_of(5) {
            let _ = client
                .append_package_log(
                    &self.app_id,
                    "DEBUG",
                    &format!("example-rust-app tick {}", self.ticks),
                )
                .await;
        }

        Ok(())
    }
}

#[tokio::main]
async fn main() -> anyhow::Result<()> {
    let app_id = env::var("TALOS_APP_ID").context("missing TALOS_APP_ID")?;
    let socket = env::var("TALOS_HUB_SOCKET").context("missing TALOS_HUB_SOCKET")?;
    let package_dev = package_dev_from_env();

    eprintln!(
        "example-rust-app: TALOS_PACKAGE_DEVELOPMENT={} (Launchpad package development)",
        if package_dev { "on" } else { "off" }
    );

    let mut client = Client::dial(&socket).await?;
    let mut runner = Runner::bootstrap(&mut client, &app_id, package_dev).await?;
    runner.tick(&mut client).await?;

    let mut tick = interval(Duration::from_secs(3));
    tick.set_missed_tick_behavior(MissedTickBehavior::Delay);

    loop {
        tokio::select! {
            biased;
            _ = tokio::signal::ctrl_c() => {
                eprintln!("example-rust-app: shutdown (ctrl+c)");
                break;
            }
            _ = tick.tick() => {
                runner.tick(&mut client).await?;
            }
        }
    }

    Ok(())
}
