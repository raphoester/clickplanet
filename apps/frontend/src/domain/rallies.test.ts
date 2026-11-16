import {describe, expect, it} from "vitest"
import {clickCostOf, covers, Rallies, Rally} from "./rallies.ts"

// Tile n sits at n hundredths of a radian along a great circle from tile 1.
const positions = Float32Array.from(Array.from({length: 100}, (_, i) => [Math.cos(i / 100), Math.sin(i / 100), 0]).flat())

const rally = (id: string, countryId: string, tile: number, endsAt = 1000): Rally => ({
    id, accountId: "a", countryId, tile, radius: 0.05, endsAt,
})

describe("a rally's ring", () => {
    it("covers every tile within its radius of its centre", () => {
        expect([6, 8, 10, 12, 14].map((tile) => covers(rally("r", "fr", 10), tile, positions))).toEqual([true, true, true, true, true])
        expect([4, 16, 80].map((tile) => covers(rally("r", "fr", 10), tile, positions))).toEqual([false, false, false])
        expect(covers(rally("r", "fr", 10), 0, positions)).toBe(false)
    })
})

describe("what a click costs", () => {
    const rings = [rally("a", "fr", 10), rally("b", "fr", 12)]

    it("is the rally's cost inside a ring of the flag, never less where rings overlap", () => {
        expect(clickCostOf(11, "fr", rings, positions, 0.5)).toBe(0.5)
    })

    it("is a whole click outside, for another flag, with no ring on, or with no cost known", () => {
        expect(clickCostOf(40, "fr", rings, positions, 0.5)).toBe(1)
        expect(clickCostOf(11, "de", rings, positions, 0.5)).toBe(1)
        expect(clickCostOf(11, "fr", [], positions, 0.5)).toBe(1)
        expect(clickCostOf(11, "fr", rings, positions, undefined)).toBe(1)
    })
})

describe("the rallies on", () => {
    it("keeps one per id, forgets the ones over, and takes the server's list whole", () => {
        const rallies = new Rallies()
        rallies.keep(rally("a", "fr", 10, 100))
        rallies.keep(rally("a", "fr", 20, 200))
        rallies.keep(rally("b", "de", 30, 300))

        expect(rallies.on(0).map((r) => r.tile)).toEqual([20, 30])
        expect(rallies.prune(250)).toBe(true)
        expect(rallies.on(250).map((r) => r.id)).toEqual(["b"])
        expect(rallies.prune(250)).toBe(false)

        rallies.replace([rally("c", "it", 40)])
        expect(rallies.on(0).map((r) => r.id)).toEqual(["c"])
    })
})
