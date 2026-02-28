use serde::Serialize;
use std::sync::Mutex;
use std::collections::VecDeque;
use tauri::AppHandle;
use tauri_plugin_shell::ShellExt;

const MAX_LOG_LINES: usize = 1000;

#[derive(Debug, Clone, Serialize)]
pub enum ConnectionStatus {
    Disconnected,
    Connecting,
    Connected,
    Error(String),
}

pub struct ActuatorStateInner {
    pub status: ConnectionStatus,
    pub logs: VecDeque<String>,
    pub child: Option<tauri_plugin_shell::process::CommandChild>,
    pub broker_url: String,
    pub actuator_id: String,
}

impl Default for ActuatorStateInner {
    fn default() -> Self {
        Self {
            status: ConnectionStatus::Disconnected,
            logs: VecDeque::with_capacity(MAX_LOG_LINES),
            child: None,
            broker_url: String::new(),
            actuator_id: String::new(),
        }
    }
}

pub struct ActuatorState(pub Mutex<ActuatorStateInner>);

impl Default for ActuatorState {
    fn default() -> Self {
        Self(Mutex::new(ActuatorStateInner::default()))
    }
}

pub fn get_status(state: tauri::State<'_, ActuatorState>) -> serde_json::Value {
    let inner = state.0.lock().unwrap();
    serde_json::json!({
        "status": inner.status,
        "brokerUrl": inner.broker_url,
        "actuatorId": inner.actuator_id,
        "logCount": inner.logs.len(),
    })
}

pub fn start_actuator(
    state: tauri::State<'_, ActuatorState>,
    app: AppHandle,
    broker_url: String,
    broker_token: String,
    actuator_id: String,
) -> Result<String, String> {
    let mut inner = state.0.lock().unwrap();

    // Stop existing if running
    if let Some(child) = inner.child.take() {
        let _ = child.kill();
    }

    inner.status = ConnectionStatus::Connecting;
    inner.broker_url = broker_url.clone();
    inner.actuator_id = actuator_id.clone();
    inner.logs.clear();

    // Spawn sidecar
    let sidecar = app
        .shell()
        .sidecar("binaries/actuator-g")
        .map_err(|e| format!("Failed to create sidecar command: {}", e))?
        .args([
            "--id", &actuator_id,
        ])
        .envs([
            ("SEKS_BROKER_URL", broker_url.as_str()),
            ("SEKS_BROKER_TOKEN", broker_token.as_str()),
        ]);

    let state_clone = std::sync::Arc::new(Mutex::new(()));
    let (mut rx, child) = sidecar
        .spawn()
        .map_err(|e| format!("Failed to spawn actuator: {}", e))?;

    inner.child = Some(child);
    inner.status = ConnectionStatus::Connected;

    // Log reader — runs in background
    // Note: in production, we'd update state from the event stream
    // For now, logs are collected but state updates are basic
    tauri::async_runtime::spawn(async move {
        use tauri_plugin_shell::process::CommandEvent;
        while let Some(event) = rx.recv().await {
            match event {
                CommandEvent::Stdout(line) => {
                    // Would update state.logs here with proper Arc<Mutex<>> sharing
                    eprintln!("[actuator-g stdout] {}", String::from_utf8_lossy(&line));
                }
                CommandEvent::Stderr(line) => {
                    eprintln!("[actuator-g stderr] {}", String::from_utf8_lossy(&line));
                }
                CommandEvent::Terminated(payload) => {
                    eprintln!("[actuator-g] terminated: {:?}", payload);
                    break;
                }
                _ => {}
            }
        }
    });

    Ok("Actuator started".to_string())
}

pub fn stop_actuator(state: tauri::State<'_, ActuatorState>) -> Result<String, String> {
    let mut inner = state.0.lock().unwrap();
    if let Some(child) = inner.child.take() {
        child.kill().map_err(|e| format!("Failed to stop: {}", e))?;
    }
    inner.status = ConnectionStatus::Disconnected;
    Ok("Actuator stopped".to_string())
}

pub fn get_logs(state: tauri::State<'_, ActuatorState>) -> Vec<String> {
    let inner = state.0.lock().unwrap();
    inner.logs.iter().cloned().collect()
}
