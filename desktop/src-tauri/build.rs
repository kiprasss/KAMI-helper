use std::path::Path;

fn main() {
    println!("cargo::rustc-check-cfg=cfg(embedded_backend)");
    println!("cargo::rerun-if-changed=binaries/kami-backend.exe");

    let windows = std::env::var("CARGO_CFG_TARGET_OS").as_deref() == Ok("windows");
    if windows && Path::new("binaries/kami-backend.exe").exists() {
        println!("cargo::rustc-cfg=embedded_backend");
    }

    tauri_build::build()
}
