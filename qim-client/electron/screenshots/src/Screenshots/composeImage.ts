import type { Bounds, History } from './types';
import { HistoryItemType } from './types';

interface ComposeImageOpts {
  image?: HTMLImageElement | null;
  url?: string;
  width: number;
  height: number;
  history: History;
  bounds: Bounds;
}

export default function composeImage({
  image,
  url,
  width,
  height,
  history,
  bounds,
}: ComposeImageOpts): Promise<Blob> {
  return new Promise<Blob>((resolve, reject) => {
    const doCompose = (img: HTMLImageElement) => {
      const $canvas = document.createElement('canvas');
      const targetWidth = bounds.width * window.devicePixelRatio;
      const targetHeight = bounds.height * window.devicePixelRatio;
      $canvas.width = targetWidth;
      $canvas.height = targetHeight;

      const ctx = $canvas.getContext('2d');
      if (!ctx) {
        return reject(new Error('convert image to blob fail'));
      }

      const rx = img.naturalWidth / width;
      const ry = img.naturalHeight / height;

      ctx.imageSmoothingEnabled = true;
      ctx.imageSmoothingQuality = 'low';
      ctx.setTransform(
        window.devicePixelRatio,
        0,
        0,
        window.devicePixelRatio,
        0,
        0,
      );
      ctx.clearRect(0, 0, bounds.width, bounds.height);
      ctx.drawImage(
        img,
        bounds.x * rx,
        bounds.y * ry,
        bounds.width * rx,
        bounds.height * ry,
        0,
        0,
        bounds.width,
        bounds.height,
      );

      history.stack.slice(0, history.index + 1).forEach((item) => {
        if (item.type === HistoryItemType.Source) {
          item.draw(ctx, item);
        }
      });

      $canvas.toBlob((blob) => {
        if (!blob) {
          return reject(new Error('canvas toBlob fail'));
        }
        resolve(blob);
      }, 'image/png');
    };

    if (image) {
      doCompose(image);
      return;
    }

    if (!url) {
      return reject(new Error('composeImage: image or url is required'));
    }

    const img = new Image();
    img.addEventListener('load', () => doCompose(img));
    img.addEventListener('error', () =>
      reject(new Error('composeImage: failed to load image from url')),
    );
    img.src = url;
  });
}
