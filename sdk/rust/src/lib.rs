use anyhow::Result;

pub struct PermissionResult {
    pub granted: bool,
    pub message: String,
}

pub trait TalosTransport {
    fn save_state(&self, app_id: &str, data: Vec<u8>) -> Result<()>;
    fn load_state(&self, app_id: &str) -> Result<Option<Vec<u8>>>;
    fn send_message(&self, app_id: &str, target_id: &str, payload: Vec<u8>) -> Result<Option<Vec<u8>>>;
    fn request_permission(&self, app_id: &str, scope: &str, reason: &str) -> Result<PermissionResult>;
    fn resolve_path(&self, app_id: &str, relative_path: &str) -> Result<String>;
}

pub struct TalosClient<T: TalosTransport> {
    app_id: String,
    transport: T,
}

impl<T: TalosTransport> TalosClient<T> {
    pub fn new(app_id: impl Into<String>, transport: T) -> Self {
        Self {
            app_id: app_id.into(),
            transport,
        }
    }

    pub fn save_state(&self, data: Vec<u8>) -> Result<()> {
        self.transport.save_state(&self.app_id, data)
    }

    pub fn load_state(&self) -> Result<Option<Vec<u8>>> {
        self.transport.load_state(&self.app_id)
    }

    pub fn send_message(&self, target_id: &str, payload: Vec<u8>) -> Result<Option<Vec<u8>>> {
        self.transport.send_message(&self.app_id, target_id, payload)
    }

    pub fn request_permission(&self, scope: &str, reason: &str) -> Result<PermissionResult> {
        self.transport.request_permission(&self.app_id, scope, reason)
    }

    pub fn resolve_path(&self, relative_path: &str) -> Result<String> {
        self.transport.resolve_path(&self.app_id, relative_path)
    }
}
