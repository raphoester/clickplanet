import {Rendering} from "../../domain/displaySettings.ts"

export type Graphics = {
    // "adaptive" is the screen's, lowered when the frames come slow; "screen" is the screen's alone.
    ratio: "one" | "screen" | "adaptive"
    antialias: boolean
    earth: boolean
    compact: boolean
}

export const PLAIN: Graphics = {ratio: "one", antialias: false, earth: false, compact: false}

// Built with VITE_ADAPTIVE_RATIO=on, HD graphics lowers its ratio on slow frames: a switch of its own, see
// docs/pixel-ratio.md.
export const ADAPTIVE_RATIO = import.meta.env.VITE_ADAPTIVE_RATIO === "on"

export function sharpOf(adaptive: boolean): Graphics {
    return {ratio: adaptive ? "adaptive" : "screen", antialias: false, earth: true, compact: true}
}

// A Map, not an object, so ?gfx=constructor finds nothing rather than a function.
const WORDS = new Map<string, Partial<Graphics>>([
    ["ratio", {ratio: "screen"}],
    ["adaptive", {ratio: "adaptive"}],
    ["aa", {antialias: true}],
    ["earth", {earth: true}],
    ["compact", {compact: true}],
    ["all", {ratio: "screen", earth: true, compact: true}],
])

export function graphicsOf(search: string, rendering: Rendering, adaptive: boolean = ADAPTIVE_RATIO): Graphics {
    const words = new URLSearchParams(search).get("gfx")
    if (words === null) return rendering === "sharp" ? sharpOf(adaptive) : PLAIN

    const graphics = {...PLAIN}
    for (const word of words.split(",")) Object.assign(graphics, WORDS.get(word.trim().toLowerCase()))
    return graphics
}
