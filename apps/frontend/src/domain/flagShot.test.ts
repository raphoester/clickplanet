import {describe, expect, it} from "vitest"
import {cropAround, groundAround} from "./flagShot.ts"

// A line of tiles, each touching the one before and the one after.
const line = (tile: number) => [tile - 1, tile + 1].filter((t) => t >= 1 && t <= 20)

describe("groundAround", () => {
    it("takes the capital, then each ring of tiles around it once", () => {
        expect(groundAround(10, 0, line)).toEqual([10])
        expect(groundAround(10, 2, line)).toEqual([10, 9, 11, 8, 12])
        expect(groundAround(1, 3, line)).toEqual([1, 2, 3, 4])
    })
})

describe("cropAround", () => {
    it("centres the crop on the point, inside the frame", () => {
        const frame = {width: 1000, height: 600}
        expect(cropAround(frame, {x: 500, y: 300}, {width: 320, height: 180})).toEqual({x: 340, y: 210, width: 320, height: 180})
        expect(cropAround(frame, {x: 10, y: 590}, {width: 320, height: 180})).toEqual({x: 0, y: 420, width: 320, height: 180})
        expect(cropAround({width: 200, height: 100}, {x: 100, y: 50}, {width: 320, height: 180})).toEqual({x: 0, y: 0, width: 200, height: 100})
    })
})
