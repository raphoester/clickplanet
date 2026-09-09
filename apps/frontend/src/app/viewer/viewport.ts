export type ViewportSize = {
    width: number
    height: number
}

/**
 * The layout viewport in CSS pixels — the box the page is laid out against,
 * whatever zoom the browser is currently showing it at.
 *
 * `window.innerWidth` is the *visual* viewport on iOS Safari: zooming in makes
 * it report less than the page is wide, and a `resize` fires with the smaller
 * figure. Sizing the renderer from that shrinks the canvas to the zoomed
 * portion, and releasing the zoom fires nothing that would widen it again — the
 * globe is left short of the right edge with a black band beside it. The root
 * element's client box is the layout viewport and ignores zoom entirely, so the
 * canvas keeps covering the page through a pinch.
 *
 * jsdom leaves that box at zero, so `window` remains the fallback.
 */
export function layoutViewport(): ViewportSize {
    const root = document.documentElement

    return {
        width: root.clientWidth || window.innerWidth,
        height: root.clientHeight || window.innerHeight,
    }
}
