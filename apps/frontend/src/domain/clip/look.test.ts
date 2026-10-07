import {describe, expect, it} from "vitest"
import {Flip, flipsOf, lookOf, TILES_ZOOM} from "./look.ts"

const landmassOf = [...Array(30).fill(1), ...Array(30).fill(2), ...Array(5).fill(3), 0]

function holding(owner: string, from: number, to: number): Map<number, string> {
    return new Map(Array.from({length: to - from + 1}, (_, i) => [from + i, owner]))
}

describe("the landmasses that changed hands", () => {
    it("are the ones whose biggest holder changed", () => {
        const before = new Map([...holding("pl", 1, 30), ...holding("fr", 31, 60)])
        const after = new Map([...holding("de", 1, 20), ...holding("pl", 21, 30), ...holding("fr", 31, 60)])

        expect(flipsOf(landmassOf, before, after, [1, 31])).toEqual([{tiles: 30, was: "pl", is: "de"}])
    })

    it("leave out a landmass nothing touched, and a small island", () => {
        const before = holding("pl", 1, 65)
        const after = holding("de", 1, 65)

        expect(flipsOf(landmassOf, before, after, [1, 61])).toHaveLength(1)
        expect(flipsOf(landmassOf, before, after, [66])).toEqual([])
    })

    it("count a landmass emptied by bombs", () => {
        expect(flipsOf(landmassOf, holding("pl", 1, 30), new Map(), [1])).toEqual([{tiles: 30, was: "pl", is: ""}])
    })
})

describe("the look of a clip", () => {
    const flip = (tiles: number): Flip => ({tiles, was: "fr", is: "ps"})

    it("is painted flags when several big landmasses changed hands over a wide front", () => {
        expect(lookOf([flip(1500), flip(900)], 1.5)).toBe("flags")
    })

    it("dives into the tiles when a wide front changed one big landmass, or some smaller ones", () => {
        expect(lookOf([flip(2500)], 1.5)).toBe("dive")
        expect(lookOf([flip(1000), flip(600)], 1.5)).toBe("dive")
    })

    it("is tiles when nothing big enough to see from far changed hands", () => {
        expect(lookOf([flip(60), flip(40), flip(30)], 1.5)).toBe("tiles")
        expect(lookOf([], 1.5)).toBe("tiles")
    })

    it("dives rather than paint flags all along when the fight fits a view close enough to read the tiles", () => {
        expect(lookOf([flip(1500), flip(900)], TILES_ZOOM)).toBe("dive")
    })
})
