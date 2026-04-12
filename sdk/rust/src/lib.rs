//! Talos hub gRPC client (Unix domain socket). Mirrors [`sdk/go/talos`](../../go/talos).

pub mod hub {
    pub mod v1 {
        tonic::include_proto!("talos.hub.v1");
    }
}

use anyhow::{anyhow, Context};
use hub::v1::hub_service_client::HubServiceClient;
use hub::v1::{BroadcastRequest, LoadStateRequest, Message, PermissionRequest, ResolvePathRequest, RouteRequest, SaveStateRequest};

#[cfg(unix)]
use std::path::PathBuf;
#[cfg(unix)]
use hyper_util::rt::TokioIo;
#[cfg(unix)]
use tokio::net::UnixStream;
#[cfg(unix)]
use tonic::transport::{Endpoint, Uri};
#[cfg(unix)]
use tower::service_fn;

/// Active connection to the Talos hub (same RPC surface as Go SDK).
pub struct Client {
    inner: HubServiceClient<tonic::transport::Channel>,
}

impl Client {
    /// Connect using a socket URL such as `unix:///path/to/talos.sock`.
    #[cfg(unix)]
    pub async fn dial(socket_url: &str) -> anyhow::Result<Self> {
        let path = parse_unix_socket_path(socket_url)?;
        let path = PathBuf::from(path);
        let channel = Endpoint::try_from("http://127.0.0.1:1")
            .context("endpoint")?
            .connect_with_connector(service_fn(move |_uri: Uri| {
                let p = path.clone();
                async move {
                    let io = UnixStream::connect(p).await?;
                    Ok::<_, std::io::Error>(TokioIo::new(io))
                }
            }))
            .await
            .context("uds connect")?;
        Ok(Self {
            inner: HubServiceClient::new(channel),
        })
    }

    #[cfg(not(unix))]
    pub async fn dial(_socket_url: &str) -> anyhow::Result<Self> {
        Err(anyhow!(
            "Talos Rust SDK: only Unix domain sockets are implemented (use unix:// paths on Linux/macOS)"
        ))
    }

    pub async fn send_message(
        &mut self,
        source_app_id: &str,
        target_app_id: &str,
        typ: &str,
        payload: &[u8],
    ) -> anyhow::Result<Message> {
        let resp = self
            .inner
            .route(RouteRequest {
                message: Some(Message {
                    source_app_id: source_app_id.to_string(),
                    target_app_id: target_app_id.to_string(),
                    r#type: typ.to_string(),
                    payload: payload.to_vec(),
                    request_id: String::new(),
                }),
            })
            .await
            .context("route rpc")?
            .into_inner();
        if !resp.error.is_empty() {
            return Err(anyhow!("route error: {}", resp.error));
        }
        resp.message.ok_or_else(|| anyhow!("empty route response"))
    }

    pub async fn broadcast(&mut self, source_app_id: &str, typ: &str, payload: &[u8]) -> anyhow::Result<i32> {
        let resp = self
            .inner
            .broadcast(BroadcastRequest {
                message: Some(Message {
                    source_app_id: source_app_id.to_string(),
                    target_app_id: String::new(),
                    r#type: typ.to_string(),
                    payload: payload.to_vec(),
                    request_id: String::new(),
                }),
            })
            .await
            .context("broadcast rpc")?
            .into_inner();
        Ok(resp.recipient_count)
    }

    pub async fn save_state(&mut self, app_id: &str, data: &[u8]) -> anyhow::Result<()> {
        let resp = self
            .inner
            .save_state(SaveStateRequest {
                app_id: app_id.to_string(),
                data: data.to_vec(),
            })
            .await
            .context("save_state rpc")?
            .into_inner();
        if !resp.ok {
            return Err(anyhow!("save state failed: {}", resp.error));
        }
        Ok(())
    }

    pub async fn load_state(&mut self, app_id: &str) -> anyhow::Result<(Vec<u8>, bool)> {
        let resp = self
            .inner
            .load_state(LoadStateRequest {
                app_id: app_id.to_string(),
            })
            .await
            .context("load_state rpc")?
            .into_inner();
        if !resp.error.is_empty() {
            return Err(anyhow!("load state failed: {}", resp.error));
        }
        Ok((resp.data, resp.found))
    }

    pub async fn request_permission(
        &mut self,
        app_id: &str,
        scope: &str,
        reason: &str,
    ) -> anyhow::Result<(bool, String)> {
        let resp = self
            .inner
            .request_permission(PermissionRequest {
                app_id: app_id.to_string(),
                scope: scope.to_string(),
                reason: reason.to_string(),
            })
            .await
            .context("request_permission rpc")?
            .into_inner();
        Ok((resp.granted, resp.message))
    }

    pub async fn resolve_path(&mut self, app_id: &str, relative_path: &str) -> anyhow::Result<String> {
        let resp = self
            .inner
            .resolve_path(ResolvePathRequest {
                app_id: app_id.to_string(),
                relative_path: relative_path.to_string(),
            })
            .await
            .context("resolve_path rpc")?
            .into_inner();
        if !resp.allowed {
            return Err(anyhow!("resolve path denied: {}", resp.error));
        }
        Ok(resp.resolved_path)
    }
}

#[cfg(unix)]
fn parse_unix_socket_path(socket_url: &str) -> anyhow::Result<&str> {
    let prefix = "unix://";
    if !socket_url.starts_with(prefix) {
        anyhow::bail!("unsupported socket url (expected unix://): {socket_url:?}");
    }
    Ok(socket_url.trim_start_matches(prefix))
}
