/**
 * Diameter in device pixels of the sprite drawn for one tile.
 *
 * Both vertex shaders read this as a uniform rather than recomputing it, and
 * `GpuPicker` sizes its picking window from it. Those have to agree: the picker
 * renders a window around the cursor and reads the middle pixel, so a sprite
 * wider than the window can cover that pixel while its own centre sits outside
 * it — and a point whose centre is outside the window is clipped entirely.
 */
export function tilePointSize(zoom: number, viewportHeight: number): number {
    return zoom * 1.5 * (viewportHeight / 1000)
}

/**
 * Widest window we will render for a pick, whatever the zoom.
 *
 * The sprite reaches its largest at maximum zoom on the tallest viewport:
 * OrbitControls caps zoom at 50, so a 3400px-tall window gives a 255px sprite.
 * Even at this size the window is a 66k-pixel render against the ~2.5M of the
 * viewport, so the cap is here to bound allocation, not to save time.
 */
export const MAX_PICK_WINDOW = 257

/**
 * Side of the square window `GpuPicker` renders, in pixels.
 *
 * It must be wide enough to contain the centre of any sprite that could cover
 * the middle pixel — that is `pointSize` across, plus the middle pixel itself —
 * and odd, so that there *is* a middle pixel.
 */
export function pickWindowSize(pointSize: number): number {
    const needed = Math.ceil(pointSize) + 1
    const odd = needed % 2 === 0 ? needed + 1 : needed
    return Math.min(Math.max(odd, 3), MAX_PICK_WINDOW)
}
