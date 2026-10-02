/**
 * The parts of the sharper, lit globe (#254) that are drawn only when the URL
 * asks for them, one at a time, with `?gfx=`.
 *
 * #254 turned the globe almost white, flickering as it turned, for players on
 * Windows Chrome with an Intel GPU (ANGLE on Direct3D 11), and was reverted.
 * Turning antialiasing off (#255) did not fix it, and the same GPU on another
 * machine shows nothing wrong, so the cause can only be found on the screens
 * that have it. Each part is a switch so those players can turn them on one at
 * a time and say which one breaks their picture.
 *
 * Off, a part is not merely multiplied by zero: its shader code is compiled out
 * (`#ifdef LIT`), so a driver that miscompiles it never sees it.
 */
export type Graphics = {
    /** Draw at the screen's pixel ratio, capped at 2, rather than at 1. */
    ratio: boolean
    /** Ask for a multisampled drawing buffer. */
    antialias: boolean
    /** The earth in the globe's light, with the glint on the sea. */
    earth: boolean
    /** The tiles in the globe's light. */
    tiles: boolean
    /** The halo in the globe's light. */
    halo: boolean
}

/** What the globe is drawn with when the URL asks for nothing. */
export const PLAIN: Graphics = {ratio: false, antialias: false, earth: false, tiles: false, halo: false}

/**
 * Each word `?gfx=` takes, and the parts it turns on. A `Map` and not an
 * object, so `?gfx=constructor` finds nothing rather than a function.
 */
const WORDS = new Map<string, (keyof Graphics)[]>([
    ["ratio", ["ratio"]],
    ["aa", ["antialias"]],
    ["earth", ["earth"]],
    ["tiles", ["tiles"]],
    ["halo", ["halo"]],
    ["light", ["earth", "tiles", "halo"]],
    ["all", ["ratio", "antialias", "earth", "tiles", "halo"]],
])

/**
 * The parts a page's query string turns on: `?gfx=` and a comma-separated list
 * of the words above, such as `?gfx=ratio,tiles`. A word it does not know turns
 * nothing on.
 */
export function graphicsOf(search: string): Graphics {
    const graphics = {...PLAIN}
    const words = new URLSearchParams(search).get("gfx") ?? ""
    for (const word of words.split(",")) {
        for (const part of WORDS.get(word.trim().toLowerCase()) ?? []) graphics[part] = true
    }
    return graphics
}
