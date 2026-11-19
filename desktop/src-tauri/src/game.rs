use std::sync::Arc;
use std::sync::Mutex;
use std::sync::atomic::{AtomicBool, AtomicU32, Ordering};
use std::time::Duration;

use serde::Serialize;

const GAME_PROCESSES: &[&str] = &["gta5.exe", "gta5_enhanced.exe"];

#[derive(Clone, Serialize)]
pub struct GameStatus {
    pub running: bool,
    pub fps: Option<u32>,
    pub fps_available: bool,
}

pub struct GameMonitor {
    pid: AtomicU32,
    presents: AtomicU32,
    fps_available: AtomicBool,
    status: Mutex<GameStatus>,
}

impl GameMonitor {
    pub fn start() -> Arc<Self> {
        let monitor = Arc::new(Self {
            pid: AtomicU32::new(0),
            presents: AtomicU32::new(0),
            fps_available: AtomicBool::new(false),
            status: Mutex::new(GameStatus {
                running: false,
                fps: None,
                fps_available: false,
            }),
        });

        etw::start(monitor.clone());

        let worker = monitor.clone();
        std::thread::spawn(move || {
            loop {
                std::thread::sleep(Duration::from_secs(1));
                worker.tick();
            }
        });

        monitor
    }

    fn tick(&self) {
        let pid = find_game_pid().unwrap_or(0);
        let previous = self.pid.swap(pid, Ordering::SeqCst);
        let presents = self.presents.swap(0, Ordering::SeqCst);
        let fps_available = self.fps_available.load(Ordering::SeqCst);

        let fps = if pid != 0 && pid == previous && fps_available {
            Some(presents)
        } else {
            None
        };

        if let Ok(mut status) = self.status.lock() {
            *status = GameStatus {
                running: pid != 0,
                fps,
                fps_available,
            };
        }
    }

    pub fn status(&self) -> GameStatus {
        self.status
            .lock()
            .map(|s| s.clone())
            .unwrap_or(GameStatus {
                running: false,
                fps: None,
                fps_available: false,
            })
    }

    #[allow(dead_code)]
    fn on_present(&self, process_id: u32) {
        let pid = self.pid.load(Ordering::Relaxed);
        if pid != 0 && pid == process_id {
            self.presents.fetch_add(1, Ordering::Relaxed);
        }
    }
}

#[cfg(windows)]
fn find_game_pid() -> Option<u32> {
    use windows_sys::Win32::Foundation::{CloseHandle, INVALID_HANDLE_VALUE};
    use windows_sys::Win32::System::Diagnostics::ToolHelp::{
        CreateToolhelp32Snapshot, PROCESSENTRY32W, Process32FirstW, Process32NextW,
        TH32CS_SNAPPROCESS,
    };

    unsafe {
        let snapshot = CreateToolhelp32Snapshot(TH32CS_SNAPPROCESS, 0);
        if snapshot == INVALID_HANDLE_VALUE {
            return None;
        }

        let mut entry: PROCESSENTRY32W = std::mem::zeroed();
        entry.dwSize = std::mem::size_of::<PROCESSENTRY32W>() as u32;

        let mut found = None;
        let mut ok = Process32FirstW(snapshot, &mut entry) != 0;

        while ok {
            let len = entry
                .szExeFile
                .iter()
                .position(|&c| c == 0)
                .unwrap_or(entry.szExeFile.len());
            let name = String::from_utf16_lossy(&entry.szExeFile[..len]).to_lowercase();

            if GAME_PROCESSES.contains(&name.as_str()) {
                found = Some(entry.th32ProcessID);
                break;
            }

            ok = Process32NextW(snapshot, &mut entry) != 0;
        }

        CloseHandle(snapshot);
        found
    }
}

#[cfg(not(windows))]
fn find_game_pid() -> Option<u32> {
    let _ = GAME_PROCESSES;
    None
}

#[cfg(windows)]
mod etw {
    use std::sync::Arc;
    use std::sync::atomic::Ordering;

    use ferrisetw::provider::Provider;
    use ferrisetw::trace::{TraceTrait, UserTrace, stop_trace_by_name};
    use ferrisetw::{EventRecord, GUID, SchemaLocator};

    use super::GameMonitor;

    const SESSION_NAME: &str = "KAMIHelperFPS";
    const DXGI_PROVIDER: &str = "CA11C036-0102-4A2D-A6AD-F03CFED5D3C9";
    const PRESENT_START: u16 = 42;
    const PRESENT_MULTIPLANE_START: u16 = 55;

    pub fn start(monitor: Arc<GameMonitor>) {
        std::thread::spawn(move || {
            let _ = stop_trace_by_name(SESSION_NAME);

            let callback_monitor = monitor.clone();
            let provider = Provider::by_guid(GUID::from(DXGI_PROVIDER))
                .add_callback(move |record: &EventRecord, _: &SchemaLocator| {
                    let id = record.event_id();
                    if id == PRESENT_START || id == PRESENT_MULTIPLANE_START {
                        callback_monitor.on_present(record.process_id());
                    }
                })
                .build();

            match UserTrace::new()
                .named(SESSION_NAME.to_string())
                .enable(provider)
                .start_and_process()
            {
                Ok(trace) => {
                    monitor.fps_available.store(true, Ordering::SeqCst);
                    std::mem::forget(trace);
                }
                Err(_) => {
                    monitor.fps_available.store(false, Ordering::SeqCst);
                }
            }
        });
    }

    pub fn stop() {
        let _ = stop_trace_by_name(SESSION_NAME);
    }
}

#[cfg(not(windows))]
mod etw {
    use std::sync::Arc;

    use super::GameMonitor;

    pub fn start(_monitor: Arc<GameMonitor>) {}

    pub fn stop() {}
}

pub fn stop_fps_session() {
    etw::stop();
}
