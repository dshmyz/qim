import type { ReactElement, PointerEvent as ReactPointerEvent } from "react";
import {
  memo,
  useCallback,
  useEffect,
  useLayoutEffect,
  useRef,
  useState,
} from "react";
import useBounds from "../hooks/useBounds";
import useStore from "../hooks/useStore";
import ScreenshotsMagnifier from "../ScreenshotsMagnifier";
import type { Point, Position } from "../types";
import getBoundsByPoints from "./getBoundsByPoints";
import "./index.less";

export default memo(function ScreenshotsBackground(): ReactElement | null {
  const { url, image, width, height } = useStore();
  const [bounds, boundsDispatcher] = useBounds();

  const elRef = useRef<HTMLDivElement>(null);
  const pointRef = useRef<Point | null>(null);
  // 用来判断鼠标是否移动过
  // 如果没有移动过位置，则pointerup时不更新
  const isMoveRef = useRef<boolean>(false);
  const [position, setPosition] = useState<Position | null>(null);

  // 使用 rAF 节流 pointermove，避免高频 getBoundingClientRect 和 setState
  const rafRef = useRef<number | null>(null);
  const pendingMoveRef = useRef<Point | null>(null);

  const updateBounds = useCallback(
    (p1: Point, p2: Point) => {
      if (!elRef.current) {
        return;
      }
      const { x, y } = elRef.current.getBoundingClientRect();

      boundsDispatcher.set(
        getBoundsByPoints(
          {
            x: p1.x - x,
            y: p1.y - y,
          },
          {
            x: p2.x - x,
            y: p2.y - y,
          },
          width,
          height
        )
      );
    },
    [width, height, boundsDispatcher]
  );

  const flushPointerMove = useCallback(() => {
    rafRef.current = null;
    const pending = pendingMoveRef.current;
    if (!pending || !pointRef.current) return;

    // 更新放大镜位置
    if (elRef.current) {
      const rect = elRef.current.getBoundingClientRect();
      if (
        pending.x < rect.left ||
        pending.y < rect.top ||
        pending.x > rect.right ||
        pending.y > rect.bottom
      ) {
        setPosition(null);
      } else {
        setPosition({
          x: pending.x - rect.x,
          y: pending.y - rect.y,
        });
      }
    }

    updateBounds(pointRef.current, pending);
    isMoveRef.current = true;
  }, [updateBounds]);

  const onPointerDown = useCallback(
    (e: ReactPointerEvent<HTMLDivElement>) => {
      // e.button 鼠标左键
      if (pointRef.current || bounds || e.button !== 0) {
        return;
      }
      pointRef.current = {
        x: e.clientX,
        y: e.clientY,
      };
      isMoveRef.current = false;
    },
    [bounds]
  );

  useEffect(() => {
    const onPointerMove = (e: PointerEvent) => {
      const pending = { x: e.clientX, y: e.clientY };
      pendingMoveRef.current = pending;

      if (!pointRef.current) {
        // 未拖拽时，只更新放大镜位置（也需要 rAF 节流）
        if (rafRef.current) return;
        rafRef.current = requestAnimationFrame(() => {
          rafRef.current = null;
          if (elRef.current) {
            const rect = elRef.current.getBoundingClientRect();
            if (
              pending.x < rect.left ||
              pending.y < rect.top ||
              pending.x > rect.right ||
              pending.y > rect.bottom
            ) {
              setPosition(null);
            } else {
              setPosition({
                x: pending.x - rect.x,
                y: pending.y - rect.y,
              });
            }
          }
        });
        return;
      }

      // 拖拽中，合并到下一帧处理
      if (!rafRef.current) {
        rafRef.current = requestAnimationFrame(flushPointerMove);
      }
    };

    const onPointerUp = (e: PointerEvent) => {
      if (!pointRef.current) {
        return;
      }

      if (isMoveRef.current) {
        updateBounds(pointRef.current, {
          x: e.clientX,
          y: e.clientY,
        });
      }
      pointRef.current = null;
      isMoveRef.current = false;
      pendingMoveRef.current = null;
    };
    window.addEventListener("pointermove", onPointerMove);
    window.addEventListener("pointerup", onPointerUp);

    return () => {
      window.removeEventListener("pointermove", onPointerMove);
      window.removeEventListener("pointerup", onPointerUp);
      if (rafRef.current) {
        cancelAnimationFrame(rafRef.current);
        rafRef.current = null;
      }
    };
  }, [updateBounds, flushPointerMove]);

  useLayoutEffect(() => {
    if (!image || bounds) {
      // 重置位置
      setPosition(null);
    }
  }, [image, bounds]);

  // 没有加载完不显示图片
  if (!url || !image) {
    return null;
  }

  return (
    <div
      ref={elRef}
      className="screenshots-background"
      onPointerDown={onPointerDown}
    >
      <img className="screenshots-background-image" src={url} />
      <div className="screenshots-background-mask" />
      {position && !bounds && (
        <ScreenshotsMagnifier x={position?.x} y={position?.y} />
      )}
    </div>
  );
});
