import {describe, expect, it} from "vitest"
import {FORTIFY_WAVE_SECONDS, reached, wavePlan} from "./fortifyWave.ts"

const positions = new Float32Array([
    1, 0, 0,
    Math.cos(0.1), Math.sin(0.1), 0,
    Math.cos(0.2), Math.sin(0.2), 0,
    Math.cos(0.4), Math.sin(0.4), 0,
])

describe("wavePlan", () => {
    it("reaches the closing tile first and the farthest tile when the wave ends", () => {
        const plan = wavePlan([4, 2, 1, 3], positions, 1)

        expect([...plan.tiles]).toEqual([1, 2, 3, 4])
        expect(plan.reach).toBeCloseTo(0.4)
        expect(plan.arrivals[0]).toBe(0)
        expect(plan.arrivals[1]).toBeCloseTo(FORTIFY_WAVE_SECONDS / 4)
        expect(plan.arrivals[3]).toBeCloseTo(FORTIFY_WAVE_SECONDS)
    })

    it("reaches a one-tile landmass at once", () => {
        const plan = wavePlan([3], positions, 3)

        expect(plan.reach).toBe(0)
        expect([...plan.arrivals]).toEqual([0])
    })
})

describe("reached", () => {
    it("counts the tiles the front has passed", () => {
        const plan = wavePlan([1, 2, 3, 4], positions, 1)

        expect(reached(plan, 0)).toBe(1)
        expect(reached(plan, FORTIFY_WAVE_SECONDS / 2)).toBe(3)
        expect(reached(plan, FORTIFY_WAVE_SECONDS)).toBe(4)
    })
})
