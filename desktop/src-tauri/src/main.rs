// Prevents additional console window on Windows in release
#![cfg_attr(not(debug_assertions), windows_subsystem = "windows")]

use tauri::{
    Manager,
    menu::{MenuBuilder, MenuItemBuilder, PredefinedMenuItem},
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
            // Build tray menu
            let open_dashboard = MenuItemBuilder::with_id("open_dashboard", "Open Dashboard")
                .build(app)?;
            let grant_file = MenuItemBuilder::with_id("grant_file", "Grant File Access...")
                .build(app)?;
            let grant_folder = MenuItemBuilder::with_id("grant_folder", "Grant Folder Access...")
                .build(app)?;
            let view_access = MenuItemBuilder::with_id("view_access", "View Granted Access")
                .build(app)?;
            let pause_agent = MenuItemBuilder::with_id("pause_agent", "Pause Agent")
                .build(app)?;
            let settings = MenuItemBuilder::with_id("settings", "Settings...")
                .build(app)?;
            let quit = MenuItemBuilder::with_id("quit", "Quit Botster")
                .build(app)?;

            let sep1 = PredefinedMenuItem::separator(app)?;
            let sep2 = PredefinedMenuItem::separator(app)?;

            let menu = MenuBuilder::new(app)
                .item(&open_dashboard)
                .item(&sep1)
                .item(&grant_file)
                .item(&grant_folder)
                .item(&view_access)
                .item(&sep2)
                .item(&pause_agent)
                .item(&settings)
                .item(&quit)
                .build()?;

            // Build tray icon with menu
            let _tray = TrayIconBuilder::new()
                .icon(app.default_window_icon().unwrap().clone())
                .tooltip("Botster Desktop — Connected")
                .menu(&menu)
                .on_menu_event(|app, event| {
                    match event.id().as_ref() {
                        "open_dashboard" | "view_access" | "settings" => {
                            if let Some(window) = app.get_webview_window("main") {
                                let _ = window.show();
                                let _ = window.set_focus();
                            }
                        }
                        "quit" => {
                            app.exit(0);
                        }
                        _ => {}
                    }
                })
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
