import {Rendering} from "../../domain/displaySettings.ts"

export type Graphics = {
    ratio: boolean
    antialias: boolean
    earth: boolean
    tiles: boolean
    halo: boolean
    compact: boolean
    etc1s: boolean
    uastc: boolean
}

export const PLAIN: Graphics = {ratio: false, antialias: false, earth: false, tiles: false, halo: false, compact: false, etc1s: false, uastc: false}

export const SHARP: Graphics = {ratio: true, antialias: true, earth: true, tiles: true, halo: true, compact: true, etc1s: false, uastc: false}

// A Map, not an object, so ?gfx=constructor finds nothing rather than a function.
const WORDS = new Map<string, (keyof Graphics)[]>([
    ["ratio", ["ratio"]],
    ["aa", ["antialias"]],
    ["earth", ["earth"]],
    ["tiles", ["tiles"]],
    ["halo", ["halo"]],
    ["compact", ["compact"]],
    ["etc1s", ["etc1s"]],
    ["uastc", ["uastc"]],
    ["light", ["earth", "tiles", "halo"]],
    ["all", ["ratio", "antialias", "earth", "tiles", "halo", "compact"]],
])

export function graphicsOf(search: string, rendering: Rendering): Graphics {
    const words = new URLSearchParams(search).get("gfx")
    if (words === null) return rendering === "sharp" ? SHARP : PLAIN

    const graphics = {...PLAIN}
    for (const word of words.split(",")) {
        for (const part of WORDS.get(word.trim().toLowerCase()) ?? []) graphics[part] = true
    }
    return graphics
}
