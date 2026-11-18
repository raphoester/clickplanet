import {describe, expect, it} from "vitest"
import {EARTH_1_SCALE, EARTH_1_SPACING, scaledCount, scaledLength, scaledZoom, scaleOf} from "./boardScale.ts"
import {EARTH_1, FIXTURE_1} from "../testdata/boards.ts"

describe("scaleOf", () => {
    it("is exactly 1 on earth-1, the reference", () => {
        expect(EARTH_1.spacing).toBe(EARTH_1_SPACING)
        expect(scaleOf(EARTH_1)).toEqual({length: 1, density: 1})
        expect(EARTH_1_SCALE).toEqual({length: 1, density: 1})
    })

    it("reads a coarser board's tiles as wider and fewer", () => {
        const {length, density} = scaleOf(FIXTURE_1)
        expect(length).toBeCloseTo(9.697, 3)
        expect(density).toBeCloseTo(1 / (9.697 * 9.697), 4)
    })
})

describe("on earth-1", () => {
    const scale = scaleOf(EARTH_1)

    it("leaves every count, length and zoom as it is", () => {
        for (const n of [1, 20, 50, 100, 400, 1000, 260503]) expect(scaledCount(n, scale)).toBe(n)
        for (const x of [0.004, 0.0046, 0.012, 0.4, 18 * 0.004]) expect(scaledLength(x, scale)).toBe(x)
        for (const z of [0.5, 9, 50]) expect(scaledZoom(z, scale)).toBe(z)
    })
})

describe("on fixture-1", () => {
    const scale = scaleOf(FIXTURE_1)

    it("counts as many tiles on the same ground, one at least", () => {
        expect(scaledCount(1000, scale)).toBe(11)
        expect(scaledCount(400, scale)).toBe(4)
        expect(scaledCount(50, scale)).toBe(1)
        expect(scaledCount(20, scale)).toBe(1)
    })

    it("measures a length in its own tiles, and a zoom as wide a tile on screen", () => {
        expect(scaledLength(0.004, scale)).toBeCloseTo(0.0388, 4)
        expect(scaledZoom(50, scale)).toBeCloseTo(5.156, 3)
    })
})
