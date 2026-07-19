# Product name build metadata

## Goal

Use the Electron build metadata field `build.productName` as the single source for the renderer's `APP_CONFIG.productName`.

## Design

- `vite.config.ts` exposes `pkg.build.productName` as a compile-time `__APP_PRODUCT_NAME__` constant.
- `vite-env.d.ts` declares that constant for TypeScript.
- `appConfig.ts` assigns `productName` from `__APP_PRODUCT_NAME__`; `getProductName()` and all of its consumers remain unchanged.

## Error handling and verification

`build.productName` is required by the existing Electron packaging configuration, so no secondary fallback is introduced. Update the existing Vitest configuration's globals and run the focused configuration tests plus type checking.
