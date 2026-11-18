import {Rendering} from "../../domain/displaySettings.ts"

export type Graphics = {
    // "adaptive" is the screen's, lowered when the frames come slow; "screen" is the screen's alone.
    ratio: "one" | "screen" | "adaptive"
    antialias: boolean
    earth: boolean
    compact: boolean
}

export const PLAIN: Graphics = {ratio: "one", antialias: false, earth: false, compact: false}

export const SHARP: Graphics = {ratio: "adaptive", antialias: false, earth: true, compact: true}

// A Map, not an object, so ?gfx=constructor finds nothing rather than a function.
const WORDS = new Map<string, Partial<Graphics>>([
    ["ratio", {ratio: "screen"}],
    ["aa", {antialias: true}],
    ["earth", {earth: true}],
    ["compact", {compact: true}],
    ["all", {ratio: "screen", earth: true, compact: true}],
])

export function graphicsOf(search: string, rendering: Rendering): Graphics {
    const words = new URLSearchParams(search).get("gfx")
    if (words === null) return rendering === "sharp" ? SHARP : PLAIN

    const graphics = {...PLAIN}
    for (const word of words.split(",")) Object.assign(graphics, WORDS.get(word.trim().toLowerCase()))
    return graphics
}
