import {Rendering} from "../../domain/displaySettings.ts"

export type Graphics = {
    ratio: boolean
    antialias: boolean
    earth: boolean
    tiles: boolean
    halo: boolean
}

export const PLAIN: Graphics = {ratio: false, antialias: false, earth: false, tiles: false, halo: false}

export const SHARP: Graphics = {ratio: true, antialias: true, earth: true, tiles: true, halo: true}

// A Map, not an object, so ?gfx=constructor finds nothing rather than a function.
const WORDS = new Map<string, (keyof Graphics)[]>([
    ["ratio", ["ratio"]],
    ["aa", ["antialias"]],
    ["earth", ["earth"]],
    ["tiles", ["tiles"]],
    ["halo", ["halo"]],
    ["light", ["earth", "tiles", "halo"]],
    ["all", ["ratio", "antialias", "earth", "tiles", "halo"]],
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
