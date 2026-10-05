import {describe, expect, it} from "vitest"
import {graphicsOf, PLAIN} from "./graphics.ts"

describe("graphicsOf", () => {
    it("turns nothing on when the URL asks for nothing", () => {
        expect(graphicsOf("")).toEqual(PLAIN)
        expect(graphicsOf("?f=fr")).toEqual(PLAIN)
    })

    it("turns on each part its word names, and no other", () => {
        expect(graphicsOf("?gfx=ratio")).toEqual({...PLAIN, ratio: true})
        expect(graphicsOf("?gfx=aa")).toEqual({...PLAIN, antialias: true})
        expect(graphicsOf("?gfx=earth")).toEqual({...PLAIN, earth: true})
        expect(graphicsOf("?gfx=tiles")).toEqual({...PLAIN, tiles: true})
        expect(graphicsOf("?gfx=halo")).toEqual({...PLAIN, halo: true})
    })

    it("takes the light as its three parts, and all as #254 whole", () => {
        expect(graphicsOf("?gfx=light")).toEqual({...PLAIN, earth: true, tiles: true, halo: true})
        expect(graphicsOf("?gfx=all")).toEqual({ratio: true, antialias: true, earth: true, tiles: true, halo: true})
    })

    it("adds up a list, whatever its case and spacing", () => {
        expect(graphicsOf("?gfx=Ratio,%20tiles")).toEqual({...PLAIN, ratio: true, tiles: true})
    })

    it("reads the switch beside the rest of the query", () => {
        expect(graphicsOf("?f=fr&gfx=halo")).toEqual({...PLAIN, halo: true})
    })

    it("turns nothing on for a word it does not know", () => {
        expect(graphicsOf("?gfx=sharp,constructor,__proto__")).toEqual(PLAIN)
    })
})
