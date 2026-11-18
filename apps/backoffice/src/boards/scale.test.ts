import {describe, expect, it} from "vitest"
import {EARTH_1_SPACING, scaleOf} from "./scale.ts"
import {EARTH_1, FIXTURE_1} from "./testBoards.ts"

// Every number the backoffice tunes on earth-1, and a few of the design's (docs/boards.md).
const LENGTHS = [0.3, 0.05, 0.0075, 0.4, 18, 0.004063093462255102, 1e-9]
const COUNTS = [1, 12, 20, 25, 50, 150, 300, 1000, 3000, 260503]
const ZOOMS = [20, 50, 1.5]

describe("the scale of a board", () => {
    it("is earth-1's own spacing", () => {
        expect(EARTH_1.spacing).toBe(EARTH_1_SPACING)
    })

    it("leaves every number exactly as it was on earth-1", () => {
        const scale = scaleOf(EARTH_1)

        expect(scale.lengthRatio).toBe(1)
        expect(scale.densityRatio).toBe(1)
        for (const x of LENGTHS) expect(Object.is(scale.length(x), x)).toBe(true)
        for (const n of COUNTS) expect(scale.tiles(n)).toBe(n)
        for (const z of ZOOMS) expect(Object.is(scale.zoom(z), z)).toBe(true)
    })

    it("reads fixture-1, about 9.7 times as far apart: lengths longer, counts fewer, zooms nearer", () => {
        const scale = scaleOf(FIXTURE_1)

        expect(scale.lengthRatio).toBeCloseTo(9.697, 3)
        expect(scale.densityRatio).toBeCloseTo(1 / 9.697 ** 2, 5)
        expect(scale.densityRatio * scale.lengthRatio * scale.lengthRatio).toBeCloseTo(1, 12)
        expect(scale.length(0.3)).toBeCloseTo(2.909, 3)
        expect(scale.length(0.05)).toBeCloseTo(0.4849, 4)
        expect(scale.zoom(50)).toBeCloseTo(5.156, 3)
        // A count rounds, and never goes under one tile.
        expect(scale.tiles(1000)).toBe(11)
        expect(scale.tiles(150)).toBe(2)
        expect(scale.tiles(20)).toBe(1)
        expect(scale.tiles(1)).toBe(1)
    })

    it("follows the spacing both ways: a board twice as dense halves a length and doubles a zoom", () => {
        const scale = scaleOf({spacing: EARTH_1_SPACING / 2})

        expect(scale.length(0.3)).toBeCloseTo(0.15, 12)
        expect(scale.zoom(50)).toBeCloseTo(100, 10)
        expect(scale.tiles(1000)).toBe(4000)
    })

    it("refuses a spacing that is no length", () => {
        for (const spacing of [0, -1, NaN, Infinity]) expect(() => scaleOf({spacing})).toThrow("a board's spacing")
    })
})
