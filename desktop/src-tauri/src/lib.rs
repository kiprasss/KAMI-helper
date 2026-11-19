use std::fs::{self, File};
use std::path::{Path, PathBuf};
use std::process::{Child, Command, Stdio};
use std::sync::{Arc, Mutex};

mod game;

use game::{GameMonitor, GameStatus};
use serde::Serialize;
use tauri::{PhysicalPosition, PhysicalSize};

use tauri::{
    Manager, RunEvent, WindowEvent,
    menu::{Menu, MenuItem},
    tray::{MouseButton, TrayIconBuilder, TrayIconEvent},
};

struct Backend(Mutex<Option<Child>>);

fn data_dir() -> PathBuf {
    std::env::var_os("LOCALAPPDATA")
        .map(PathBuf::from)
        .or_else(|| std::env::var_os("HOME").map(PathBuf::from))
        .unwrap_or_else(std::env::temp_dir)
        .join("KAMI Helper")
}

#[cfg(embedded_backend)]
const EMBEDDED_BACKEND: &[u8] = include_bytes!(concat!(
    env!("CARGO_MANIFEST_DIR"),
    "/binaries/kami-backend.exe"
));

#[cfg(embedded_backend)]
fn write_backend(dir: &Path) -> Option<PathBuf> {
    let path = dir.join("kami-backend.exe");

    let up_to_date = fs::read(&path)
        .map(|current| current == EMBEDDED_BACKEND)
        .unwrap_or(false);

    if up_to_date || fs::write(&path, EMBEDDED_BACKEND).is_ok() {
        return Some(path);
    }

    path.exists().then_some(path)
}

#[cfg(embedded_backend)]
fn backend_path(data: &Path) -> Option<PathBuf> {
    std::env::current_exe()
        .ok()
        .and_then(|exe| exe.parent().map(Path::to_path_buf))
        .and_then(|dir| write_backend(&dir))
        .or_else(|| write_backend(data))
}

#[cfg(not(embedded_backend))]
fn backend_path(_dir: &Path) -> Option<PathBuf> {
    let exe = std::env::current_exe().ok()?;
    let name = if cfg!(windows) { "kami-backend.exe" } else { "kami-backend" };
    let path = exe.parent()?.join(name);

    path.exists().then_some(path)
}

fn stop_stale_backend() {
    #[cfg(windows)]
    {
        use std::os::windows::process::CommandExt;

        let _ = Command::new("taskkill")
            .args(["/F", "/IM", "kami-backend.exe"])
            .stdin(Stdio::null())
            .stdout(Stdio::null())
            .stderr(Stdio::null())
            .creation_flags(0x0800_0000)
            .status();

        std::thread::sleep(std::time::Duration::from_millis(400));
    }
}

fn start_backend() -> Option<Child> {
    stop_stale_backend();

    let dir = data_dir();
    fs::create_dir_all(&dir).ok()?;

    let path = backend_path(&dir)?;

    let log = File::create(dir.join("backend.log")).ok()?;
    let log_err = log.try_clone().ok()?;

    let mut command = Command::new(path);
    command
        .current_dir(&dir)
        .env("CACHE_FILE", dir.join("forum-cache.json"))
        .stdin(Stdio::null())
        .stdout(Stdio::from(log))
        .stderr(Stdio::from(log_err));

    #[cfg(windows)]
    {
        use std::os::windows::process::CommandExt;
        command.creation_flags(0x0800_0000);
    }

    command.spawn().ok()
}

fn stop_backend(app: &tauri::AppHandle) {
    if let Some(state) = app.try_state::<Backend>() {
        if let Ok(mut guard) = state.0.lock() {
            if let Some(mut child) = guard.take() {
                let _ = child.kill();
                let _ = child.wait();
            }
        }
    }
}

const MINI_WIDTH: f64 = 300.0;
const MINI_HEIGHT: f64 = 46.0;
const MINI_TOP_MARGIN: f64 = 12.0;

#[derive(Serialize)]
struct MonitorInfo {
    id: String,
    label: String,
    primary: bool,
}

fn monitor_id(monitor: &tauri::Monitor) -> String {
    let position = monitor.position();
    monitor
        .name()
        .cloned()
        .unwrap_or_else(|| format!("{}x{}", position.x, position.y))
}

#[tauri::command]
fn list_monitors(app: tauri::AppHandle) -> Vec<MonitorInfo> {
    let primary = app.primary_monitor().ok().flatten().map(|m| monitor_id(&m));

    app.available_monitors()
        .unwrap_or_default()
        .iter()
        .enumerate()
        .map(|(index, monitor)| {
            let id = monitor_id(monitor);
            let is_primary = primary.as_deref() == Some(id.as_str());
            let size = monitor.size();
            let mut label = format!("Монитор {} — {}×{}", index + 1, size.width, size.height);
            if is_primary {
                label.push_str(" (основной)");
            }
            MonitorInfo {
                id,
                label,
                primary: is_primary,
            }
        })
        .collect()
}

#[tauri::command]
fn mini_show(app: tauri::AppHandle, monitor: Option<String>) -> Result<(), String> {
    let window = app
        .get_webview_window("mini")
        .ok_or("мини-окно не найдено")?;

    let monitors = app.available_monitors().map_err(|e| e.to_string())?;
    let target = monitor
        .and_then(|id| monitors.iter().find(|m| monitor_id(m) == id).cloned())
        .or_else(|| app.primary_monitor().ok().flatten())
        .or_else(|| monitors.first().cloned())
        .ok_or("мониторы не найдены")?;

    let scale = target.scale_factor();
    let width = (MINI_WIDTH * scale).round() as u32;
    let height = (MINI_HEIGHT * scale).round() as u32;
    let position = target.position();
    let size = target.size();

    let x = position.x + (size.width as i32 - width as i32) / 2;
    let y = position.y + (MINI_TOP_MARGIN * scale).round() as i32;

    window
        .set_size(PhysicalSize::new(width, height))
        .map_err(|e| e.to_string())?;
    window
        .set_position(PhysicalPosition::new(x, y))
        .map_err(|e| e.to_string())?;
    let _ = window.set_ignore_cursor_events(true);
    let _ = window.set_always_on_top(true);
    window.show().map_err(|e| e.to_string())?;

    Ok(())
}

#[tauri::command]
fn mini_hide(app: tauri::AppHandle) {
    if let Some(window) = app.get_webview_window("mini") {
        let _ = window.hide();
    }
}

#[tauri::command]
fn game_status(monitor: tauri::State<'_, Arc<GameMonitor>>) -> GameStatus {
    monitor.status()
}

fn show_main(app: &tauri::AppHandle) {
    mini_hide(app.clone());

    if let Some(window) = app.get_webview_window("main") {
        let _ = window.show();
        let _ = window.set_focus();
    }
}

#[cfg_attr(mobile, tauri::mobile_entry_point)]
pub fn run() {
    tauri::Builder::default()
        .plugin(tauri_plugin_single_instance::init(|app, _args, _cwd| {
            show_main(app);
        }))
        .plugin(tauri_plugin_global_shortcut::Builder::new().build())
        .plugin(tauri_plugin_opener::init())
        .setup(|app| {
            app.manage(Backend(Mutex::new(start_backend())));
            app.manage(GameMonitor::start());

            let open = MenuItem::with_id(app, "open", "Открыть KAMI HELPER", true, None::<&str>)?;

            let quit = MenuItem::with_id(app, "quit", "Выход", true, None::<&str>)?;

            let menu = Menu::with_items(app, &[&open, &quit])?;

            TrayIconBuilder::new()
                .icon(app.default_window_icon().unwrap().clone())
                .tooltip("KAMI HELPER")
                .menu(&menu)
                .show_menu_on_left_click(false)
                .on_menu_event(|app, event| match event.id().as_ref() {
                    "open" => show_main(app),
                    "quit" => {
                        app.exit(0);
                    }
                    _ => {}
                })
                .on_tray_icon_event(|tray, event| {
                    if let TrayIconEvent::DoubleClick {
                        button: MouseButton::Left,
                        ..
                    } = event
                    {
                        show_main(tray.app_handle());
                    }
                })
                .build(app)?;

            Ok(())
        })
        .on_window_event(|window, event| {
            if let WindowEvent::CloseRequested { api, .. } = event {
                api.prevent_close();
                let _ = window.hide();
            }
        })
        .invoke_handler(tauri::generate_handler![
            list_monitors,
            mini_show,
            mini_hide,
            game_status
        ])
        .build(tauri::generate_context!())
        .expect("не удалось запустить приложение")
        .run(|app, event| {
            if let RunEvent::Exit = event {
                stop_backend(app);
                game::stop_fps_session();
            }
        });
}
