import { describe, expect, it } from 'vitest';
import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';

function extractRule(css: string, selector: string): string {
  const match = css.match(new RegExp(`${selector.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')}\\s*\\{([\\s\\S]*?)\\n\\}`));
  return match?.[1] ?? '';
}

describe('screenshots selection layering', () => {
  it('renders the selection canvas above the dimmed background mask', () => {
    const backgroundStyle = readFileSync(
      resolve(__dirname, '../../electron/screenshots/src/Screenshots/ScreenshotsBackground/index.less'),
      'utf8'
    );
    const canvasStyle = readFileSync(
      resolve(__dirname, '../../electron/screenshots/src/Screenshots/ScreenshotsCanvas/index.less'),
      'utf8'
    );

    const backgroundMaskRule = extractRule(backgroundStyle, '&-mask');
    const canvasRule = extractRule(canvasStyle, '.screenshots-canvas');

    const backgroundMaskZIndex = Number(backgroundMaskRule.match(/z-index:\s*(\d+)/)?.[1] ?? 0);
    const canvasZIndex = Number(canvasRule.match(/z-index:\s*(\d+)/)?.[1] ?? 0);

    expect(canvasZIndex).toBeGreaterThan(backgroundMaskZIndex);
  });

  it('renders the operations toolbar above the selection canvas', () => {
    const operationsStyle = readFileSync(
      resolve(__dirname, '../../electron/screenshots/src/Screenshots/ScreenshotsOperations/index.less'),
      'utf8'
    );
    const canvasStyle = readFileSync(
      resolve(__dirname, '../../electron/screenshots/src/Screenshots/ScreenshotsCanvas/index.less'),
      'utf8'
    );

    const operationsRule = extractRule(operationsStyle, '.screenshots-operations');
    const canvasRule = extractRule(canvasStyle, '.screenshots-canvas');

    const operationsZIndex = Number(operationsRule.match(/z-index:\s*(\d+)/)?.[1] ?? 0);
    const canvasZIndex = Number(canvasRule.match(/z-index:\s*(\d+)/)?.[1] ?? 0);

    expect(operationsZIndex).toBeGreaterThan(canvasZIndex);
  });

  it('builds the transparent mask hole into the Electron screenshot page', () => {
    const electronHtml = readFileSync(
      resolve(__dirname, '../../electron/screenshots/src/dist/electron.html'),
      'utf8'
    );
    const electronScript = electronHtml.match(/static\/js\/(electron\.[^"]+\.js)/)?.[1];

    expect(electronScript).toBeTruthy();

    const electronBundle = readFileSync(
      resolve(__dirname, `../../electron/screenshots/src/dist/static/js/${electronScript}`),
      'utf8'
    );

    expect(electronBundle).toContain('boxShadow');
    expect(electronBundle).toContain('transparent');
  });

  it('sends the captured image to the screenshot page before showing the overlay window', () => {
    const screenshotRuntime = readFileSync(
      resolve(__dirname, '../../electron/screenshots/lib/index.cjs'),
      'utf8'
    );

    const sendCaptureIndex = screenshotRuntime.indexOf("this.$view.webContents.send('SCREENSHOTS:capture'");
    const showOverlayIndex = screenshotRuntime.indexOf('yield this.showCaptureWindow(display)');

    expect(sendCaptureIndex).toBeGreaterThan(-1);
    expect(showOverlayIndex).toBeGreaterThan(-1);
    expect(sendCaptureIndex).toBeLessThan(showOverlayIndex);
  });

  it('reports screenshot startCapture failures back to the renderer', () => {
    const mainProcess = readFileSync(
      resolve(__dirname, '../../electron/main.js'),
      'utf8'
    );

    expect(mainProcess).toContain("sendScreenshotError('截图失败，请检查屏幕录制权限或稍后重试', err)");
    expect(mainProcess).toContain("mainWindow.webContents.send('screenshot-error', { message, code: errorCode, diagnostics })");
  });

  it('checks macOS screen recording permission before starting screenshot capture', () => {
    const mainProcess = readFileSync(
      resolve(__dirname, '../../electron/main.js'),
      'utf8'
    );

    expect(mainProcess).toContain('systemPreferences.getMediaAccessStatus(\'screen\')');
    expect(mainProcess).toContain('openSystemPreferences(\'security\', \'Privacy_ScreenCapture\')');
    expect(mainProcess).toContain("code: 'screen_permission_denied'");
    expect(mainProcess).toContain('请在系统设置中允许 QIM 进行屏幕录制');
  });

  it('wraps screenshot requests with timeout and diagnostic metadata', () => {
    const mainProcess = readFileSync(
      resolve(__dirname, '../../electron/main.js'),
      'utf8'
    );

    expect(mainProcess).toContain('const SCREENSHOT_CAPTURE_TIMEOUT_MS = 12000');
    expect(mainProcess).toContain("function getScreenshotDiagnostics()");
    expect(mainProcess).toContain("sessionType: process.env.XDG_SESSION_TYPE || 'unknown'");
    expect(mainProcess).toContain("function withScreenshotTimeout(capturePromise)");
    expect(mainProcess).toContain("code: 'capture_timeout'");
    expect(mainProcess).toContain("mainWindow.webContents.send('screenshot-error', { message, code: errorCode, diagnostics })");
    expect(mainProcess).toContain("startScreenshotCapture({ hideMainWindow: false })");
    expect(mainProcess).toContain("startScreenshotCapture({ hideMainWindow: true })");
  });

  it('guards renderer screenshot requests with a visible capture state', () => {
    const chatWindow = readFileSync(
      resolve(__dirname, '../../src/components/chat/ChatWindow.vue'),
      'utf8'
    );

    expect(chatWindow).toContain("type ScreenshotStatus = 'idle' | 'preparing' | 'capturing' | 'processing' | 'failed'");
    expect(chatWindow).toContain('const screenshotStatus = ref<ScreenshotStatus>(\'idle\')');
    expect(chatWindow).toContain('const isScreenshotBusy = computed(() =>');
    expect(chatWindow).toContain("if (isScreenshotBusy.value)");
    expect(chatWindow).toContain("screenshotStatus.value = 'preparing'");
    expect(chatWindow).toContain("screenshotStatus.value = 'processing'");
    expect(chatWindow).toContain("screenshotStatus.value = 'failed'");
    expect(chatWindow).toContain("screenshotStatus.value = 'idle'");
  });

  it('exposes Linux screenshot overlay knobs for compositor tuning', () => {
    const screenshotRuntime = readFileSync(
      resolve(__dirname, '../../electron/screenshots/lib/index.cjs'),
      'utf8'
    );
    const mainProcess = readFileSync(
      resolve(__dirname, '../../electron/main.js'),
      'utf8'
    );

    expect(screenshotRuntime).toContain('function getLinuxOverlayOptions()');
    expect(screenshotRuntime).toContain("process.env.QIM_SCREENSHOT_LINUX_SHOW_MODE || 'show'");
    expect(screenshotRuntime).toContain('QIM_SCREENSHOT_LINUX_PAINT_DELAY_MS');
    expect(screenshotRuntime).toContain('QIM_SCREENSHOT_LINUX_TRANSPARENT');
    expect(screenshotRuntime).toContain("this.$win.showInactive()");
    expect(screenshotRuntime).toContain('this._overlayOptions.paintDelayMs');
    expect(mainProcess).toContain('screenshotOverlay: screenshotInstance?.getOverlayDiagnostics?.() || null');
  });
});
