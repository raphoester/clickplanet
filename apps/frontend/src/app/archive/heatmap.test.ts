import {describe, expect, it} from "vitest"
import {placeOf} from "./heatmap.ts"

// The globe's frame, as the backoffice reads it: y is north, -x is longitude 0, z is 90° east.
const at = (x: number, y: number, z: number) => Float32Array.from([x, y, z])

describe("placeOf", () => {
    it("reads a tile's latitude and longitude off its position", () => {
        expect(placeOf(at(-1, 0, 0), 1)).toEqual({latitude: 0, longitude: 0})
        expect(placeOf(at(0, 0, 1), 1).longitude).toBeCloseTo(90)
        expect(placeOf(at(0, 1, 0), 1).latitude).toBeCloseTo(90)
        expect(Math.abs(placeOf(at(1, 0, 0), 1).longitude)).toBeCloseTo(180)
    })
})
