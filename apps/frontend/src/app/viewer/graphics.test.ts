import {describe, expect, it} from "vitest"
import {graphicsOf, PLAIN, SHARP} from "./graphics.ts"

describe("graphicsOf", () => {
    it("follows the setting when the URL asks for nothing", () => {
        expect(graphicsOf("", "sharp")).toEqual(SHARP)
        expect(graphicsOf("?f=fr", "sharp")).toEqual(SHARP)
        expect(graphicsOf("", "soft")).toEqual(PLAIN)
        expect(graphicsOf("?f=fr", "soft")).toEqual(PLAIN)
    })

    it("antialiases only when the URL asks for it", () => {
        expect(graphicsOf("", "sharp").antialias).toBe(false)
        expect(graphicsOf("?gfx=all", "sharp").antialias).toBe(false)
        expect(graphicsOf("?gfx=all,aa", "soft")).toEqual({...SHARP, ratio: "screen", antialias: true})
    })

    it("lets the URL override the setting either way", () => {
        expect(graphicsOf("?gfx=", "sharp")).toEqual(PLAIN)
        expect(graphicsOf("?gfx=earth", "sharp")).toEqual({...PLAIN, earth: true})
        expect(graphicsOf("?gfx=all", "soft")).toEqual({...SHARP, ratio: "screen"})
    })

    it("lowers the ratio on slow frames only for the setting, never for the URL", () => {
        expect(graphicsOf("", "sharp").ratio).toBe("adaptive")
        expect(graphicsOf("?gfx=ratio", "sharp").ratio).toBe("screen")
        expect(graphicsOf("?gfx=all", "sharp").ratio).toBe("screen")
    })

    it("turns on each part its word names, and no other", () => {
        expect(graphicsOf("?gfx=ratio", "sharp")).toEqual({...PLAIN, ratio: "screen"})
        expect(graphicsOf("?gfx=aa", "sharp")).toEqual({...PLAIN, antialias: true})
        expect(graphicsOf("?gfx=earth", "sharp")).toEqual({...PLAIN, earth: true})
        expect(graphicsOf("?gfx=compact", "soft")).toEqual({...PLAIN, compact: true})
    })

    it("adds up a list, whatever its case and spacing", () => {
        expect(graphicsOf("?gfx=Ratio,%20earth", "soft")).toEqual({...PLAIN, ratio: "screen", earth: true})
    })

    it("reads the switch beside the rest of the query", () => {
        expect(graphicsOf("?f=fr&gfx=earth", "soft")).toEqual({...PLAIN, earth: true})
    })

    it("turns nothing on for a word it does not know", () => {
        expect(graphicsOf("?gfx=sharp,constructor,__proto__", "sharp")).toEqual(PLAIN)
    })
})
