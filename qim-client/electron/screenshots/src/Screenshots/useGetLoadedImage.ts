import { useEffect, useRef, useState } from 'react';

export default function useGetLoadedImage(
  url?: string,
): HTMLImageElement | null {
  const [image, setImage] = useState<HTMLImageElement | null>(null);
  const prevUrlRef = useRef<string | null>(null);

  useEffect(() => {
    // 如果 url 没变化，复用已有的 image，避免重复解码大尺寸 dataURL
    if (url === prevUrlRef.current) {
      return;
    }
    prevUrlRef.current = url ?? null;

    if (url == null) {
      setImage(null);
      return;
    }

    // 不先置空 image，复用上一张图直到新图加载完成，避免白屏闪烁
    const $image = document.createElement('img');
    // 禁用自动解码，使用 syncDecode 手动控制时机
    $image.decoding = 'sync';

    const onLoad = () => setImage($image);
    const onError = () => setImage(null);

    $image.addEventListener('load', onLoad);
    $image.addEventListener('error', onError);
    $image.src = url;

    return () => {
      $image.removeEventListener('load', onLoad);
      $image.removeEventListener('error', onError);
    };
  }, [url]);

  return image;
}
