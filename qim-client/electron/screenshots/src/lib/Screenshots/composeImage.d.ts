import type { Bounds, History } from './types';
interface ComposeImageOpts {
    image?: HTMLImageElement | null;
    url?: string;
    width: number;
    height: number;
    history: History;
    bounds: Bounds;
}
export default function composeImage({ image, url, width, height, history, bounds, }: ComposeImageOpts): Promise<Blob>;
export {};
