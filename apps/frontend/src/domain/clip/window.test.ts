import {describe, expect, it} from "vitest"
import {aroundCell, cellOf} from "./geometry.ts"
import {candidatesOf, inCandidate, Moment, newsSince} from "./window.ts"

const HOUR = 3_600_000
const paris = cellOf({x: 0.03, y: 0.75, z: 0.66})
const sydney = cellOf({x: 0.5, y: -0.55, z: -0.67})

function burst(cell: number, from: number, count: number, captured = true): Moment[] {
    return Array.from({length: count}, (_, i) => ({at: from + i * 1000, cell, captured}))
}

describe("the candidate windows", () => {
    it("find each length's busiest stretch around a place", () => {
        const moments = [...burst(paris, 10 * HOUR, 500), ...burst(paris, 30 * HOUR, 50)]

        const hour = candidatesOf(moments, 0, 72 * HOUR, 1)

        expect(hour).toHaveLength(1)
        expect(hour[0].since).toBeLessThanOrEqual(10 * HOUR)
        expect(hour[0].until).toBeGreaterThan(10 * HOUR)
        expect(inCandidate(hour[0], paris)).toBe(true)
    })

    it("offer places far apart, not the same war twice", () => {
        const moments = [...burst(paris, 0, 300), ...burst(paris + 1, 0, 300), ...burst(sydney, 0, 100)]

        const places = candidatesOf(moments, 0, 2 * HOUR, 1).map(({cell}) => cell)

        expect(places).toHaveLength(2)
        expect(places.some((cell) => aroundCell(cell).includes(sydney))).toBe(true)
    })

    it("count only tiles taken from another flag", () => {
        expect(candidatesOf(burst(paris, 0, 100, false), 0, HOUR, 1)).toEqual([])
    })
})

describe("the cells", () => {
    it("wrap round the antimeridian and stop at the poles", () => {
        expect(aroundCell(0)).toHaveLength(6)
        expect(aroundCell(paris)).toHaveLength(9)
    })
})

describe("the news", () => {
    it("is the last 12 hours of a replay, when no window is asked for", () => {
        expect(newsSince(0, 72 * HOUR)).toBe(60 * HOUR)
    })

    it("is all of a replay shorter than that", () => {
        expect(newsSince(70 * HOUR, 72 * HOUR)).toBe(70 * HOUR)
    })
})
