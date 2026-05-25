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
});
