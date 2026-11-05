import {describe, expect, it} from "vitest"
import {flightAt, territoryShotOf} from "./flight.ts"
import {Point} from "./clip/geometry.ts"

const onFront = (x: number, y: number): Point => ({x, y, z: Math.sqrt(1 - x * x - y * y)})

// A territory on the face toward +z, seen with the north pole up: x runs to the right, y up.
const patch = (left: number, right: number, bottom: number, top: number): Point[] =>
    [left, (left + right) / 2, right].flatMap((x) => [bottom, (bottom + top) / 2, top].map((y) => onFront(x, y)))

const ASPECT = 1.6

describe("territoryShotOf", () => {
    it("centres the territory and fits its wider side to the screen", () => {
        const shot = territoryShotOf(patch(-0.2, 0.2, -0.1, 0.1), ASPECT, 1, 50)!

        expect(shot.direction.x).toBeCloseTo(0)
        expect(shot.direction.y).toBeCloseTo(0)
        expect(shot.zoom).toBeCloseTo(0.6 * ASPECT / 0.2)
    })

    it("fits a tall territory by its height", () => {
        const shot = territoryShotOf(patch(-0.05, 0.05, -0.25, 0.25), ASPECT, 1, 50)!

        expect(shot.zoom).toBeCloseTo(0.6 / 0.25)
    })

    it("centres on the middle of the territory's extent, not on where most of its tiles are", () => {
        const lopsided = [...patch(0.1, 0.12, -0.05, 0.05), ...patch(0.12, 0.3, 0, 0.01)]

        const shot = territoryShotOf(lopsided, ASPECT, 1, 50)!

        expect(shot.direction.x).toBeCloseTo(0.2, 2)
    })

    it("never goes out past the tiles nor closer than the largest tiles", () => {
        expect(territoryShotOf(patch(-0.6, 0.6, -0.6, 0.6), ASPECT, 5, 20)!.zoom).toBe(5)
        expect(territoryShotOf([onFront(0.1, 0.1)], ASPECT, 5, 20)!.zoom).toBe(20)
    })

    it("has nothing to fly to without a tile", () => {
        expect(territoryShotOf([], ASPECT, 5, 20)).toBeUndefined()
    })
})

describe("flightAt", () => {
    const from = {direction: {x: 0, y: 0, z: 1}, zoom: 1}
    const to = {direction: {x: 1, y: 0, z: 0}, zoom: 9}

    it("leaves from the first shot and lands on the second", () => {
        expect(flightAt(from, to, 0)).toEqual(from)
        expect(flightAt(from, to, 1).zoom).toBeCloseTo(9)
        expect(flightAt(from, to, 1).direction.x).toBeCloseTo(1)
        expect(flightAt(from, to, 2).zoom).toBeCloseTo(9)
    })

    it("turns halfway and zooms by the same ratio each step, at mid-flight", () => {
        const middle = flightAt(from, to, 0.5)

        expect(middle.zoom).toBeCloseTo(3)
        expect(middle.direction.x).toBeCloseTo(Math.SQRT1_2)
        expect(middle.direction.z).toBeCloseTo(Math.SQRT1_2)
    })

    it("starts and lands softly", () => {
        expect(flightAt(from, to, 0.05).zoom - 1).toBeLessThan((9 - 1) * 0.05)
        expect(9 - flightAt(from, to, 0.95).zoom).toBeLessThan((9 - 1) * 0.05)
    })
})
