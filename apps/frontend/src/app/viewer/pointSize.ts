export function tilePointSize(zoom: number, viewportHeight: number): number {
    return zoom * 1.5 * (viewportHeight / 1000)
}

export const MAX_PICK_WINDOW = 257

export function pickWindowSize(pointSize: number): number {
    const needed = Math.ceil(pointSize) + 1
    const odd = needed % 2 === 0 ? needed + 1 : needed
    return Math.min(Math.max(odd, 3), MAX_PICK_WINDOW)
}
