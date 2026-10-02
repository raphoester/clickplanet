export type ViewportSize = {
    width: number
    height: number
}

export function layoutViewport(): ViewportSize {
    const root = document.documentElement

    // Not window.innerWidth first: iOS Safari shrinks it to the pinch-zoomed area.
    return {
        width: root.clientWidth || window.innerWidth,
        height: root.clientHeight || window.innerHeight,
    }
}
