// Prevents additional console window on Windows in release, DO NOT REMOVE!!
#![cfg_attr(not(debug_assertions), windows_subsystem = "windows")]

// Learn more about Tauri commands at https://tauri.app/v1/guides/features/command
#[tauri::command]
fn greet(name: &str) -> String {
    format!("Hello, {}! You've been greeted from Rust!", name)
}


use tokio::task;
use std::process::Command;
use tauri::Manager;

// #[tauri::command]
// async fn run_background() {
//   let child = task::spawn_blocking(|| {
//     let mut child = Command::new("my_program")
//         .spawn()
//         .expect("failed to execute process");

//     child.wait().expect("failed to wait on child");
//   });

//   // child runs in the background
//   // we can continue with other work

//   child.await;
// }

// https://tauri.app/v1/guides/features/events/#window-specific-events-1

#[tauri::command(async)]
async fn run_background() {
    let child = task::spawn_blocking(|| {
        Command::new("ls").output().expect("failed to execute process");
        // run child process
    });

    // continue with other work

    let _ = child.await; // wait for background process
}

fn main() {
    tauri::Builder::default()
        .setup(|app| {
            #[cfg(debug_assertions)] // only include this code on debug builds
            {
                let window = app.get_window("main").unwrap();
                window.open_devtools();
            }
            Ok(())
        })
        .invoke_handler(tauri::generate_handler![greet, run_background])
        .run(tauri::generate_context!())
        .expect("error while running tauri application");
}

// fn main() {
//     let mut child = Command::new(
//         "C:\\Users\\Piotrek\\Projects\\dispel-multi\\launcher\\src-tauri\\src-tauri.exe",
//     )
//     .arg("file.txt")
//     .spawn()
//     .expect("failed to execute process");
//
//     let ecode = child.wait().expect("failed to wait on child");
//
//     println!("Hello from Rust {:?}", ecode);
// }

// #[tauri::command]
// async fn run_process() {
//   let mut child = Command::new("my_program")
//         // .stdout(std::process::Stdio::piped())

//       .spawn()
//       .expect("failed to execute process");

// // let stdout = child.stdout.take().expect("failed to get stdout");
// //   let mut lines = std::io::BufReader::new(stdout).lines();
// // let output = lines.next_line().await;
//   // return output
// //   output.unwrap()

//   let _ = child.wait().await;