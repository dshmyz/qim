use base64::Engine;
use base64::engine::general_purpose::STANDARD;
use serde::{Deserialize, Serialize};
use std::fs;
use std::path::PathBuf;
use std::sync::atomic::{AtomicBool, Ordering};
use std::sync::Mutex;
use std::thread;
use std::time::Duration;
use tauri::{AppHandle, Emitter, Manager, WindowEvent};
use tauri::tray::{MouseButton, MouseButtonState, TrayIconBuilder, TrayIconEvent};
use tauri::menu::{MenuBuilder, MenuItemBuilder, PredefinedMenuItem};
use tauri_plugin_global_shortcut::GlobalShortcutExt;
use tauri::WebviewWindowBuilder;
use tauri::WebviewUrl;
use xcap::Monitor;

static IS_FLASHING: AtomicBool = AtomicBool::new(false);
static SCREENSHOT_CAPTURE: Mutex<Option<ScreenshotCapture>> = Mutex::new(None);

#[derive(Serialize, Deserialize, Clone)]
#[serde(rename_all = "camelCase")]
struct ScreenshotCapture {
  file_path: String,
  data_url: String,
  display_info: String,
  hide_window: bool,
  was_maximized: bool,
  full_res_data: Vec<u8>,
  full_res_w: u32,
  full_res_h: u32,
}

#[derive(Serialize)]
#[serde(rename_all = "camelCase")]
struct FileDialogResult {
  canceled: bool,
  file_path: String,
}

#[derive(Serialize)]
#[serde(rename_all = "camelCase")]
struct AppInfo {
  version: String,
  platform: String,
  user_data_dir: String,
}

#[derive(Serialize)]
#[serde(rename_all = "camelCase")]
struct UpdateInfo {
  available: bool,
  version: String,
  url: String,
}

#[derive(Serialize)]
#[serde(rename_all = "camelCase")]
struct ScreenSource {
  id: String,
  name: String,
  width: u32,
  height: u32,
  x: i32,
  y: i32,
}

#[derive(Deserialize)]
#[serde(rename_all = "camelCase")]
struct CropBounds {
  x: i32,
  y: i32,
  width: i32,
  height: i32,
}

fn crop_full_res(bounds_json: &str) -> Result<Vec<u8>, String> {
  let bounds: CropBounds = serde_json::from_str(bounds_json).map_err(|e| e.to_string())?;
  let guard = SCREENSHOT_CAPTURE.lock().unwrap();
  let capture = guard.as_ref().ok_or("no screenshot")?;
  let (full_w, full_h) = (capture.full_res_w, capture.full_res_h);
  let img = image::ImageBuffer::<image::Rgba<u8>, Vec<u8>>::from_raw(
    full_w, full_h, capture.full_res_data.clone()
  ).ok_or("invalid full-res buffer")?;
  drop(guard);

  // Scale bounds 2x (display is half-res, original is full-res)
  let (x, y, w, h) = (bounds.x * 2, bounds.y * 2, bounds.width * 2, bounds.height * 2);
  let cropped = image::DynamicImage::ImageRgba8(img).crop_imm(
    x.max(0) as u32, y.max(0) as u32,
    (w as u32).min(full_w), (h as u32).min(full_h),
  );

  let mut png_bytes = std::io::Cursor::new(Vec::new());
  cropped.write_to(&mut png_bytes, image::ImageFormat::Png).map_err(|e| e.to_string())?;
  Ok(png_bytes.into_inner())
}

#[derive(Deserialize)]
#[serde(rename_all = "camelCase")]
struct OpenFileDialogOptions {
  title: Option<String>,
  default_dir: Option<String>,
  filters: Option<Vec<String>>,
}

fn user_data_dir() -> PathBuf {
  let mut dir = dirs::home_dir().unwrap_or_else(|| PathBuf::from("."));
  dir.push(".qim");
  dir.push("app");
  let _ = fs::create_dir_all(&dir);
  dir
}

fn avatar_cache_dir() -> PathBuf {
  let mut dir = user_data_dir();
  dir.push("avatar-cache");
  let _ = fs::create_dir_all(&dir);
  dir
}

fn show_main_window(app: &AppHandle) {
  if let Some(w) = app.get_webview_window("main") {
    let _ = w.show();
    let _ = w.set_always_on_top(true);
    let _ = w.set_always_on_top(false);
  }
}

fn setup_tray(app: &AppHandle) -> Result<(), Box<dyn std::error::Error>> {
  let show_item = MenuItemBuilder::with_id("show-window", "显示主窗口").build(app)?;
  let quit_item = MenuItemBuilder::with_id("quit-app", "退出").build(app)?;
  let sep = PredefinedMenuItem::separator(app)?;

  let menu = MenuBuilder::new(app)
    .item(&show_item)
    .item(&sep)
    .item(&quit_item)
    .build()?;

  TrayIconBuilder::with_id("main-tray")
    .icon(app.default_window_icon().cloned().unwrap())
    .menu(&menu)
    .show_menu_on_left_click(false)
    .on_menu_event(|app, event| {
      match event.id.as_ref() {
        "show-window" => show_main_window(app),
        "quit-app" => app.exit(0),
        _ => {}
      }
    })
    .on_tray_icon_event(|tray, event| {
      if let TrayIconEvent::Click { button: MouseButton::Left, button_state: MouseButtonState::Up, .. } = event {
        show_main_window(tray.app_handle());
      }
    })
    .build(app)?;

  Ok(())
}

fn setup_shortcuts(app: &AppHandle) -> Result<(), Box<dyn std::error::Error>> {
  app.global_shortcut().on_shortcut("CmdOrControl+Shift+A", |app, _shortcut, event| {
    if event.state == tauri_plugin_global_shortcut::ShortcutState::Pressed {
      start_screenshot_logic(app.clone(), true);
    }
  })?;
  Ok(())
}

fn setup_close_to_tray(app: &AppHandle) {
  if let Some(w) = app.get_webview_window("main") {
    let app_clone = app.clone();
    w.on_window_event(move |event| {
      if let WindowEvent::CloseRequested { api, .. } = event {
        api.prevent_close();
        if let Some(w) = app_clone.get_webview_window("main") {
          let _ = w.hide();
        }
      }
    });
  }
}

fn start_screenshot_logic(app: AppHandle, hide_window: bool) {
  let was_maximized = app.get_webview_window("main")
    .and_then(|w| w.is_maximized().ok())
    .unwrap_or(false);

  if hide_window {
    if let Some(w) = app.get_webview_window("main") {
      let _ = w.hide();
    }
  }

  // Create the separate overlay window (like Electron's BrowserWindow in kiosk mode)
  let app_for_overlay = app.clone();
  let overlay_result = WebviewWindowBuilder::new(
    &app_for_overlay,
    "screenshot-overlay",
    WebviewUrl::App("/src/screenshots/index.html".into()),
  )
    .title("截图")
    .decorations(false)
    .always_on_top(true)
    .skip_taskbar(true)
    .resizable(false)
    .maximized(true)
    .visible(false) // Show after screenshot is captured
    .build();

  let app_clone = app.clone();
  thread::spawn(move || {
    let mut log = String::new();
    let _ = fs::write("/tmp/qim_debug.log", "thread started
");

    if hide_window {
      thread::sleep(Duration::from_millis(200));
    }

    // Save to app data dir
    let mut screenshot_dir = user_data_dir();
    screenshot_dir.push("screenshots");
    let _ = fs::create_dir_all(&screenshot_dir);
    let screenshot_file = screenshot_dir.join("latest.jpg");
    let screenshot_file_str = screenshot_file.to_string_lossy().to_string();

    let mut data_url = String::new();
    let mut display_info = String::new();
    let mut full_res_buf = Vec::new();
    let (mut full_w, mut full_h) = (0u32, 0u32);
    let mut success = false;

    let t0 = std::time::Instant::now();
    let monitors = Monitor::all().unwrap_or_default();
    if let Some(monitor) = monitors.first() {
      let scale = monitor.scale_factor().unwrap_or(1.0) as f64;
      let phys_w = monitor.width().unwrap_or(1920) as f64;
      let phys_h = monitor.height().unwrap_or(1080) as f64;
      let logical_w = (phys_w / scale).round() as u32;
      let logical_h = (phys_h / scale).round() as u32;
      display_info = serde_json::json!({"width": logical_w, "height": logical_h}).to_string();

      if let Ok(image) = monitor.capture_image() {
        log.push_str(&format!("capture_image: {}ms ({}x{})\n", t0.elapsed().as_millis(), image.width(), image.height()));
        (full_w, full_h) = (image.width(), image.height());

        // Keep full-res raw data in memory for high-quality crop output
        full_res_buf = image.as_raw().to_vec();

        // Fast 2x downsample for display overlay
        let t1 = std::time::Instant::now();
        let (sw, sh) = (image.width(), image.height());
        let (dw, dh) = (sw / 2, sh / 2);
        let src = image.as_raw();
        let mut dst = vec![0u8; (dw * dh * 3) as usize];
        for y in 0..dh {
          let sy = (y * 2) as usize;
          let row0 = &src[sy * sw as usize * 4..];
          let row1 = &src[(sy + 1) * sw as usize * 4..];
          for x in 0..dw {
            let sx = (x * 2) as usize;
            let i = (y * dw + x) as usize * 3;
            for c in 0..3 {
              let sum = row0[sx * 4 + c] as u16
                + row0[(sx + 1) * 4 + c] as u16
                + row1[sx * 4 + c] as u16
                + row1[(sx + 1) * 4 + c] as u16;
              dst[i + c] = (sum / 4) as u8;
            }
          }
        }
        log.push_str(&format!("downsample2x: {}ms ({}x{} -> {}x{})\n", t1.elapsed().as_millis(), sw, sh, dw, dh));

        // JPEG via DynamicImage::write_to (faster than JpegEncoder::encode)
        let t2 = std::time::Instant::now();
        let rgb_buf = image::ImageBuffer::<image::Rgb<u8>, Vec<u8>>::from_raw(dw, dh, dst).unwrap();
        let mut jpg_bytes = std::io::Cursor::new(Vec::new());
        if image::DynamicImage::ImageRgb8(rgb_buf).write_to(&mut jpg_bytes, image::ImageFormat::Jpeg).is_ok() {
          log.push_str(&format!("jpeg(half-res): {}ms\n", t2.elapsed().as_millis()));
          let t3 = std::time::Instant::now();
          let jpg_data = jpg_bytes.into_inner();
          let _ = fs::write(&screenshot_file, &jpg_data);
          data_url = format!("data:image/jpeg;base64,{}", STANDARD.encode(&jpg_data));
          log.push_str(&format!("file+base64: {}ms jpg={}KB\n", t3.elapsed().as_millis(), jpg_data.len()/1024));
          success = true;
        }
      }
    }

    if success {
      let capture = ScreenshotCapture {
        file_path: screenshot_file_str.clone(),
        data_url: data_url.clone(),
        display_info: display_info.clone(),
        hide_window,
        was_maximized,
        full_res_data: full_res_buf,
        full_res_w: full_w,
        full_res_h: full_h,
      };
      {
        let mut guard = SCREENSHOT_CAPTURE.lock().unwrap();
        *guard = Some(capture);
      }

      if let Ok(overlay) = overlay_result {
        let t4 = std::time::Instant::now();
        let escaped_data_url = data_url.replace('\\', "\\\\").replace('\'', "\\'");
        let escaped_display = display_info.replace('\\', "\\\\").replace('\'', "\\'");
        let js = format!(
          "window.__SCREENSHOT_DATA__ = {{imageUrl: '{}', displayInfo: '{}'}}; window.dispatchEvent(new CustomEvent('screenshot-data-ready'))",
          escaped_data_url, escaped_display
        );
        let _ = overlay.eval(&js);
        log.push_str(&format!("eval: {}ms data_url_len={}\n", t4.elapsed().as_millis(), data_url.len()));
        let t5 = std::time::Instant::now();
        thread::sleep(Duration::from_millis(50));
        let _ = overlay.show();
        let _ = overlay.set_focus();
        log.push_str(&format!("show+focus: {}ms\n", t5.elapsed().as_millis()));
        log.push_str(&format!("TOTAL: {}ms\n", t0.elapsed().as_millis()));
      }
      let _ = fs::write("/tmp/qim_debug.log", &log);
    } else {
      log.push_str("capture FAILED\n");
      let _ = fs::write("/tmp/qim_debug.log", &log);
      // Capture failed: show main window back
      if let Some(w) = app_clone.get_webview_window("main") {
        let _ = w.show();
      }
    }
  });
}

fn close_screenshot_overlay_and_restore(app: &AppHandle) {
  // Close the overlay window
  if let Some(w) = app.get_webview_window("screenshot-overlay") {
    let _ = w.close();
  }

  // Show and restore main window
  if let Some(w) = app.get_webview_window("main") {
    let _ = w.show();
    let _ = w.set_focus();
    let _ = w.set_always_on_top(false);

    let was_maximized = SCREENSHOT_CAPTURE.lock().unwrap()
      .as_ref()
      .map(|c| c.was_maximized)
      .unwrap_or(false);

    if !was_maximized {
      let _ = w.unmaximize();
    }
  }

  // Clear stored capture data
  {
    let mut guard = SCREENSHOT_CAPTURE.lock().unwrap();
    *guard = None;
  }
}

fn spawn_flash_timer(app: AppHandle) {
  thread::spawn(move || {
    let tray = app.tray_by_id("main-tray");
    while IS_FLASHING.load(Ordering::SeqCst) {
      if let Some(t) = &tray {
        let _ = t.set_visible(false);
      }
      thread::sleep(Duration::from_millis(500));
      if let Some(t) = &tray {
        let _ = t.set_visible(true);
      }
      thread::sleep(Duration::from_millis(500));
    }
    if let Some(t) = &tray {
      let _ = t.set_visible(true);
    }
  });
}

#[tauri::command]
fn minimize_window(app: AppHandle) -> Result<(), String> {
  app.get_webview_window("main")
    .ok_or("window not found".to_string())?
    .minimize()
    .map_err(|e| e.to_string())
}

#[tauri::command]
fn maximize_window(app: AppHandle) -> Result<(), String> {
  let window = app.get_webview_window("main").ok_or("window not found".to_string())?;
  let is_max = window.is_maximized().map_err(|e| e.to_string())?;
  if is_max {
    window.unmaximize().map_err(|e| e.to_string())
  } else {
    window.maximize().map_err(|e| e.to_string())
  }
}

#[tauri::command]
fn close_window(app: AppHandle) -> Result<(), String> {
  app.get_webview_window("main")
    .ok_or("window not found".to_string())?
    .hide()
    .map_err(|e| e.to_string())
}

#[tauri::command]
fn is_maximized(app: AppHandle) -> Result<bool, String> {
  app.get_webview_window("main")
    .ok_or("window not found".to_string())?
    .is_maximized()
    .map_err(|e| e.to_string())
}

#[tauri::command]
fn open_external(url: String) -> Result<(), String> {
  tauri_plugin_opener::open_url(url, None::<&str>).map_err(|e| e.to_string())
}

#[tauri::command]
async fn open_file_dialog(opts: OpenFileDialogOptions, app: AppHandle) -> Result<FileDialogResult, String> {
  use tauri_plugin_dialog::DialogExt;

  let mut builder = app.dialog().file();
  if let Some(title) = &opts.title {
    builder = builder.set_title(title);
  }
  if let Some(filters) = &opts.filters {
    if filters.len() >= 2 {
      let extensions: Vec<&str> = filters[1..].iter().map(|s| s.as_str()).collect();
      builder = builder.add_filter(&filters[0], &extensions);
    }
  }

  let path = builder.blocking_pick_file();
  match path {
    Some(p) => Ok(FileDialogResult {
      canceled: false,
      file_path: p.to_string(),
    }),
    None => Ok(FileDialogResult {
      canceled: true,
      file_path: String::new(),
    }),
  }
}

#[tauri::command]
fn save_file_as(file_name: String, data: Vec<u8>) -> Result<FileDialogResult, String> {
  let mut path = dirs::download_dir().unwrap_or_else(|| user_data_dir());
  path.push(file_name);
  fs::write(&path, data).map_err(|e| e.to_string())?;
  Ok(FileDialogResult {
    canceled: false,
    file_path: path.to_string_lossy().to_string(),
  })
}

#[tauri::command]
fn download_file(file_name: String, data: Vec<u8>, save_dir: String) -> Result<FileDialogResult, String> {
  let mut target_dir = if save_dir.is_empty() {
    dirs::download_dir().unwrap_or_else(|| user_data_dir())
  } else {
    PathBuf::from(save_dir)
  };
  fs::create_dir_all(&target_dir).map_err(|e| e.to_string())?;
  target_dir.push(file_name);
  fs::write(&target_dir, data).map_err(|e| e.to_string())?;

  Ok(FileDialogResult {
    canceled: false,
    file_path: target_dir.to_string_lossy().to_string(),
  })
}

#[tauri::command]
fn get_app_info() -> AppInfo {
  AppInfo {
    version: "1.0.0".to_string(),
    platform: std::env::consts::OS.to_string(),
    user_data_dir: user_data_dir().to_string_lossy().to_string(),
  }
}

#[tauri::command]
async fn cache_avatar(avatar_url: String) -> Result<String, String> {
  let bytes = reqwest::get(&avatar_url)
    .await
    .map_err(|e| e.to_string())?
    .bytes()
    .await
    .map_err(|e| e.to_string())?;

  let ext = avatar_url
    .split('.')
    .next_back()
    .filter(|v| v.len() <= 10)
    .map(|v| format!(".{v}"))
    .unwrap_or_else(|| ".png".to_string());

  let digest = md5::compute(avatar_url.as_bytes());
  let mut path = avatar_cache_dir();
  path.push(format!("{:x}{ext}", digest));

  if !path.exists() {
    fs::write(&path, &bytes).map_err(|e| e.to_string())?;
  }

  Ok(path.to_string_lossy().to_string())
}

#[tauri::command]
fn cleanup_avatar_cache(max_age_days: i64) -> Result<(), String> {
  let days = if max_age_days <= 0 { 7 } else { max_age_days };
  let max_age = std::time::Duration::from_secs((days as u64) * 24 * 60 * 60);
  let now = std::time::SystemTime::now();

  let dir = avatar_cache_dir();
  if let Ok(entries) = fs::read_dir(dir) {
    for entry in entries.flatten() {
      if let Ok(meta) = entry.metadata() {
        if let Ok(modified) = meta.modified() {
          if now.duration_since(modified).unwrap_or_default() > max_age {
            let _ = fs::remove_file(entry.path());
          }
        }
      }
    }
  }

  Ok(())
}

#[tauri::command]
fn flash_tray(app: AppHandle, enabled: bool) -> Result<(), String> {
  IS_FLASHING.store(enabled, Ordering::SeqCst);
  app.emit("tray-flash", enabled).map_err(|e| e.to_string())?;

  if enabled {
    spawn_flash_timer(app);
  }

  Ok(())
}

#[tauri::command]
fn check_for_updates() -> UpdateInfo {
  UpdateInfo {
    available: false,
    version: String::new(),
    url: String::new(),
  }
}

#[tauri::command]
fn download_update() -> UpdateInfo {
  UpdateInfo {
    available: false,
    version: String::new(),
    url: String::new(),
  }
}

#[tauri::command]
fn get_screen_sources() -> Vec<ScreenSource> {
  let monitors = Monitor::all().unwrap_or_default();
  monitors.iter().map(|m| ScreenSource {
    id: format!("screen:{}", m.id().unwrap_or(0)),
    name: m.name().unwrap_or_else(|_| "显示器".into()),
    width: m.width().unwrap_or(0),
    height: m.height().unwrap_or(0),
    x: m.x().unwrap_or(0),
    y: m.y().unwrap_or(0),
  }).collect()
}

#[tauri::command]
fn start_screenshot(app: AppHandle, hide_window: bool) -> Result<(), String> {
  let _ = fs::write("/tmp/qim_debug.log", "start_screenshot COMMAND called
");
  start_screenshot_logic(app, hide_window);
  Ok(())
}

#[tauri::command]
fn get_screenshot_capture() -> Option<ScreenshotCapture> {
  SCREENSHOT_CAPTURE.lock().unwrap().clone()
}

#[tauri::command]
fn read_screenshot_file() -> Option<ScreenshotCapture> {
  let guard = SCREENSHOT_CAPTURE.lock().unwrap();
  let capture = guard.as_ref()?;
  // Re-read the file and return fresh data URL (convertFileSrc fallback)
  if let Ok(png_data) = fs::read(&capture.file_path) {
    let data_url = format!("data:image/png;base64,{}", STANDARD.encode(&png_data));
    Some(ScreenshotCapture {
      data_url,
      ..capture.clone()
    })
  } else {
    Some(capture.clone())
  }
}

#[tauri::command]
fn ok_screenshot_overlay(app: AppHandle, data_url: String, bounds_json: String) -> Result<(), String> {
  close_screenshot_overlay_and_restore(&app);
  // Crop from full-res original for Retina-quality output
  let final_url = match crop_full_res(&bounds_json) {
    Ok(png) => format!("data:image/png;base64,{}", STANDARD.encode(&png)),
    Err(_e) => data_url
  };
  app.emit_to("main", "screenshot-taken", vec![final_url, bounds_json])
    .map_err(|e| e.to_string())
}

#[tauri::command]
fn cancel_screenshot_overlay(app: AppHandle) -> Result<(), String> {
  close_screenshot_overlay_and_restore(&app);
  Ok(())
}

#[tauri::command]
fn save_screenshot_overlay(app: AppHandle, _data: Vec<u8>, bounds_json: String) -> Result<FileDialogResult, String> {
  close_screenshot_overlay_and_restore(&app);

  let png_data = crop_full_res(&bounds_json)?;

  let mut path = dirs::download_dir().unwrap_or_else(|| user_data_dir());
  let ts = chrono::Local::now().format("%Y%m%d_%H%M%S").to_string();
  path.push(format!("screenshot_{ts}.png"));
  fs::write(&path, &png_data).map_err(|e| e.to_string())?;

  let file_path = path.to_string_lossy().to_string();
  app.emit_to("main", "screenshot-taken", vec![
    format!("data:image/png;base64,{}", STANDARD.encode(&png_data)),
    bounds_json,
  ]).map_err(|e| e.to_string())?;

  Ok(FileDialogResult {
    canceled: false,
    file_path,
  })
}

#[tauri::command]
fn cancel_screenshot(app: AppHandle) -> Result<(), String> {
  app.emit("screenshot-cancel", ()).map_err(|e| e.to_string())
}

#[tauri::command]
fn complete_screenshot(app: AppHandle, data_url: String, bounds_json: String) -> Result<(), String> {
  app.emit("screenshot-taken", vec![data_url, bounds_json])
    .map_err(|e| e.to_string())
}

#[tauri::command]
fn save_screenshot(data: Vec<u8>, _bounds_json: String) -> Result<FileDialogResult, String> {
  let mut path = dirs::download_dir().unwrap_or_else(|| user_data_dir());
  let ts = chrono::Local::now().format("%Y%m%d_%H%M%S").to_string();
  path.push(format!("screenshot_{ts}.png"));
  fs::write(&path, data).map_err(|e| e.to_string())?;
  Ok(FileDialogResult {
    canceled: false,
    file_path: path.to_string_lossy().to_string(),
  })
}

#[tauri::command]
fn get_platform() -> String {
  std::env::consts::OS.to_string()
}

#[tauri::command]
fn quit_app(app: AppHandle) {
  app.exit(0);
}

#[cfg_attr(mobile, tauri::mobile_entry_point)]
pub fn run() {
  tauri::Builder::default()
    .plugin(tauri_plugin_opener::init())
    .plugin(tauri_plugin_dialog::init())
    .plugin(tauri_plugin_global_shortcut::Builder::new().build())
    .setup(|app| {
      setup_tray(&app.handle())?;
      setup_shortcuts(&app.handle())?;
      setup_close_to_tray(&app.handle());
      Ok(())
    })
    .invoke_handler(tauri::generate_handler![
      minimize_window,
      maximize_window,
      close_window,
      is_maximized,
      open_external,
      open_file_dialog,
      save_file_as,
      download_file,
      get_app_info,
      cache_avatar,
      cleanup_avatar_cache,
      flash_tray,
      check_for_updates,
      download_update,
      get_screen_sources,
      start_screenshot,
      get_screenshot_capture,
      read_screenshot_file,
      ok_screenshot_overlay,
      cancel_screenshot_overlay,
      save_screenshot_overlay,
      cancel_screenshot,
      complete_screenshot,
      save_screenshot,
      get_platform,
      quit_app
    ])
    .run(tauri::generate_context!())
    .expect("error while running tauri application");
}