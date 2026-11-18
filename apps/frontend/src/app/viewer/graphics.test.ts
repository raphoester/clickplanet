import {describe, expect, it} from "vitest"
import {graphicsOf, PLAIN, sharpOf} from "./graphics.ts"

const SHARP = sharpOf(false)

describe("graphicsOf", () => {
    it("follows the setting when the URL asks for nothing", () => {
        expect(graphicsOf("", "sharp", false)).toEqual(SHARP)
        expect(graphicsOf("?f=fr", "sharp", false)).toEqual(SHARP)
        expect(graphicsOf("", "soft", false)).toEqual(PLAIN)
        expect(graphicsOf("?f=fr", "soft", true)).toEqual(PLAIN)
    })

    it("antialiases only when the URL asks for it", () => {
        expect(graphicsOf("", "sharp", false).antialias).toBe(false)
        expect(graphicsOf("?gfx=all", "sharp", false).antialias).toBe(false)
        expect(graphicsOf("?gfx=all,aa", "soft", false)).toEqual({...SHARP, antialias: true})
    })

    it("lets the URL override the setting either way", () => {
        expect(graphicsOf("?gfx=", "sharp", false)).toEqual(PLAIN)
        expect(graphicsOf("?gfx=earth", "sharp", false)).toEqual({...PLAIN, earth: true})
        expect(graphicsOf("?gfx=all", "soft", false)).toEqual(SHARP)
    })

    it("lowers the ratio on slow frames for the setting only when the build turns it on", () => {
        expect(graphicsOf("", "sharp", true).ratio).toBe("adaptive")
        expect(graphicsOf("", "sharp", false).ratio).toBe("screen")
    })

    it("lowers the ratio on slow frames for the URL only when it names adaptive", () => {
        expect(graphicsOf("?gfx=ratio", "sharp", true).ratio).toBe("screen")
        expect(graphicsOf("?gfx=all", "sharp", true).ratio).toBe("screen")
        expect(graphicsOf("?gfx=adaptive,earth,compact", "soft", false)).toEqual(sharpOf(true))
    })

    it("turns on each part its word names, and no other", () => {
        expect(graphicsOf("?gfx=ratio", "sharp", false)).toEqual({...PLAIN, ratio: "screen"})
        expect(graphicsOf("?gfx=adaptive", "sharp", false)).toEqual({...PLAIN, ratio: "adaptive"})
        expect(graphicsOf("?gfx=aa", "sharp", false)).toEqual({...PLAIN, antialias: true})
        expect(graphicsOf("?gfx=earth", "sharp", false)).toEqual({...PLAIN, earth: true})
        expect(graphicsOf("?gfx=compact", "soft", false)).toEqual({...PLAIN, compact: true})
    })

    it("adds up a list, whatever its case and spacing", () => {
        expect(graphicsOf("?gfx=Ratio,%20earth", "soft", false)).toEqual({...PLAIN, ratio: "screen", earth: true})
    })

    it("reads the switch beside the rest of the query", () => {
        expect(graphicsOf("?f=fr&gfx=earth", "soft", false)).toEqual({...PLAIN, earth: true})
    })

    it("turns nothing on for a word it does not know", () => {
        expect(graphicsOf("?gfx=sharp,constructor,__proto__", "sharp", false)).toEqual(PLAIN)
    })
})
