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
    // 异步解码避免主线程阻塞，大图(4K+)数据量可达 30MB+，同步解码会冻结 UI
    $image.decoding = 'async';

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
