//! Ignored by default: requires a running Talos hub socket at `TALOS_TEST_SOCKET` (unix://...).

use std::env;

#[tokio::test]
#[ignore = "integration: set TALOS_TEST_SOCKET=unix:///path/to/sock and run Talos"]
async fn dial_and_load_state_roundtrip() {
    let url = env::var("TALOS_TEST_SOCKET").expect("TALOS_TEST_SOCKET");
    let mut c = talos_sdk::Client::dial(&url).await.expect("dial");
    let app = "app.integration.test";
    c.save_state(app, b"ping").await.expect("save");
    let (data, found) = c.load_state(app).await.expect("load");
    assert!(found);
    assert_eq!(data, b"ping");
}
