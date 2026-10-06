import {describe, expect, it} from "vitest"
import {graphicsOf, PLAIN, SHARP} from "./graphics.ts"

describe("graphicsOf", () => {
    it("follows the setting when the URL asks for nothing", () => {
        expect(graphicsOf("", "sharp")).toEqual(SHARP)
        expect(graphicsOf("?f=fr", "sharp")).toEqual(SHARP)
        expect(graphicsOf("", "plain")).toEqual(PLAIN)
        expect(graphicsOf("?f=fr", "plain")).toEqual(PLAIN)
    })

    it("is #254 whole when sharp", () => {
        expect(graphicsOf("?gfx=all", "plain")).toEqual(SHARP)
    })

    it("lets the URL override the setting either way", () => {
        expect(graphicsOf("?gfx=", "sharp")).toEqual(PLAIN)
        expect(graphicsOf("?gfx=halo", "sharp")).toEqual({...PLAIN, halo: true})
        expect(graphicsOf("?gfx=all", "plain")).toEqual(SHARP)
    })

    it("turns on each part its word names, and no other", () => {
        expect(graphicsOf("?gfx=ratio", "sharp")).toEqual({...PLAIN, ratio: true})
        expect(graphicsOf("?gfx=aa", "sharp")).toEqual({...PLAIN, antialias: true})
        expect(graphicsOf("?gfx=earth", "sharp")).toEqual({...PLAIN, earth: true})
        expect(graphicsOf("?gfx=tiles", "sharp")).toEqual({...PLAIN, tiles: true})
        expect(graphicsOf("?gfx=halo", "sharp")).toEqual({...PLAIN, halo: true})
    })

    it("takes the light as its three parts", () => {
        expect(graphicsOf("?gfx=light", "plain")).toEqual({...PLAIN, earth: true, tiles: true, halo: true})
    })

    it("adds up a list, whatever its case and spacing", () => {
        expect(graphicsOf("?gfx=Ratio,%20tiles", "plain")).toEqual({...PLAIN, ratio: true, tiles: true})
    })

    it("reads the switch beside the rest of the query", () => {
        expect(graphicsOf("?f=fr&gfx=halo", "plain")).toEqual({...PLAIN, halo: true})
    })

    it("turns nothing on for a word it does not know", () => {
        expect(graphicsOf("?gfx=sharp,constructor,__proto__", "sharp")).toEqual(PLAIN)
    })
})
