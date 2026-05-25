import type { ReactElement } from 'react';
import './icons/iconfont.less';
import './screenshots.less';
import type { Position } from './types';
import type { Lang } from './zh_CN';
export interface ScreenshotsProps {
    url?: string;
    width: number;
    height: number;
    lang?: Partial<Lang>;
    initialPosition?: Position;
    className?: string;
    [key: string]: unknown;
}
export default function Screenshots({ url, width, height, lang, initialPosition, className, ...props }: ScreenshotsProps): ReactElement;
