// Prevents additional console window on Windows in release
#![cfg_attr(not(debug_assertions), windows_subsystem = "windows")]

use tauri::{
    Manager,
    tray::{MouseButton, MouseButtonState, TrayIconBuilder, TrayIconEvent},
};

mod sidecar;

#[tauri::command]
fn get_status(state: tauri::State<'_, sidecar::ActuatorState>) -> serde_json::Value {
    sidecar::get_status(state)
}

#[tauri::command]
fn start_actuator(
    state: tauri::State<'_, sidecar::ActuatorState>,
    app: tauri::AppHandle,
    broker_url: String,
    broker_token: String,
    actuator_id: String,
) -> Result<String, String> {
    sidecar::start_actuator(state, app, broker_url, broker_token, actuator_id)
}

#[tauri::command]
fn stop_actuator(state: tauri::State<'_, sidecar::ActuatorState>) -> Result<String, String> {
    sidecar::stop_actuator(state)
}

#[tauri::command]
fn get_logs(state: tauri::State<'_, sidecar::ActuatorState>) -> Vec<String> {
    sidecar::get_logs(state)
}

fn main() {
    tauri::Builder::default()
        .plugin(tauri_plugin_shell::init())
        .manage(sidecar::ActuatorState::default())
        .invoke_handler(tauri::generate_handler![
            get_status,
            start_actuator,
            stop_actuator,
            get_logs,
        ])
        .setup(|app| {
            // Build tray icon
            let _tray = TrayIconBuilder::new()
                .icon(app.default_window_icon().unwrap().clone())
                .tooltip("Actuator G")
                .on_tray_icon_event(|tray, event| {
                    if let TrayIconEvent::Click {
                        button: MouseButton::Left,
                        button_state: MouseButtonState::Up,
                        ..
                    } = event
                    {
                        let app = tray.app_handle();
                        if let Some(window) = app.get_webview_window("main") {
                            if window.is_visible().unwrap_or(false) {
                                let _ = window.hide();
                            } else {
                                let _ = window.show();
                                let _ = window.set_focus();
                            }
                        }
                    }
                })
                .build(app)?;

            Ok(())
        })
        .run(tauri::generate_context!())
        .expect("error while running application");
}
