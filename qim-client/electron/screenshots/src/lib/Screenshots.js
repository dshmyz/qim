import "./rslib-runtime.js";
import react, { cloneElement, forwardRef, memo, useCallback, useContext, useEffect, useImperativeHandle, useLayoutEffect, useMemo, useRef, useState } from "react";
import { createPortal } from "react-dom";
function composeImage({ image, url, width, height, history, bounds }) {
    return new Promise((resolve, reject)=>{
        const doCompose = (img)=>{
            const $canvas = document.createElement('canvas');
            const targetWidth = bounds.width * window.devicePixelRatio;
            const targetHeight = bounds.height * window.devicePixelRatio;
            $canvas.width = targetWidth;
            $canvas.height = targetHeight;
            const ctx = $canvas.getContext('2d');
            if (!ctx) return reject(new Error('convert image to blob fail'));
            const rx = img.naturalWidth / width;
            const ry = img.naturalHeight / height;
            ctx.imageSmoothingEnabled = true;
            ctx.imageSmoothingQuality = 'low';
            ctx.setTransform(window.devicePixelRatio, 0, 0, window.devicePixelRatio, 0, 0);
            ctx.clearRect(0, 0, bounds.width, bounds.height);
            ctx.drawImage(img, bounds.x * rx, bounds.y * ry, bounds.width * rx, bounds.height * ry, 0, 0, bounds.width, bounds.height);
            history.stack.slice(0, history.index + 1).forEach((item)=>{
                if (1 === item.type) item.draw(ctx, item);
            });
            $canvas.toBlob((blob)=>{
                if (!blob) return reject(new Error('canvas toBlob fail'));
                resolve(blob);
            }, 'image/png');
        };
        if (image) return void doCompose(image);
        if (!url) return reject(new Error('composeImage: image or url is required'));
        const img = new Image();
        img.addEventListener('load', ()=>doCompose(img));
        img.addEventListener('error', ()=>reject(new Error('composeImage: failed to load image from url')));
        img.src = url;
    });
}
const zhCN = {
    magnifier_position_label: '坐标',
    operation_ok_title: '确定',
    operation_cancel_title: '取消',
    operation_save_title: '保存',
    operation_redo_title: '重做',
    operation_undo_title: '撤销',
    operation_mosaic_title: '马赛克',
    operation_text_title: '文本',
    operation_brush_title: '画笔',
    operation_arrow_title: '箭头',
    operation_ellipse_title: '椭圆',
    operation_rectangle_title: '矩形'
};
const zh_CN = zhCN;
const ScreenshotsContext = react.createContext({
    store: {
        url: void 0,
        image: null,
        width: 0,
        height: 0,
        lang: zh_CN,
        emitterRef: {
            current: {}
        },
        canvasContextRef: {
            current: null
        },
        history: {
            index: -1,
            stack: []
        },
        bounds: null,
        cursor: 'move',
        operation: void 0,
        initialPosition: void 0
    },
    dispatcher: {
        call: void 0,
        setHistory: void 0,
        setBounds: void 0,
        setCursor: void 0,
        setOperation: void 0
    }
});
function useDispatcher() {
    const { dispatcher } = useContext(ScreenshotsContext);
    return dispatcher;
}
function useStore() {
    const { store } = useContext(ScreenshotsContext);
    return store;
}
function useBounds() {
    const { bounds } = useStore();
    const { setBounds } = useDispatcher();
    const set = useCallback((bounds)=>{
        setBounds?.(bounds);
    }, [
        setBounds
    ]);
    const reset = useCallback(()=>{
        setBounds?.(null);
    }, [
        setBounds
    ]);
    return [
        bounds,
        {
            set,
            reset
        }
    ];
}
function useLang() {
    const { lang } = useStore();
    return lang;
}
const magnifierWidth = 100;
const magnifierHeight = 80;
const Screenshots_ScreenshotsMagnifier = /*#__PURE__*/ memo(function({ x, y }) {
    const { width, height, image } = useStore();
    const lang = useLang();
    const [position, setPosition] = useState(null);
    const elRef = useRef(null);
    const canvasRef = useRef(null);
    const ctxRef = useRef(null);
    const [rgb, setRgb] = useState('000000');
    useLayoutEffect(()=>{
        if (!elRef.current) return;
        const elRect = elRef.current.getBoundingClientRect();
        let tx = x + 20;
        let ty = y + 20;
        if (tx + elRect.width > width) tx = x - elRect.width - 20;
        if (ty + elRect.height > height) ty = y - elRect.height - 20;
        if (tx < 0) tx = 0;
        if (ty < 0) ty = 0;
        setPosition({
            x: tx,
            y: ty
        });
    }, [
        width,
        height,
        x,
        y
    ]);
    useEffect(()=>{
        if (!image || !canvasRef.current) {
            ctxRef.current = null;
            return;
        }
        if (!ctxRef.current) ctxRef.current = canvasRef.current.getContext('2d');
        if (!ctxRef.current) return;
        const ctx = ctxRef.current;
        ctx.clearRect(0, 0, magnifierWidth, magnifierHeight);
        const rx = image.naturalWidth / width;
        const ry = image.naturalHeight / height;
        ctx.drawImage(image, x * rx - magnifierWidth / 2, y * ry - magnifierHeight / 2, magnifierWidth, magnifierHeight, 0, 0, magnifierWidth, magnifierHeight);
        const { data } = ctx.getImageData(Math.floor(magnifierWidth / 2), Math.floor(magnifierHeight / 2), 1, 1);
        const hex = Array.from(data.slice(0, 3)).map((val)=>val >= 16 ? val.toString(16) : `0${val.toString(16)}`).join('').toUpperCase();
        setRgb(hex);
    }, [
        width,
        height,
        image,
        x,
        y
    ]);
    return /*#__PURE__*/ React.createElement("div", {
        ref: elRef,
        className: "screenshots-magnifier",
        style: {
            transform: `translate(${position?.x}px, ${position?.y}px)`
        }
    }, /*#__PURE__*/ React.createElement("div", {
        className: "screenshots-magnifier-body"
    }, /*#__PURE__*/ React.createElement("canvas", {
        ref: canvasRef,
        className: "screenshots-magnifier-body-canvas",
        width: magnifierWidth,
        height: magnifierHeight
    })), /*#__PURE__*/ React.createElement("div", {
        className: "screenshots-magnifier-footer"
    }, /*#__PURE__*/ React.createElement("div", {
        className: "screenshots-magnifier-footer-item"
    }, lang.magnifier_position_label, ": (", x, ",", y, ")"), /*#__PURE__*/ React.createElement("div", {
        className: "screenshots-magnifier-footer-item"
    }, "RGB: #", rgb)));
});
function getBoundsByPoints({ x: x1, y: y1 }, { x: x2, y: y2 }, width, height) {
    if (x1 > x2) [x1, x2] = [
        x2,
        x1
    ];
    if (y1 > y2) [y1, y2] = [
        y2,
        y1
    ];
    if (x1 < 0) x1 = 0;
    if (x2 > width) x2 = width;
    if (y1 < 0) y1 = 0;
    if (y2 > height) y2 = height;
    return {
        x: x1,
        y: y1,
        width: x2 - x1,
        height: y2 - y1
    };
}
const Screenshots_ScreenshotsBackground = /*#__PURE__*/ memo(function() {
    const { url, image, width, height } = useStore();
    const [bounds, boundsDispatcher] = useBounds();
    const elRef = useRef(null);
    const pointRef = useRef(null);
    const isMoveRef = useRef(false);
    const [position, setPosition] = useState(null);
    const rafRef = useRef(null);
    const pendingMoveRef = useRef(null);
    const updateBounds = useCallback((p1, p2)=>{
        if (!elRef.current) return;
        const { x, y } = elRef.current.getBoundingClientRect();
        boundsDispatcher.set(getBoundsByPoints({
            x: p1.x - x,
            y: p1.y - y
        }, {
            x: p2.x - x,
            y: p2.y - y
        }, width, height));
    }, [
        width,
        height,
        boundsDispatcher
    ]);
    const flushPointerMove = useCallback(()=>{
        rafRef.current = null;
        const pending = pendingMoveRef.current;
        if (!pending || !pointRef.current) return;
        if (elRef.current) {
            const rect = elRef.current.getBoundingClientRect();
            pending.x < rect.left || pending.y < rect.top || pending.x > rect.right || pending.y > rect.bottom ? setPosition(null) : setPosition({
                x: pending.x - rect.x,
                y: pending.y - rect.y
            });
        }
        updateBounds(pointRef.current, pending);
        isMoveRef.current = true;
    }, [
        updateBounds
    ]);
    const onPointerDown = useCallback((e)=>{
        if (pointRef.current || bounds || 0 !== e.button) return;
        pointRef.current = {
            x: e.clientX,
            y: e.clientY
        };
        isMoveRef.current = false;
    }, [
        bounds
    ]);
    useEffect(()=>{
        const onPointerMove = (e)=>{
            const pending = {
                x: e.clientX,
                y: e.clientY
            };
            pendingMoveRef.current = pending;
            if (!pointRef.current) {
                if (rafRef.current) return;
                rafRef.current = requestAnimationFrame(()=>{
                    rafRef.current = null;
                    if (elRef.current) {
                        const rect = elRef.current.getBoundingClientRect();
                        pending.x < rect.left || pending.y < rect.top || pending.x > rect.right || pending.y > rect.bottom ? setPosition(null) : setPosition({
                            x: pending.x - rect.x,
                            y: pending.y - rect.y
                        });
                    }
                });
                return;
            }
            if (!rafRef.current) rafRef.current = requestAnimationFrame(flushPointerMove);
        };
        const onPointerUp = (e)=>{
            if (!pointRef.current) return;
            if (isMoveRef.current) updateBounds(pointRef.current, {
                x: e.clientX,
                y: e.clientY
            });
            pointRef.current = null;
            isMoveRef.current = false;
            pendingMoveRef.current = null;
        };
        window.addEventListener("pointermove", onPointerMove);
        window.addEventListener("pointerup", onPointerUp);
        return ()=>{
            window.removeEventListener("pointermove", onPointerMove);
            window.removeEventListener("pointerup", onPointerUp);
            if (rafRef.current) {
                cancelAnimationFrame(rafRef.current);
                rafRef.current = null;
            }
        };
    }, [
        updateBounds,
        flushPointerMove
    ]);
    useLayoutEffect(()=>{
        if (!image || bounds) setPosition(null);
    }, [
        image,
        bounds
    ]);
    if (!url) return null;
    const maskStyle = bounds ? {
        left: bounds.x,
        top: bounds.y,
        width: bounds.width,
        height: bounds.height,
        right: "auto",
        bottom: "auto",
        backgroundColor: "transparent",
        boxShadow: `0 0 0 ${Math.max(width, height)}px rgba(0, 0, 0, 0.3)`
    } : void 0;
    return /*#__PURE__*/ React.createElement("div", {
        ref: elRef,
        className: "screenshots-background",
        onPointerDown: onPointerDown
    }, /*#__PURE__*/ React.createElement("img", {
        className: "screenshots-background-image",
        src: url
    }), /*#__PURE__*/ React.createElement("div", {
        className: "screenshots-background-mask",
        style: maskStyle
    }), position && !bounds && /*#__PURE__*/ React.createElement(Screenshots_ScreenshotsMagnifier, {
        x: position?.x,
        y: position?.y
    }));
});
function useCursor() {
    const { cursor } = useStore();
    const { setCursor } = useDispatcher();
    const set = useCallback((cursor)=>{
        setCursor?.(cursor);
    }, [
        setCursor
    ]);
    const reset = useCallback(()=>{
        setCursor?.('move');
    }, [
        setCursor
    ]);
    return [
        cursor,
        {
            set,
            reset
        }
    ];
}
function useEmitter() {
    const { emitterRef } = useStore();
    const on = useCallback((event, listener)=>{
        const emitter = emitterRef.current;
        if (Array.isArray(emitter[event])) emitter[event].push(listener);
        else emitter[event] = [
            listener
        ];
    }, [
        emitterRef
    ]);
    const off = useCallback((event, listener)=>{
        const emitter = emitterRef.current;
        if (Array.isArray(emitter[event])) {
            const index = emitter[event].indexOf(listener);
            if (-1 !== index) emitter[event].splice(index, 1);
        }
    }, [
        emitterRef
    ]);
    const emit = useCallback((event, ...args)=>{
        const emitter = emitterRef.current;
        if (Array.isArray(emitter[event])) emitter[event].forEach((listener)=>{
            listener(...args);
        });
    }, [
        emitterRef
    ]);
    const reset = useCallback(()=>{
        emitterRef.current = {};
    }, [
        emitterRef
    ]);
    return {
        on,
        off,
        emit,
        reset
    };
}
function useHistory() {
    const { history } = useStore();
    const { setHistory } = useDispatcher();
    const push = useCallback((action)=>{
        const { index, stack } = history;
        stack.forEach((item)=>{
            if (1 === item.type) item.isSelected = false;
        });
        if (1 === action.type) action.isSelected = true;
        else if (0 === action.type) action.source.isSelected = true;
        stack.splice(index + 1);
        stack.push(action);
        setHistory?.({
            index: stack.length - 1,
            stack
        });
    }, [
        history,
        setHistory
    ]);
    const pop = useCallback(()=>{
        const { stack } = history;
        stack.pop();
        setHistory?.({
            index: stack.length - 1,
            stack
        });
    }, [
        history,
        setHistory
    ]);
    const undo = useCallback(()=>{
        const { index, stack } = history;
        const item = stack[index];
        if (item) {
            if (1 === item.type) item.isSelected = false;
            else if (0 === item.type) item.source.editHistory.pop();
        }
        setHistory?.({
            index: index <= 0 ? -1 : index - 1,
            stack
        });
    }, [
        history,
        setHistory
    ]);
    const redo = useCallback(()=>{
        const { index, stack } = history;
        const item = stack[index + 1];
        if (item) {
            if (1 === item.type) item.isSelected = false;
            else if (0 === item.type) item.source.editHistory.push(item);
        }
        setHistory?.({
            index: index >= stack.length - 1 ? stack.length - 1 : index + 1,
            stack
        });
    }, [
        history,
        setHistory
    ]);
    const set = useCallback((history)=>{
        setHistory?.({
            ...history
        });
    }, [
        setHistory
    ]);
    const select = useCallback((action)=>{
        history.stack.forEach((item)=>{
            if (1 === item.type) if (item === action) item.isSelected = true;
            else item.isSelected = false;
        });
        setHistory?.({
            ...history
        });
    }, [
        history,
        setHistory
    ]);
    const clearSelect = useCallback(()=>{
        history.stack.forEach((item)=>{
            if (1 === item.type) item.isSelected = false;
        });
        setHistory?.({
            ...history
        });
    }, [
        history,
        setHistory
    ]);
    const reset = useCallback(()=>{
        setHistory?.({
            index: -1,
            stack: []
        });
    }, [
        setHistory
    ]);
    return [
        {
            index: history.index,
            stack: history.stack,
            top: history.stack.slice(history.index, history.index + 1)[0]
        },
        {
            push,
            pop,
            undo,
            redo,
            set,
            select,
            clearSelect,
            reset
        }
    ];
}
function useOperation() {
    const { operation } = useStore();
    const { setOperation } = useDispatcher();
    const set = useCallback((operation)=>{
        setOperation?.(operation);
    }, [
        setOperation
    ]);
    const reset = useCallback(()=>{
        setOperation?.(void 0);
    }, [
        setOperation
    ]);
    return [
        operation,
        {
            set,
            reset
        }
    ];
}
function getBoundsByPoints_getBoundsByPoints({ x: x1, y: y1 }, { x: x2, y: y2 }, bounds, width, height, resizeOrMove) {
    if (x1 > x2) [x1, x2] = [
        x2,
        x1
    ];
    if (y1 > y2) [y1, y2] = [
        y2,
        y1
    ];
    if (x1 < 0) {
        x1 = 0;
        if ('move' === resizeOrMove) x2 = bounds.width;
    }
    if (x2 > width) {
        x2 = width;
        if ('move' === resizeOrMove) x1 = x2 - bounds.width;
    }
    if (y1 < 0) {
        y1 = 0;
        if ('move' === resizeOrMove) y2 = bounds.height;
    }
    if (y2 > height) {
        y2 = height;
        if ('move' === resizeOrMove) y1 = y2 - bounds.height;
    }
    return {
        x: x1,
        y: y1,
        width: Math.max(x2 - x1, 1),
        height: Math.max(y2 - y1, 1)
    };
}
function getPoints_getBoundsByPoints(e, resizeOrMove, point, bounds) {
    const x = e.clientX - point.x;
    const y = e.clientY - point.y;
    let x1 = bounds.x;
    let y1 = bounds.y;
    let x2 = bounds.x + bounds.width;
    let y2 = bounds.y + bounds.height;
    switch(resizeOrMove){
        case 'top':
            y1 += y;
            break;
        case 'top-right':
            x2 += x;
            y1 += y;
            break;
        case 'right':
            x2 += x;
            break;
        case 'right-bottom':
            x2 += x;
            y2 += y;
            break;
        case 'bottom':
            y2 += y;
            break;
        case 'bottom-left':
            x1 += x;
            y2 += y;
            break;
        case 'left':
            x1 += x;
            break;
        case 'left-top':
            x1 += x;
            y1 += y;
            break;
        case 'move':
            x1 += x;
            y1 += y;
            x2 += x;
            y2 += y;
            break;
        default:
            break;
    }
    return [
        {
            x: x1,
            y: y1
        },
        {
            x: x2,
            y: y2
        }
    ];
}
function isPointInDraw(bounds, canvas, history, e) {
    if (!canvas) return false;
    const $canvas = document.createElement('canvas');
    $canvas.width = bounds.width;
    $canvas.height = bounds.height;
    const ctx = $canvas.getContext('2d');
    if (!ctx) return false;
    const { left, top } = canvas.getBoundingClientRect();
    const x = e.clientX - left;
    const y = e.clientY - top;
    const stack = [
        ...history.stack.slice(0, history.index + 1)
    ];
    return stack.reverse().find((item)=>{
        if (1 !== item.type) return false;
        ctx.clearRect(0, 0, bounds.width, bounds.height);
        return item.isHit?.(ctx, item, {
            x,
            y
        });
    });
}
const borders = [
    "top",
    "right",
    "bottom",
    "left"
];
const resizePoints = [
    "top",
    "top-right",
    "right",
    "right-bottom",
    "bottom",
    "bottom-left",
    "left",
    "left-top"
];
const Screenshots_ScreenshotsCanvas = /*#__PURE__*/ memo(/*#__PURE__*/ forwardRef(function(_props, ref) {
    const { url, image, width, height } = useStore();
    const emitter = useEmitter();
    const [history] = useHistory();
    const [cursor] = useCursor();
    const [bounds, boundsDispatcher] = useBounds();
    const [operation] = useOperation();
    const resizeOrMoveRef = useRef(void 0);
    const pointRef = useRef(null);
    const boundsRef = useRef(null);
    const canvasRef = useRef(null);
    const ctxRef = useRef(null);
    const isCanResize = bounds && !operation;
    const draw = useCallback(()=>{
        if (!ctxRef.current) return;
        const ctx = ctxRef.current;
        ctx.imageSmoothingEnabled = true;
        ctx.imageSmoothingQuality = "low";
        ctx.clearRect(0, 0, ctx.canvas.width, ctx.canvas.height);
        history.stack.slice(0, history.index + 1).forEach((item)=>{
            if (1 === item.type) item.draw(ctx, item);
        });
    }, [
        history
    ]);
    const onPointerDown = useCallback((e, resizeOrMove)=>{
        if (0 !== e.button || !bounds) return;
        if (operation) {
            const draw = isPointInDraw(bounds, canvasRef.current, history, e.nativeEvent);
            if (draw) emitter.emit("drawselect", draw, e.nativeEvent);
            else emitter.emit("pointerdown", e.nativeEvent);
        } else {
            resizeOrMoveRef.current = resizeOrMove;
            pointRef.current = {
                x: e.clientX,
                y: e.clientY
            };
            boundsRef.current = {
                x: bounds.x,
                y: bounds.y,
                width: bounds.width,
                height: bounds.height
            };
        }
    }, [
        bounds,
        operation,
        emitter,
        history
    ]);
    const updateBounds = useCallback((e)=>{
        if (!resizeOrMoveRef.current || !pointRef.current || !boundsRef.current || !bounds) return;
        const points = getPoints_getBoundsByPoints(e, resizeOrMoveRef.current, pointRef.current, boundsRef.current);
        boundsDispatcher.set(getBoundsByPoints_getBoundsByPoints(points[0], points[1], bounds, width, height, resizeOrMoveRef.current));
    }, [
        width,
        height,
        bounds,
        boundsDispatcher
    ]);
    useLayoutEffect(()=>{
        if (!bounds || !canvasRef.current) {
            ctxRef.current = null;
            return;
        }
        if (!ctxRef.current) ctxRef.current = canvasRef.current.getContext("2d");
        draw();
    }, [
        image,
        bounds,
        draw
    ]);
    useEffect(()=>{
        const onPointerMove = (e)=>{
            if (operation) emitter.emit("pointermove", e);
            else {
                if (!resizeOrMoveRef.current || !pointRef.current || !boundsRef.current) return;
                updateBounds(e);
            }
        };
        const onPointerUp = (e)=>{
            if (operation) emitter.emit("pointerup", e);
            else {
                if (!resizeOrMoveRef.current || !pointRef.current || !boundsRef.current) return;
                updateBounds(e);
                resizeOrMoveRef.current = void 0;
                pointRef.current = null;
                boundsRef.current = null;
            }
        };
        window.addEventListener("pointermove", onPointerMove);
        window.addEventListener("pointerup", onPointerUp);
        return ()=>{
            window.removeEventListener("pointermove", onPointerMove);
            window.removeEventListener("pointerup", onPointerUp);
        };
    }, [
        updateBounds,
        operation,
        emitter
    ]);
    useImperativeHandle(ref, ()=>ctxRef.current);
    return /*#__PURE__*/ React.createElement("div", {
        className: "screenshots-canvas",
        style: {
            width: bounds?.width || 0,
            height: bounds?.height || 0,
            transform: bounds ? `translate(${bounds.x}px, ${bounds.y}px)` : "none"
        }
    }, /*#__PURE__*/ React.createElement("div", {
        className: "screenshots-canvas-body"
    }, /*#__PURE__*/ React.createElement("img", {
        className: "screenshots-canvas-image",
        src: url,
        style: {
            width,
            height,
            transform: bounds ? `translate(${-bounds.x}px, ${-bounds.y}px)` : "none"
        }
    }), /*#__PURE__*/ React.createElement("canvas", {
        ref: canvasRef,
        className: "screenshots-canvas-panel",
        width: bounds?.width || 0,
        height: bounds?.height || 0
    })), /*#__PURE__*/ React.createElement("div", {
        className: "screenshots-canvas-mask",
        style: {
            cursor
        },
        onPointerDown: (e)=>onPointerDown(e, "move")
    }, isCanResize && /*#__PURE__*/ React.createElement("div", {
        className: "screenshots-canvas-size"
    }, bounds.width, " \xd7 ", bounds.height)), borders.map((border)=>/*#__PURE__*/ React.createElement("div", {
            key: border,
            className: `screenshots-canvas-border-${border}`
        })), isCanResize && resizePoints.map((resizePoint)=>/*#__PURE__*/ React.createElement("div", {
            key: resizePoint,
            className: `screenshots-canvas-point-${resizePoint}`,
            onPointerDown: (e)=>onPointerDown(e, resizePoint)
        })));
}));
function useCanvasContextRef() {
    const { canvasContextRef } = useStore();
    return canvasContextRef;
}
function useCanvasPointerDown(onPointerDown) {
    const emitter = useEmitter();
    useEffect(()=>{
        emitter.on('pointerdown', onPointerDown);
        return ()=>{
            emitter.off('pointerdown', onPointerDown);
        };
    }, [
        onPointerDown,
        emitter
    ]);
}
function useCanvasPointerMove(onPointerMove) {
    const emitter = useEmitter();
    useEffect(()=>{
        emitter.on('pointermove', onPointerMove);
        return ()=>{
            emitter.off('pointermove', onPointerMove);
        };
    }, [
        onPointerMove,
        emitter
    ]);
}
function useCanvasPointerUp(onPointerUp) {
    const emitter = useEmitter();
    useEffect(()=>{
        emitter.on("pointerup", onPointerUp);
        return ()=>{
            emitter.off("pointerup", onPointerUp);
        };
    }, [
        onPointerUp,
        emitter
    ]);
}
function useDrawSelect(onDrawSelect) {
    const emitter = useEmitter();
    useEffect(()=>{
        emitter.on('drawselect', onDrawSelect);
        return ()=>{
            emitter.off('drawselect', onDrawSelect);
        };
    }, [
        onDrawSelect,
        emitter
    ]);
}
const Screenshots_ScreenshotsOption = /*#__PURE__*/ memo(function({ open, content, children }) {
    const childrenRef = useRef(null);
    const popoverRef = useRef(null);
    const contentRef = useRef(null);
    const operationsRect = useContext(ScreenshotsOperationsCtx);
    const [placement, setPlacement] = useState("bottom");
    const [position, setPosition] = useState(null);
    const [offsetX, setOffsetX] = useState(0);
    const getPopoverEl = ()=>{
        if (!popoverRef.current) popoverRef.current = document.createElement("div");
        return popoverRef.current;
    };
    useEffect(()=>{
        const $el = getPopoverEl();
        if (open) document.body.appendChild($el);
        return ()=>{
            $el.remove();
        };
    }, [
        open
    ]);
    useEffect(()=>{
        if (!open || !operationsRect || !childrenRef.current || !contentRef.current) return;
        const childrenRect = childrenRef.current.getBoundingClientRect();
        const contentRect = contentRef.current.getBoundingClientRect();
        let currentPlacement = placement;
        let x = childrenRect.left + childrenRect.width / 2;
        let y = childrenRect.top + childrenRect.height;
        let currentOffsetX = offsetX;
        if (x + contentRect.width / 2 > operationsRect.x + operationsRect.width) {
            const ox = x;
            x = operationsRect.x + operationsRect.width - contentRect.width / 2;
            currentOffsetX = ox - x;
        }
        if (x < operationsRect.x + contentRect.width / 2) {
            const ox = x;
            x = operationsRect.x + contentRect.width / 2;
            currentOffsetX = ox - x;
        }
        if (y > window.innerHeight - contentRect.height) {
            if ("bottom" === currentPlacement) currentPlacement = "top";
            y = childrenRect.top - contentRect.height;
        }
        if (y < 0) {
            if ("top" === currentPlacement) currentPlacement = "bottom";
            y = childrenRect.top + childrenRect.height;
        }
        if (currentPlacement !== placement) setPlacement(currentPlacement);
        if (position?.x !== x || position.y !== y) setPosition({
            x,
            y
        });
        if (currentOffsetX !== offsetX) setOffsetX(currentOffsetX);
    });
    return /*#__PURE__*/ React.createElement(React.Fragment, null, /*#__PURE__*/ cloneElement(children, {
        ref: childrenRef
    }), open && content && /*#__PURE__*/ createPortal(/*#__PURE__*/ React.createElement("div", {
        ref: contentRef,
        className: "screenshots-option",
        style: {
            visibility: position ? "visible" : "hidden",
            transform: `translate(${position?.x ?? 0}px, ${position?.y ?? 0}px)`
        },
        "data-placement": placement
    }, /*#__PURE__*/ React.createElement("div", {
        className: "screenshots-option-container"
    }, content), /*#__PURE__*/ React.createElement("div", {
        className: "screenshots-option-arrow",
        style: {
            marginLeft: offsetX
        }
    })), getPopoverEl()));
});
const Screenshots_ScreenshotsButton = /*#__PURE__*/ memo(function({ title, icon, checked, disabled, option, onClick }) {
    const classNames = [
        'screenshots-button'
    ];
    const onButtonClick = useCallback((e)=>{
        if (disabled || !onClick) return;
        onClick(e);
    }, [
        disabled,
        onClick
    ]);
    if (checked) classNames.push('screenshots-button-checked');
    if (disabled) classNames.push('screenshots-button-disabled');
    return /*#__PURE__*/ React.createElement(Screenshots_ScreenshotsOption, {
        open: checked,
        content: option
    }, /*#__PURE__*/ React.createElement("div", {
        className: classNames.join(' '),
        title: title,
        onClick: onButtonClick
    }, /*#__PURE__*/ React.createElement("span", {
        className: icon
    })));
});
const Screenshots_ScreenshotsColor = /*#__PURE__*/ memo(function({ value, onChange }) {
    const colors = [
        '#ee5126',
        '#fceb4d',
        '#90e746',
        '#51c0fa',
        '#7a7a7a',
        '#ffffff'
    ];
    return /*#__PURE__*/ React.createElement("div", {
        className: "screenshots-color"
    }, colors.map((color)=>{
        const classNames = [
            'screenshots-color-item'
        ];
        if (color === value) classNames.push('screenshots-color-active');
        return /*#__PURE__*/ React.createElement("div", {
            key: color,
            className: classNames.join(' '),
            style: {
                backgroundColor: color
            },
            onClick: ()=>onChange?.(color)
        });
    }));
});
const Screenshots_ScreenshotsSize = /*#__PURE__*/ memo(function({ value, onChange }) {
    const sizes = [
        3,
        6,
        9
    ];
    return /*#__PURE__*/ React.createElement("div", {
        className: "screenshots-size"
    }, sizes.map((size)=>{
        const classNames = [
            'screenshots-size-item'
        ];
        if (size === value) classNames.push('screenshots-size-active');
        return /*#__PURE__*/ React.createElement("div", {
            key: size,
            className: classNames.join(' '),
            onClick: ()=>onChange?.(size)
        }, /*#__PURE__*/ React.createElement("div", {
            className: "screenshots-size-pointer",
            style: {
                width: 1.8 * size,
                height: 1.8 * size
            }
        }));
    }));
});
const Screenshots_ScreenshotsSizeColor = /*#__PURE__*/ memo(function({ size, color, onSizeChange, onColorChange }) {
    return /*#__PURE__*/ React.createElement("div", {
        className: "screenshots-sizecolor"
    }, /*#__PURE__*/ React.createElement(Screenshots_ScreenshotsSize, {
        value: size,
        onChange: onSizeChange
    }), /*#__PURE__*/ React.createElement(Screenshots_ScreenshotsColor, {
        value: color,
        onChange: onColorChange
    }));
});
const CircleRadius = 4;
function drawDragCircle(ctx, x, y) {
    ctx.lineWidth = 1;
    ctx.strokeStyle = '#000000';
    ctx.fillStyle = '#ffffff';
    ctx.beginPath();
    ctx.arc(x, y, CircleRadius, 0, 2 * Math.PI);
    ctx.fill();
    ctx.stroke();
}
function isHit(ctx, action, point) {
    action.draw(ctx, action);
    const { data } = ctx.getImageData(point.x, point.y, 1, 1);
    return data.some((val)=>0 !== val);
}
function isHitCircle(canvas, e, point) {
    if (!canvas) return false;
    const { left, top } = canvas.getBoundingClientRect();
    const x = e.clientX - left;
    const y = e.clientY - top;
    return (point.x - x) ** 2 + (point.y - y) ** 2 < CircleRadius ** 2;
}
function getEditedArrowData(action) {
    let { x1, y1, x2, y2 } = action.data;
    action.editHistory.forEach(({ data })=>{
        const x = data.x2 - data.x1;
        const y = data.y2 - data.y1;
        if (0 === data.type) {
            x1 += x;
            y1 += y;
            x2 += x;
            y2 += y;
        } else if (1 === data.type) {
            x1 += x;
            y1 += y;
        } else if (2 === data.type) {
            x2 += x;
            y2 += y;
        }
    });
    return {
        ...action.data,
        x1,
        x2,
        y1,
        y2
    };
}
function draw_draw(ctx, action) {
    const { size, color, x1, x2, y1, y2 } = getEditedArrowData(action);
    ctx.lineCap = 'round';
    ctx.lineJoin = 'bevel';
    ctx.lineWidth = size;
    ctx.strokeStyle = color;
    const dx = x2 - x1;
    const dy = y2 - y1;
    const length = 3 * size;
    const angle = Math.atan2(dy, dx);
    ctx.beginPath();
    ctx.moveTo(x1, y1);
    ctx.lineTo(x2, y2);
    ctx.lineTo(x2 - length * Math.cos(angle - Math.PI / 6), y2 - length * Math.sin(angle - Math.PI / 6));
    ctx.moveTo(x2, y2);
    ctx.lineTo(x2 - length * Math.cos(angle + Math.PI / 6), y2 - length * Math.sin(angle + Math.PI / 6));
    ctx.stroke();
    if (action.isSelected) {
        drawDragCircle(ctx, x1, y1);
        drawDragCircle(ctx, x2, y2);
    }
}
function Arrow() {
    const lang = useLang();
    const [, cursorDispatcher] = useCursor();
    const [operation, operationDispatcher] = useOperation();
    const [history, historyDispatcher] = useHistory();
    const canvasContextRef = useCanvasContextRef();
    const [size, setSize] = useState(3);
    const [color, setColor] = useState('#ee5126');
    const arrowRef = useRef(null);
    const arrowEditRef = useRef(null);
    const checked = 'Arrow' === operation;
    const selectArrow = useCallback(()=>{
        operationDispatcher.set('Arrow');
        cursorDispatcher.set('default');
    }, [
        operationDispatcher,
        cursorDispatcher
    ]);
    const onSelectArrow = useCallback(()=>{
        if (checked) return;
        selectArrow();
        historyDispatcher.clearSelect();
    }, [
        checked,
        selectArrow,
        historyDispatcher
    ]);
    const onDrawSelect = useCallback((action, e)=>{
        if ('Arrow' !== action.name || !canvasContextRef.current) return;
        const source = action;
        selectArrow();
        const { x1, y1, x2, y2 } = getEditedArrowData(source);
        let type = 0;
        if (isHitCircle(canvasContextRef.current.canvas, e, {
            x: x1,
            y: y1
        })) type = 1;
        else if (isHitCircle(canvasContextRef.current.canvas, e, {
            x: x2,
            y: y2
        })) type = 2;
        arrowEditRef.current = {
            type: 0,
            data: {
                type,
                x1: e.clientX,
                y1: e.clientY,
                x2: e.clientX,
                y2: e.clientY
            },
            source
        };
        historyDispatcher.select(action);
    }, [
        canvasContextRef,
        selectArrow,
        historyDispatcher
    ]);
    const onPointerDown = useCallback((e)=>{
        if (!checked || arrowRef.current || !canvasContextRef.current) return;
        const { left, top } = canvasContextRef.current.canvas.getBoundingClientRect();
        arrowRef.current = {
            name: 'Arrow',
            type: 1,
            data: {
                size,
                color,
                x1: e.clientX - left,
                y1: e.clientY - top,
                x2: e.clientX - left,
                y2: e.clientY - top
            },
            editHistory: [],
            draw: draw_draw,
            isHit: isHit
        };
    }, [
        checked,
        color,
        size,
        canvasContextRef
    ]);
    const onPointerMove = useCallback((e)=>{
        if (!checked || !canvasContextRef.current) return;
        if (arrowEditRef.current) {
            arrowEditRef.current.data.x2 = e.clientX;
            arrowEditRef.current.data.y2 = e.clientY;
            if (history.top !== arrowEditRef.current) {
                arrowEditRef.current.source.editHistory.push(arrowEditRef.current);
                historyDispatcher.push(arrowEditRef.current);
            } else historyDispatcher.set(history);
        } else if (arrowRef.current) {
            const { left, top } = canvasContextRef.current.canvas.getBoundingClientRect();
            arrowRef.current.data.x2 = e.clientX - left;
            arrowRef.current.data.y2 = e.clientY - top;
            if (history.top !== arrowRef.current) historyDispatcher.push(arrowRef.current);
            else historyDispatcher.set(history);
        }
    }, [
        checked,
        history,
        canvasContextRef,
        historyDispatcher
    ]);
    const onPointerUp = useCallback(()=>{
        if (!checked) return;
        if (arrowRef.current) historyDispatcher.clearSelect();
        arrowRef.current = null;
        arrowEditRef.current = null;
    }, [
        checked,
        historyDispatcher
    ]);
    useDrawSelect(onDrawSelect);
    useCanvasPointerDown(onPointerDown);
    useCanvasPointerMove(onPointerMove);
    useCanvasPointerUp(onPointerUp);
    return /*#__PURE__*/ React.createElement(Screenshots_ScreenshotsButton, {
        title: lang.operation_arrow_title,
        icon: "icon-arrow",
        checked: checked,
        onClick: onSelectArrow,
        option: /*#__PURE__*/ React.createElement(Screenshots_ScreenshotsSizeColor, {
            size: size,
            color: color,
            onSizeChange: setSize,
            onColorChange: setColor
        })
    });
}
function Brush_draw_draw(ctx, action) {
    const { size, color, points } = action.data;
    ctx.lineCap = 'round';
    ctx.lineJoin = 'round';
    ctx.lineWidth = size;
    ctx.strokeStyle = color;
    const distance = action.editHistory.reduce((distance, { data })=>({
            x: distance.x + data.x2 - data.x1,
            y: distance.y + data.y2 - data.y1
        }), {
        x: 0,
        y: 0
    });
    ctx.beginPath();
    points.forEach((item, index)=>{
        if (0 === index) ctx.moveTo(item.x + distance.x, item.y + distance.y);
        else ctx.lineTo(item.x + distance.x, item.y + distance.y);
    });
    ctx.stroke();
    if (action.isSelected) {
        ctx.lineWidth = 1;
        ctx.strokeStyle = '#000000';
        ctx.beginPath();
        points.forEach((item, index)=>{
            if (0 === index) ctx.moveTo(item.x + distance.x, item.y + distance.y);
            else ctx.lineTo(item.x + distance.x, item.y + distance.y);
        });
        ctx.stroke();
    }
}
function Brush() {
    const lang = useLang();
    const [, cursorDispatcher] = useCursor();
    const [operation, operationDispatcher] = useOperation();
    const canvasContextRef = useCanvasContextRef();
    const [history, historyDispatcher] = useHistory();
    const [size, setSize] = useState(3);
    const [color, setColor] = useState('#ee5126');
    const brushRef = useRef(null);
    const brushEditRef = useRef(null);
    const checked = 'Brush' === operation;
    const selectBrush = useCallback(()=>{
        operationDispatcher.set('Brush');
        cursorDispatcher.set('default');
    }, [
        operationDispatcher,
        cursorDispatcher
    ]);
    const onSelectBrush = useCallback(()=>{
        if (checked) return;
        selectBrush();
        historyDispatcher.clearSelect();
    }, [
        checked,
        selectBrush,
        historyDispatcher
    ]);
    const onDrawSelect = useCallback((action, e)=>{
        if ('Brush' !== action.name) return;
        selectBrush();
        brushEditRef.current = {
            type: 0,
            data: {
                x1: e.clientX,
                y1: e.clientY,
                x2: e.clientX,
                y2: e.clientY
            },
            source: action
        };
        historyDispatcher.select(action);
    }, [
        selectBrush,
        historyDispatcher
    ]);
    const onPointerDown = useCallback((e)=>{
        if (!checked || brushRef.current || !canvasContextRef.current) return;
        const { left, top } = canvasContextRef.current.canvas.getBoundingClientRect();
        brushRef.current = {
            name: 'Brush',
            type: 1,
            data: {
                size,
                color,
                points: [
                    {
                        x: e.clientX - left,
                        y: e.clientY - top
                    }
                ]
            },
            editHistory: [],
            draw: Brush_draw_draw,
            isHit: isHit
        };
    }, [
        checked,
        canvasContextRef,
        size,
        color
    ]);
    const onPointerMove = useCallback((e)=>{
        if (!checked || !canvasContextRef.current) return;
        if (brushEditRef.current) {
            brushEditRef.current.data.x2 = e.clientX;
            brushEditRef.current.data.y2 = e.clientY;
            if (history.top !== brushEditRef.current) {
                brushEditRef.current.source.editHistory.push(brushEditRef.current);
                historyDispatcher.push(brushEditRef.current);
            } else historyDispatcher.set(history);
        } else if (brushRef.current) {
            const { left, top } = canvasContextRef.current.canvas.getBoundingClientRect();
            brushRef.current.data.points.push({
                x: e.clientX - left,
                y: e.clientY - top
            });
            if (history.top !== brushRef.current) historyDispatcher.push(brushRef.current);
            else historyDispatcher.set(history);
        }
    }, [
        checked,
        history,
        canvasContextRef,
        historyDispatcher
    ]);
    const onPointerUp = useCallback(()=>{
        if (!checked) return;
        if (brushRef.current) historyDispatcher.clearSelect();
        brushRef.current = null;
        brushEditRef.current = null;
    }, [
        checked,
        historyDispatcher
    ]);
    useDrawSelect(onDrawSelect);
    useCanvasPointerDown(onPointerDown);
    useCanvasPointerMove(onPointerMove);
    useCanvasPointerUp(onPointerUp);
    return /*#__PURE__*/ React.createElement(Screenshots_ScreenshotsButton, {
        title: lang.operation_brush_title,
        icon: "icon-brush",
        checked: checked,
        onClick: onSelectBrush,
        option: /*#__PURE__*/ React.createElement(Screenshots_ScreenshotsSizeColor, {
            size: size,
            color: color,
            onSizeChange: setSize,
            onColorChange: setColor
        })
    });
}
function useCall() {
    const dispatcher = useDispatcher();
    const call = useCallback((funcName, ...args)=>{
        dispatcher.call?.(funcName, ...args);
    }, [
        dispatcher
    ]);
    return call;
}
function useReset() {
    const emitter = useEmitter();
    const [, boundsDispatcher] = useBounds();
    const [, cursorDispatcher] = useCursor();
    const [, historyDispatcher] = useHistory();
    const [, operatioDispatcher] = useOperation();
    const reset = useCallback(()=>{
        emitter.reset();
        historyDispatcher.reset();
        boundsDispatcher.reset();
        cursorDispatcher.reset();
        operatioDispatcher.reset();
    }, [
        emitter,
        historyDispatcher,
        boundsDispatcher,
        cursorDispatcher,
        operatioDispatcher
    ]);
    return reset;
}
function Cancel() {
    const call = useCall();
    const reset = useReset();
    const lang = useLang();
    const onClick = useCallback(()=>{
        call('onCancel');
        reset();
    }, [
        call,
        reset
    ]);
    return /*#__PURE__*/ React.createElement(Screenshots_ScreenshotsButton, {
        title: lang.operation_cancel_title,
        icon: "icon-cancel",
        onClick: onClick
    });
}
function getEditedEllipseData(action) {
    let { x1, y1, x2, y2 } = action.data;
    action.editHistory.forEach(({ data })=>{
        const x = data.x2 - data.x1;
        const y = data.y2 - data.y1;
        if (0 === data.type) {
            x1 += x;
            y1 += y;
            x2 += x;
            y2 += y;
        } else if (1 === data.type) y1 += y;
        else if (2 === data.type) {
            x2 += x;
            y1 += y;
        } else if (3 === data.type) x2 += x;
        else if (4 === data.type) {
            x2 += x;
            y2 += y;
        } else if (5 === data.type) y2 += y;
        else if (6 === data.type) {
            x1 += x;
            y2 += y;
        } else if (7 === data.type) x1 += x;
        else if (8 === data.type) {
            x1 += x;
            y1 += y;
        }
    });
    return {
        ...action.data,
        x1,
        x2,
        y1,
        y2
    };
}
function Ellipse_draw_draw(ctx, action) {
    const { size, color, x1, y1, x2, y2 } = getEditedEllipseData(action);
    ctx.lineCap = 'butt';
    ctx.lineJoin = 'miter';
    ctx.lineWidth = size;
    ctx.strokeStyle = color;
    const x = (x1 + x2) / 2;
    const y = (y1 + y2) / 2;
    const rx = Math.abs(x2 - x1) / 2;
    const ry = Math.abs(y2 - y1) / 2;
    const k = 0.5522848;
    const ox = rx * k;
    const oy = ry * k;
    ctx.beginPath();
    ctx.moveTo(x - rx, y);
    ctx.bezierCurveTo(x - rx, y - oy, x - ox, y - ry, x, y - ry);
    ctx.bezierCurveTo(x + ox, y - ry, x + rx, y - oy, x + rx, y);
    ctx.bezierCurveTo(x + rx, y + oy, x + ox, y + ry, x, y + ry);
    ctx.bezierCurveTo(x - ox, y + ry, x - rx, y + oy, x - rx, y);
    ctx.closePath();
    ctx.stroke();
    if (action.isSelected) {
        ctx.lineWidth = 1;
        ctx.strokeStyle = '#000000';
        ctx.fillStyle = '#ffffff';
        ctx.beginPath();
        ctx.moveTo(x1, y1);
        ctx.lineTo(x2, y1);
        ctx.lineTo(x2, y2);
        ctx.lineTo(x1, y2);
        ctx.closePath();
        ctx.stroke();
        drawDragCircle(ctx, (x1 + x2) / 2, y1);
        drawDragCircle(ctx, x2, y1);
        drawDragCircle(ctx, x2, (y1 + y2) / 2);
        drawDragCircle(ctx, x2, y2);
        drawDragCircle(ctx, (x1 + x2) / 2, y2);
        drawDragCircle(ctx, x1, y2);
        drawDragCircle(ctx, x1, (y1 + y2) / 2);
        drawDragCircle(ctx, x1, y1);
    }
}
function Ellipse() {
    const lang = useLang();
    const [history, historyDispatcher] = useHistory();
    const [operation, operationDispatcher] = useOperation();
    const [, cursorDispatcher] = useCursor();
    const canvasContextRef = useCanvasContextRef();
    const [size, setSize] = useState(3);
    const [color, setColor] = useState('#ee5126');
    const ellipseRef = useRef(null);
    const ellipseEditRef = useRef(null);
    const checked = 'Ellipse' === operation;
    const selectEllipse = useCallback(()=>{
        operationDispatcher.set('Ellipse');
        cursorDispatcher.set('crosshair');
    }, [
        operationDispatcher,
        cursorDispatcher
    ]);
    const onSelectEllipse = useCallback(()=>{
        if (checked) return;
        selectEllipse();
        historyDispatcher.clearSelect();
    }, [
        checked,
        selectEllipse,
        historyDispatcher
    ]);
    const onDrawSelect = useCallback((action, e)=>{
        if ('Ellipse' !== action.name || !canvasContextRef.current) return;
        const source = action;
        selectEllipse();
        const { x1, y1, x2, y2 } = getEditedEllipseData(source);
        let type = 0;
        if (isHitCircle(canvasContextRef.current.canvas, e, {
            x: (x1 + x2) / 2,
            y: y1
        })) type = 1;
        else if (isHitCircle(canvasContextRef.current.canvas, e, {
            x: x2,
            y: y1
        })) type = 2;
        else if (isHitCircle(canvasContextRef.current.canvas, e, {
            x: x2,
            y: (y1 + y2) / 2
        })) type = 3;
        else if (isHitCircle(canvasContextRef.current.canvas, e, {
            x: x2,
            y: y2
        })) type = 4;
        else if (isHitCircle(canvasContextRef.current.canvas, e, {
            x: (x1 + x2) / 2,
            y: y2
        })) type = 5;
        else if (isHitCircle(canvasContextRef.current.canvas, e, {
            x: x1,
            y: y2
        })) type = 6;
        else if (isHitCircle(canvasContextRef.current.canvas, e, {
            x: x1,
            y: (y1 + y2) / 2
        })) type = 7;
        else if (isHitCircle(canvasContextRef.current.canvas, e, {
            x: x1,
            y: y1
        })) type = 8;
        ellipseEditRef.current = {
            type: 0,
            data: {
                type,
                x1: e.clientX,
                y1: e.clientY,
                x2: e.clientX,
                y2: e.clientY
            },
            source
        };
        historyDispatcher.select(action);
    }, [
        canvasContextRef,
        selectEllipse,
        historyDispatcher
    ]);
    const onPointerDown = useCallback((e)=>{
        if (!checked || !canvasContextRef.current || ellipseRef.current) return;
        const { left, top } = canvasContextRef.current.canvas.getBoundingClientRect();
        const x = e.clientX - left;
        const y = e.clientY - top;
        ellipseRef.current = {
            name: 'Ellipse',
            type: 1,
            data: {
                size,
                color,
                x1: x,
                y1: y,
                x2: x,
                y2: y
            },
            editHistory: [],
            draw: Ellipse_draw_draw,
            isHit: isHit
        };
    }, [
        checked,
        size,
        color,
        canvasContextRef
    ]);
    const onPointerMove = useCallback((e)=>{
        if (!checked || !canvasContextRef.current) return;
        if (ellipseEditRef.current) {
            ellipseEditRef.current.data.x2 = e.clientX;
            ellipseEditRef.current.data.y2 = e.clientY;
            if (history.top !== ellipseEditRef.current) {
                ellipseEditRef.current.source.editHistory.push(ellipseEditRef.current);
                historyDispatcher.push(ellipseEditRef.current);
            } else historyDispatcher.set(history);
        } else if (ellipseRef.current) {
            const { left, top } = canvasContextRef.current.canvas.getBoundingClientRect();
            ellipseRef.current.data.x2 = e.clientX - left;
            ellipseRef.current.data.y2 = e.clientY - top;
            if (history.top !== ellipseRef.current) historyDispatcher.push(ellipseRef.current);
            else historyDispatcher.set(history);
        }
    }, [
        checked,
        canvasContextRef,
        history,
        historyDispatcher
    ]);
    const onPointerUp = useCallback(()=>{
        if (!checked) return;
        if (ellipseRef.current) historyDispatcher.clearSelect();
        ellipseRef.current = null;
        ellipseEditRef.current = null;
    }, [
        checked,
        historyDispatcher
    ]);
    useDrawSelect(onDrawSelect);
    useCanvasPointerDown(onPointerDown);
    useCanvasPointerMove(onPointerMove);
    useCanvasPointerUp(onPointerUp);
    return /*#__PURE__*/ React.createElement(Screenshots_ScreenshotsButton, {
        title: lang.operation_ellipse_title,
        icon: "icon-ellipse",
        checked: checked,
        onClick: onSelectEllipse,
        option: /*#__PURE__*/ React.createElement(Screenshots_ScreenshotsSizeColor, {
            size: size,
            color: color,
            onSizeChange: setSize,
            onColorChange: setColor
        })
    });
}
function getColor(x, y, imageData) {
    if (!imageData) return [
        0,
        0,
        0,
        0
    ];
    const { data, width } = imageData;
    const index = y * width * 4 + 4 * x;
    return Array.from(data.slice(index, index + 4));
}
function Mosaic_draw(ctx, action) {
    const { tiles, size } = action.data;
    tiles.forEach((tile)=>{
        const r = Math.round(tile.color[0]);
        const g = Math.round(tile.color[1]);
        const b = Math.round(tile.color[2]);
        const a = tile.color[3] / 255;
        ctx.fillStyle = `rgba(${r}, ${g}, ${b}, ${a})`;
        ctx.fillRect(tile.x - size / 2, tile.y - size / 2, size, size);
    });
}
function Mosaic() {
    const lang = useLang();
    const { image, width, height } = useStore();
    const [operation, operationDispatcher] = useOperation();
    const canvasContextRef = useCanvasContextRef();
    const [history, historyDispatcher] = useHistory();
    const [bounds] = useBounds();
    const [, cursorDispatcher] = useCursor();
    const [size, setSize] = useState(3);
    const imageDataRef = useRef(null);
    const mosaicRef = useRef(null);
    const checked = 'Mosaic' === operation;
    const selectMosaic = useCallback(()=>{
        operationDispatcher.set('Mosaic');
        cursorDispatcher.set('crosshair');
    }, [
        operationDispatcher,
        cursorDispatcher
    ]);
    const onSelectMosaic = useCallback(()=>{
        if (checked) return;
        selectMosaic();
        historyDispatcher.clearSelect();
    }, [
        checked,
        selectMosaic,
        historyDispatcher
    ]);
    const onPointerDown = useCallback((e)=>{
        if (!checked || mosaicRef.current || !imageDataRef.current || !canvasContextRef.current) return;
        const rect = canvasContextRef.current.canvas.getBoundingClientRect();
        const x = e.clientX - rect.x;
        const y = e.clientY - rect.y;
        const mosaicSize = 2 * size;
        mosaicRef.current = {
            name: 'Mosaic',
            type: 1,
            data: {
                size: mosaicSize,
                tiles: [
                    {
                        x,
                        y,
                        color: getColor(x, y, imageDataRef.current)
                    }
                ]
            },
            editHistory: [],
            draw: Mosaic_draw
        };
    }, [
        checked,
        size,
        canvasContextRef
    ]);
    const onPointerMove = useCallback((e)=>{
        if (!checked || !mosaicRef.current || !canvasContextRef.current || !imageDataRef.current) return;
        const rect = canvasContextRef.current.canvas.getBoundingClientRect();
        const x = e.clientX - rect.x;
        const y = e.clientY - rect.y;
        const mosaicSize = mosaicRef.current.data.size;
        const mosaicTiles = mosaicRef.current.data.tiles;
        let lastTile = mosaicTiles[mosaicTiles.length - 1];
        if (lastTile) {
            const dx = lastTile.x - x;
            const dy = lastTile.y - y;
            let length = Math.sqrt(dx ** 2 + dy ** 2);
            const sin = -dy / length;
            const cos = -dx / length;
            while(length > mosaicSize){
                const cx = Math.floor(lastTile.x + mosaicSize * cos);
                const cy = Math.floor(lastTile.y + mosaicSize * sin);
                lastTile = {
                    x: cx,
                    y: cy,
                    color: getColor(cx, cy, imageDataRef.current)
                };
                mosaicTiles.push(lastTile);
                length -= mosaicSize;
            }
            if (length > mosaicSize / 2) mosaicTiles.push({
                x,
                y,
                color: getColor(x, y, imageDataRef.current)
            });
        } else mosaicTiles.push({
            x,
            y,
            color: getColor(x, y, imageDataRef.current)
        });
        if (history.top !== mosaicRef.current) historyDispatcher.push(mosaicRef.current);
        else historyDispatcher.set(history);
    }, [
        checked,
        canvasContextRef,
        history,
        historyDispatcher
    ]);
    const onPointerUp = useCallback(()=>{
        if (!checked) return;
        mosaicRef.current = null;
    }, [
        checked
    ]);
    useCanvasPointerDown(onPointerDown);
    useCanvasPointerMove(onPointerMove);
    useCanvasPointerUp(onPointerUp);
    useEffect(()=>{
        if (!bounds || !image || !checked) return;
        const $canvas = document.createElement('canvas');
        const canvasContext = $canvas.getContext('2d');
        if (!canvasContext) return;
        $canvas.width = bounds.width;
        $canvas.height = bounds.height;
        const rx = image.naturalWidth / width;
        const ry = image.naturalHeight / height;
        canvasContext.drawImage(image, bounds.x * rx, bounds.y * ry, bounds.width * rx, bounds.height * ry, 0, 0, bounds.width, bounds.height);
        imageDataRef.current = canvasContext.getImageData(0, 0, bounds.width, bounds.height);
    }, [
        width,
        height,
        bounds,
        image,
        checked
    ]);
    return /*#__PURE__*/ React.createElement(Screenshots_ScreenshotsButton, {
        title: lang.operation_mosaic_title,
        icon: "icon-mosaic",
        checked: checked,
        onClick: onSelectMosaic,
        option: /*#__PURE__*/ React.createElement(Screenshots_ScreenshotsSize, {
            value: size,
            onChange: setSize
        })
    });
}
function Ok() {
    const { url, image, width, height, history, bounds, lang } = useStore();
    const [, historyDispatcher] = useHistory();
    const call = useCall();
    const reset = useReset();
    const onClick = useCallback(()=>{
        historyDispatcher.clearSelect();
        setTimeout(()=>{
            if (!bounds) return;
            composeImage({
                image,
                url,
                width,
                height,
                history,
                bounds
            }).then((blob)=>{
                call('onOk', blob, bounds);
                reset();
            });
        });
    }, [
        historyDispatcher,
        image,
        url,
        width,
        height,
        history,
        bounds,
        call,
        reset
    ]);
    return /*#__PURE__*/ React.createElement(Screenshots_ScreenshotsButton, {
        title: lang.operation_ok_title,
        icon: "icon-ok",
        onClick: onClick
    });
}
function getEditedRectangleData(action) {
    let { x1, y1, x2, y2 } = action.data;
    action.editHistory.forEach(({ data })=>{
        const x = data.x2 - data.x1;
        const y = data.y2 - data.y1;
        if (0 === data.type) {
            x1 += x;
            y1 += y;
            x2 += x;
            y2 += y;
        } else if (1 === data.type) y1 += y;
        else if (2 === data.type) {
            x2 += x;
            y1 += y;
        } else if (3 === data.type) x2 += x;
        else if (4 === data.type) {
            x2 += x;
            y2 += y;
        } else if (5 === data.type) y2 += y;
        else if (6 === data.type) {
            x1 += x;
            y2 += y;
        } else if (7 === data.type) x1 += x;
        else if (8 === data.type) {
            x1 += x;
            y1 += y;
        }
    });
    return {
        ...action.data,
        x1,
        x2,
        y1,
        y2
    };
}
function Rectangle_draw_draw(ctx, action) {
    const { size, color, x1, y1, x2, y2 } = getEditedRectangleData(action);
    ctx.lineCap = 'butt';
    ctx.lineJoin = 'miter';
    ctx.lineWidth = size;
    ctx.strokeStyle = color;
    ctx.beginPath();
    ctx.moveTo(x1, y1);
    ctx.lineTo(x2, y1);
    ctx.lineTo(x2, y2);
    ctx.lineTo(x1, y2);
    ctx.closePath();
    ctx.stroke();
    if (action.isSelected) {
        ctx.lineWidth = 1;
        ctx.strokeStyle = '#000000';
        ctx.fillStyle = '#ffffff';
        drawDragCircle(ctx, (x1 + x2) / 2, y1);
        drawDragCircle(ctx, x2, y1);
        drawDragCircle(ctx, x2, (y1 + y2) / 2);
        drawDragCircle(ctx, x2, y2);
        drawDragCircle(ctx, (x1 + x2) / 2, y2);
        drawDragCircle(ctx, x1, y2);
        drawDragCircle(ctx, x1, (y1 + y2) / 2);
        drawDragCircle(ctx, x1, y1);
    }
}
function Rectangle() {
    const lang = useLang();
    const [history, historyDispatcher] = useHistory();
    const [operation, operationDispatcher] = useOperation();
    const [, cursorDispatcher] = useCursor();
    const canvasContextRef = useCanvasContextRef();
    const [size, setSize] = useState(3);
    const [color, setColor] = useState('#ee5126');
    const rectangleRef = useRef(null);
    const rectangleEditRef = useRef(null);
    const checked = 'Rectangle' === operation;
    const selectRectangle = useCallback(()=>{
        operationDispatcher.set('Rectangle');
        cursorDispatcher.set('crosshair');
    }, [
        operationDispatcher,
        cursorDispatcher
    ]);
    const onSelectRectangle = useCallback(()=>{
        if (checked) return;
        selectRectangle();
        historyDispatcher.clearSelect();
    }, [
        checked,
        selectRectangle,
        historyDispatcher
    ]);
    const onDrawSelect = useCallback((action, e)=>{
        if ('Rectangle' !== action.name || !canvasContextRef.current) return;
        const source = action;
        selectRectangle();
        const { x1, y1, x2, y2 } = getEditedRectangleData(source);
        let type = 0;
        if (isHitCircle(canvasContextRef.current.canvas, e, {
            x: (x1 + x2) / 2,
            y: y1
        })) type = 1;
        else if (isHitCircle(canvasContextRef.current.canvas, e, {
            x: x2,
            y: y1
        })) type = 2;
        else if (isHitCircle(canvasContextRef.current.canvas, e, {
            x: x2,
            y: (y1 + y2) / 2
        })) type = 3;
        else if (isHitCircle(canvasContextRef.current.canvas, e, {
            x: x2,
            y: y2
        })) type = 4;
        else if (isHitCircle(canvasContextRef.current.canvas, e, {
            x: (x1 + x2) / 2,
            y: y2
        })) type = 5;
        else if (isHitCircle(canvasContextRef.current.canvas, e, {
            x: x1,
            y: y2
        })) type = 6;
        else if (isHitCircle(canvasContextRef.current.canvas, e, {
            x: x1,
            y: (y1 + y2) / 2
        })) type = 7;
        else if (isHitCircle(canvasContextRef.current.canvas, e, {
            x: x1,
            y: y1
        })) type = 8;
        rectangleEditRef.current = {
            type: 0,
            data: {
                type,
                x1: e.clientX,
                y1: e.clientY,
                x2: e.clientX,
                y2: e.clientY
            },
            source: action
        };
        historyDispatcher.select(action);
    }, [
        canvasContextRef,
        selectRectangle,
        historyDispatcher
    ]);
    const onPointerDown = useCallback((e)=>{
        if (!checked || !canvasContextRef.current || rectangleRef.current) return;
        const { left, top } = canvasContextRef.current.canvas.getBoundingClientRect();
        const x = e.clientX - left;
        const y = e.clientY - top;
        rectangleRef.current = {
            name: 'Rectangle',
            type: 1,
            data: {
                size,
                color,
                x1: x,
                y1: y,
                x2: x,
                y2: y
            },
            editHistory: [],
            draw: Rectangle_draw_draw,
            isHit: isHit
        };
    }, [
        checked,
        size,
        color,
        canvasContextRef
    ]);
    const onPointerMove = useCallback((e)=>{
        if (!checked || !canvasContextRef.current) return;
        if (rectangleEditRef.current) {
            rectangleEditRef.current.data.x2 = e.clientX;
            rectangleEditRef.current.data.y2 = e.clientY;
            if (history.top !== rectangleEditRef.current) {
                rectangleEditRef.current.source.editHistory.push(rectangleEditRef.current);
                historyDispatcher.push(rectangleEditRef.current);
            } else historyDispatcher.set(history);
        } else if (rectangleRef.current) {
            const { left, top } = canvasContextRef.current.canvas.getBoundingClientRect();
            const rectangleData = rectangleRef.current.data;
            rectangleData.x2 = e.clientX - left;
            rectangleData.y2 = e.clientY - top;
            if (history.top !== rectangleRef.current) historyDispatcher.push(rectangleRef.current);
            else historyDispatcher.set(history);
        }
    }, [
        checked,
        canvasContextRef,
        history,
        historyDispatcher
    ]);
    const onPointerUp = useCallback(()=>{
        if (!checked) return;
        if (rectangleRef.current) historyDispatcher.clearSelect();
        rectangleRef.current = null;
        rectangleEditRef.current = null;
    }, [
        checked,
        historyDispatcher
    ]);
    useDrawSelect(onDrawSelect);
    useCanvasPointerDown(onPointerDown);
    useCanvasPointerMove(onPointerMove);
    useCanvasPointerUp(onPointerUp);
    return /*#__PURE__*/ React.createElement(Screenshots_ScreenshotsButton, {
        title: lang.operation_rectangle_title,
        icon: "icon-rectangle",
        checked: checked,
        onClick: onSelectRectangle,
        option: /*#__PURE__*/ React.createElement(Screenshots_ScreenshotsSizeColor, {
            size: size,
            color: color,
            onSizeChange: setSize,
            onColorChange: setColor
        })
    });
}
function Redo() {
    const lang = useLang();
    const [history, historyDispatcher] = useHistory();
    const onClick = useCallback(()=>{
        historyDispatcher.redo();
    }, [
        historyDispatcher
    ]);
    return /*#__PURE__*/ React.createElement(Screenshots_ScreenshotsButton, {
        title: lang.operation_redo_title,
        icon: "icon-redo",
        disabled: !history.stack.length || history.stack.length - 1 === history.index,
        onClick: onClick
    });
}
function Save() {
    const { url, image, width, height, history, bounds, lang } = useStore();
    const [, historyDispatcher] = useHistory();
    const call = useCall();
    const reset = useReset();
    const onClick = useCallback(()=>{
        historyDispatcher.clearSelect();
        setTimeout(()=>{
            if (!bounds) return;
            composeImage({
                image,
                url,
                width,
                height,
                history,
                bounds
            }).then((blob)=>{
                call('onSave', blob, bounds);
                reset();
            });
        });
    }, [
        historyDispatcher,
        image,
        url,
        width,
        height,
        history,
        bounds,
        call,
        reset
    ]);
    return /*#__PURE__*/ React.createElement(Screenshots_ScreenshotsButton, {
        title: lang.operation_save_title,
        icon: "icon-save",
        onClick: onClick
    });
}
const hiddenTextareaStyle = `
min-width: 0 !important;
width: 0 !important;
min-height: 0 !important;
height:0 !important;
visibility: hidden !important;
overflow: hidden !important;
position: absolute !important;
z-index: -1000 !important;
top:0 !important;
right:0 !important;
`;
const sizeStyle = [
    'letter-spacing',
    'line-height',
    'padding-top',
    'padding-bottom',
    'font-family',
    'font-weight',
    'font-size',
    'font-variant',
    'text-rendering',
    'text-transform',
    'text-indent',
    'padding-left',
    'padding-right',
    'border-width',
    'box-sizing',
    'white-space',
    'word-break'
];
let hiddenTextarea;
function getComputedSizeInfo(node) {
    const style = window.getComputedStyle(node);
    const boxSizing = style.getPropertyValue('box-sizing') || style.getPropertyValue('-moz-box-sizing') || style.getPropertyValue('-webkit-box-sizing');
    const paddingSize = parseFloat(style.getPropertyValue('padding-bottom')) + parseFloat(style.getPropertyValue('padding-top'));
    const borderSize = parseFloat(style.getPropertyValue('border-bottom-width')) + parseFloat(style.getPropertyValue('border-top-width'));
    const sizingStyle = sizeStyle.map((name)=>`${name}:${style.getPropertyValue(name)}`).join(';');
    return {
        sizingStyle,
        paddingSize,
        borderSize,
        boxSizing
    };
}
function calculateNodeSize(textarea, value, maxWidth, maxHeight) {
    if (!hiddenTextarea) {
        hiddenTextarea = document.createElement('textarea');
        hiddenTextarea.setAttribute('tab-index', '-1');
        document.body.appendChild(hiddenTextarea);
    }
    const { paddingSize, borderSize, boxSizing, sizingStyle } = getComputedSizeInfo(textarea);
    hiddenTextarea.setAttribute('style', `${sizingStyle};${hiddenTextareaStyle};max-width:${maxWidth}px;max-height:${maxHeight}px`);
    hiddenTextarea.value = value || ' ';
    let width = hiddenTextarea.scrollWidth;
    let height = hiddenTextarea.scrollHeight;
    if ('border-box' === boxSizing) {
        width += borderSize;
        height += borderSize;
    } else if ('content-box' === boxSizing) {
        width -= paddingSize;
        height -= paddingSize;
    }
    return {
        width: Math.min(width, maxWidth),
        height: Math.min(height, maxHeight)
    };
}
const Screenshots_ScreenshotsTextarea = /*#__PURE__*/ memo(function({ x, y, maxWidth, maxHeight, size, color, value, onChange, onBlur }) {
    const popoverRef = useRef(null);
    const textareaRef = useRef(null);
    const [width, setWidth] = useState(0);
    const [height, setHeight] = useState(0);
    const getPopoverEl = ()=>{
        if (!popoverRef.current) popoverRef.current = document.createElement('div');
        return popoverRef.current;
    };
    useLayoutEffect(()=>{
        if (popoverRef.current) {
            document.body.appendChild(popoverRef.current);
            requestAnimationFrame(()=>{
                textareaRef.current?.focus();
            });
        }
        return ()=>{
            popoverRef.current?.remove();
        };
    }, []);
    useLayoutEffect(()=>{
        if (!textareaRef.current) return;
        const { width, height } = calculateNodeSize(textareaRef.current, value, maxWidth, maxHeight);
        setWidth(width);
        setHeight(height);
    }, [
        value,
        maxWidth,
        maxHeight
    ]);
    return /*#__PURE__*/ createPortal(/*#__PURE__*/ React.createElement("textarea", {
        ref: textareaRef,
        className: "screenshots-textarea",
        style: {
            color,
            width,
            height,
            maxWidth,
            maxHeight,
            fontSize: size,
            lineHeight: `${size}px`,
            transform: `translate(${x}px, ${y}px)`
        },
        value: value,
        onChange: (e)=>onChange?.(e.target.value),
        onBlur: (e)=>onBlur?.(e)
    }), getPopoverEl());
});
const Text_sizes = {
    3: 18,
    6: 32,
    9: 46
};
function Text_draw(ctx, action) {
    const { size, color, fontFamily, x, y, text } = action.data;
    ctx.fillStyle = color;
    ctx.textAlign = 'left';
    ctx.textBaseline = 'top';
    ctx.font = `${size}px ${fontFamily}`;
    const distance = action.editHistory.reduce((distance, { data })=>({
            x: distance.x + data.x2 - data.x1,
            y: distance.y + data.y2 - data.y1
        }), {
        x: 0,
        y: 0
    });
    text.split('\n').forEach((item, index)=>{
        ctx.fillText(item, x + distance.x, y + distance.y + index * size);
    });
}
function Text_isHit(ctx, action, point) {
    ctx.textAlign = 'left';
    ctx.textBaseline = 'top';
    ctx.font = `${action.data.size}px ${action.data.fontFamily}`;
    let width = 0;
    let height = 0;
    action.data.text.split('\n').forEach((item)=>{
        const measured = ctx.measureText(item);
        if (width < measured.width) width = measured.width;
        height += action.data.size;
    });
    const { x, y } = action.editHistory.reduce((distance, { data })=>({
            x: distance.x + data.x2 - data.x1,
            y: distance.y + data.y2 - data.y1
        }), {
        x: 0,
        y: 0
    });
    const left = action.data.x + x;
    const top = action.data.y + y;
    const right = left + width;
    const bottom = top + height;
    return point.x >= left && point.x <= right && point.y >= top && point.y <= bottom;
}
function Text() {
    const lang = useLang();
    const [history, historyDispatcher] = useHistory();
    const [bounds] = useBounds();
    const [operation, operationDispatcher] = useOperation();
    const [, cursorDispatcher] = useCursor();
    const canvasContextRef = useCanvasContextRef();
    const [size, setSize] = useState(3);
    const [color, setColor] = useState('#ee5126');
    const textRef = useRef(null);
    const textEditRef = useRef(null);
    const [textareaBounds, setTextareaBounds] = useState(null);
    const [text, setText] = useState('');
    const checked = 'Text' === operation;
    const selectText = useCallback(()=>{
        operationDispatcher.set('Text');
        cursorDispatcher.set('default');
    }, [
        operationDispatcher,
        cursorDispatcher
    ]);
    const onSelectText = useCallback(()=>{
        if (checked) return;
        selectText();
        historyDispatcher.clearSelect();
    }, [
        checked,
        selectText,
        historyDispatcher
    ]);
    const onSizeChange = useCallback((size)=>{
        if (textRef.current) textRef.current.data.size = Text_sizes[size];
        setSize(size);
    }, []);
    const onColorChange = useCallback((color)=>{
        if (textRef.current) textRef.current.data.color = color;
        setColor(color);
    }, []);
    const onTextareaChange = useCallback((value)=>{
        setText(value);
        if (checked && textRef.current) textRef.current.data.text = value;
    }, [
        checked
    ]);
    const onTextareaBlur = useCallback(()=>{
        if (textRef.current?.data.text) historyDispatcher.push(textRef.current);
        textRef.current = null;
        setText('');
        setTextareaBounds(null);
    }, [
        historyDispatcher
    ]);
    const onDrawSelect = useCallback((action, e)=>{
        if ('Text' !== action.name) return;
        selectText();
        textEditRef.current = {
            type: 0,
            data: {
                x1: e.clientX,
                y1: e.clientY,
                x2: e.clientX,
                y2: e.clientY
            },
            source: action
        };
        historyDispatcher.select(action);
    }, [
        selectText,
        historyDispatcher
    ]);
    const onPointerDown = useCallback((e)=>{
        if (!checked || !canvasContextRef.current || textRef.current || !bounds) return;
        const { left, top } = canvasContextRef.current.canvas.getBoundingClientRect();
        const fontFamily = window.getComputedStyle(canvasContextRef.current.canvas).fontFamily;
        const x = e.clientX - left;
        const y = e.clientY - top;
        textRef.current = {
            name: 'Text',
            type: 1,
            data: {
                size: Text_sizes[size],
                color,
                fontFamily,
                x,
                y,
                text: ''
            },
            editHistory: [],
            draw: Text_draw,
            isHit: Text_isHit
        };
        setTextareaBounds({
            x: e.clientX,
            y: e.clientY,
            maxWidth: bounds.width - x,
            maxHeight: bounds.height - y
        });
    }, [
        checked,
        size,
        color,
        bounds,
        canvasContextRef
    ]);
    const onPointerMove = useCallback((e)=>{
        if (!checked) return;
        if (textEditRef.current) {
            textEditRef.current.data.x2 = e.clientX;
            textEditRef.current.data.y2 = e.clientY;
            if (history.top !== textEditRef.current) {
                textEditRef.current.source.editHistory.push(textEditRef.current);
                historyDispatcher.push(textEditRef.current);
            } else historyDispatcher.set(history);
        }
    }, [
        checked,
        history,
        historyDispatcher
    ]);
    const onPointerUp = useCallback(()=>{
        if (!checked) return;
        textEditRef.current = null;
    }, [
        checked
    ]);
    useDrawSelect(onDrawSelect);
    useCanvasPointerDown(onPointerDown);
    useCanvasPointerMove(onPointerMove);
    useCanvasPointerUp(onPointerUp);
    return /*#__PURE__*/ React.createElement(React.Fragment, null, /*#__PURE__*/ React.createElement(Screenshots_ScreenshotsButton, {
        title: lang.operation_text_title,
        icon: "icon-text",
        checked: checked,
        onClick: onSelectText,
        option: /*#__PURE__*/ React.createElement(Screenshots_ScreenshotsSizeColor, {
            size: size,
            color: color,
            onSizeChange: onSizeChange,
            onColorChange: onColorChange
        })
    }), checked && textareaBounds && /*#__PURE__*/ React.createElement(Screenshots_ScreenshotsTextarea, {
        x: textareaBounds.x,
        y: textareaBounds.y,
        maxWidth: textareaBounds.maxWidth,
        maxHeight: textareaBounds.maxHeight,
        size: Text_sizes[size],
        color: color,
        value: text,
        onChange: onTextareaChange,
        onBlur: onTextareaBlur
    }));
}
function Undo() {
    const lang = useLang();
    const [history, historyDispatcher] = useHistory();
    const onClick = useCallback(()=>{
        historyDispatcher.undo();
    }, [
        historyDispatcher
    ]);
    return /*#__PURE__*/ React.createElement(Screenshots_ScreenshotsButton, {
        title: lang.operation_undo_title,
        icon: "icon-undo",
        disabled: -1 === history.index,
        onClick: onClick
    });
}
const operations = [
    Rectangle,
    Ellipse,
    Arrow,
    Brush,
    Text,
    Mosaic,
    '|',
    Undo,
    Redo,
    '|',
    Save,
    Cancel,
    Ok
];
const ScreenshotsOperationsCtx = /*#__PURE__*/ react.createContext(null);
const Screenshots_ScreenshotsOperations = /*#__PURE__*/ memo(function() {
    const { width, height } = useStore();
    const [bounds] = useBounds();
    const [operationsRect, setOperationsRect] = useState(null);
    const [position, setPosition] = useState(null);
    const elRef = useRef(null);
    const onDoubleClick = useCallback((e)=>{
        e.stopPropagation();
    }, []);
    const onContextMenu = useCallback((e)=>{
        e.preventDefault();
        e.stopPropagation();
    }, []);
    useEffect(()=>{
        if (!bounds || !elRef.current) return;
        const elRect = elRef.current.getBoundingClientRect();
        let x = bounds.x + bounds.width - elRect.width;
        let y = bounds.y + bounds.height + 10;
        if (x < 0) x = 0;
        if (x > width - elRect.width) x = width - elRect.width;
        if (y > height - elRect.height) y = height - elRect.height - 10;
        setPosition((prev)=>{
            if (prev && Math.abs(prev.x - x) <= 1 && Math.abs(prev.y - y) <= 1) return prev;
            return {
                x,
                y
            };
        });
        setOperationsRect((prev)=>{
            if (prev && Math.abs(prev.x - elRect.x) <= 1 && Math.abs(prev.y - elRect.y) <= 1 && Math.abs(prev.width - elRect.width) <= 1 && Math.abs(prev.height - elRect.height) <= 1) return prev;
            return {
                x: elRect.x,
                y: elRect.y,
                width: elRect.width,
                height: elRect.height
            };
        });
    }, [
        bounds,
        width,
        height
    ]);
    if (!bounds) return null;
    return /*#__PURE__*/ react.createElement(ScreenshotsOperationsCtx.Provider, {
        value: operationsRect
    }, /*#__PURE__*/ react.createElement("div", {
        ref: elRef,
        className: "screenshots-operations",
        style: {
            visibility: position ? 'visible' : 'hidden',
            transform: `translate(${position?.x ?? 0}px, ${position?.y ?? 0}px)`
        },
        onDoubleClick: onDoubleClick,
        onContextMenu: onContextMenu
    }, /*#__PURE__*/ react.createElement("div", {
        className: "screenshots-operations-buttons"
    }, operations.map((OperationButton, index)=>{
        if ('|' === OperationButton) return /*#__PURE__*/ react.createElement("div", {
            key: index,
            className: "screenshots-operations-divider"
        });
        return /*#__PURE__*/ react.createElement(OperationButton, {
            key: index
        });
    }))));
});
function useGetLoadedImage(url) {
    const [image, setImage] = useState(null);
    const prevUrlRef = useRef(null);
    useEffect(()=>{
        if (url === prevUrlRef.current) return;
        prevUrlRef.current = url ?? null;
        if (null == url) return void setImage(null);
        const $image = document.createElement('img');
        $image.decoding = 'async';
        const onLoad = ()=>setImage($image);
        const onError = ()=>setImage(null);
        $image.addEventListener('load', onLoad);
        $image.addEventListener('error', onError);
        $image.src = url;
        return ()=>{
            $image.removeEventListener('load', onLoad);
            $image.removeEventListener('error', onError);
        };
    }, [
        url
    ]);
    return image;
}
function Screenshots({ url, width, height, lang, initialPosition, className, ...props }) {
    const propsRef = useRef(props);
    propsRef.current = props;
    const image = useGetLoadedImage(url);
    const canvasContextRef = useRef(null);
    const emitterRef = useRef({});
    const [history, setHistory] = useState({
        index: -1,
        stack: []
    });
    const [bounds, setBounds] = useState(null);
    const [cursor, setCursor] = useState('move');
    const [operation, setOperation] = useState(void 0);
    const store = useMemo(()=>({
            url,
            width,
            height,
            image,
            lang: {
                ...zh_CN,
                ...lang
            },
            emitterRef,
            canvasContextRef,
            history,
            bounds,
            cursor,
            operation,
            initialPosition
        }), [
        url,
        width,
        height,
        image,
        lang,
        history,
        bounds,
        cursor,
        operation,
        initialPosition
    ]);
    const call = useCallback((funcName, ...args)=>{
        const func = propsRef.current[funcName];
        if ('function' == typeof func) func(...args);
    }, []);
    const dispatcher = useMemo(()=>({
            call,
            setHistory,
            setBounds,
            setCursor,
            setOperation
        }), [
        call
    ]);
    const classNames = [
        'screenshots'
    ];
    if (className) classNames.push(className);
    const reset = useCallback(()=>{
        emitterRef.current = {};
        setHistory({
            index: -1,
            stack: []
        });
        setBounds(null);
        setCursor('move');
        setOperation(void 0);
    }, []);
    const onDoubleClick = useCallback(async (e)=>{
        if (0 !== e.button || !image) return;
        if (bounds && canvasContextRef.current) composeImage({
            image,
            url,
            width,
            height,
            history,
            bounds
        }).then((blob)=>{
            call('onOk', blob, bounds);
            reset();
        });
        else {
            const targetBounds = {
                x: 0,
                y: 0,
                width,
                height
            };
            composeImage({
                image,
                url,
                width,
                height,
                history,
                bounds: targetBounds
            }).then((blob)=>{
                call('onOk', blob, targetBounds);
                reset();
            });
        }
    }, [
        image,
        url,
        history,
        bounds,
        width,
        height,
        call,
        reset
    ]);
    const onContextMenu = useCallback((e)=>{
        if (2 !== e.button) return;
        e.preventDefault();
        call('onCancel');
        reset();
    }, [
        call,
        reset
    ]);
    useLayoutEffect(()=>{
        reset();
    }, [
        url
    ]);
    return /*#__PURE__*/ React.createElement(ScreenshotsContext.Provider, {
        value: {
            store,
            dispatcher
        }
    }, /*#__PURE__*/ React.createElement("div", {
        className: classNames.join(' '),
        style: {
            width,
            height
        },
        onDoubleClick: onDoubleClick,
        onContextMenu: onContextMenu
    }, /*#__PURE__*/ React.createElement(Screenshots_ScreenshotsBackground, null), /*#__PURE__*/ React.createElement(Screenshots_ScreenshotsCanvas, {
        ref: canvasContextRef
    }), /*#__PURE__*/ React.createElement(Screenshots_ScreenshotsOperations, null)));
}
export default Screenshots;
