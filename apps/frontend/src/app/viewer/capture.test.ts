import {describe, expect, it} from "vitest"
import {flipRows} from "./capture.ts"

/** One byte per pixel spelled out as RGBA, so a row reads as its own number. */
function image(rows: number[][]): Uint8ClampedArray {
    return new Uint8ClampedArray(rows.flatMap((row) => row.flatMap((pixel) => [pixel, pixel, pixel, 255])))
}

// GL hands back the bottom row first. Getting this wrong does not fail loudly —
// it ships an upside-down planet.
describe("flipRows", () => {
    it("turns the buffer the right way up", () => {
        const flipped = flipRows(image([[1, 2], [3, 4], [5, 6]]), 2, 3)

        expect(flipped).toEqual(image([[5, 6], [3, 4], [1, 2]]))
    })

    it("keeps the channels of a pixel in order", () => {
        const pixels = new Uint8ClampedArray([1, 2, 3, 4, 5, 6, 7, 8])

        expect(Array.from(flipRows(pixels, 1, 2))).toEqual([5, 6, 7, 8, 1, 2, 3, 4])
    })

    it("leaves a single row alone", () => {
        expect(flipRows(image([[7, 8, 9]]), 3, 1)).toEqual(image([[7, 8, 9]]))
    })

    it("does not write over the buffer it was given", () => {
        const original = image([[1], [2]])
        flipRows(original, 1, 2)

        expect(original).toEqual(image([[1], [2]]))
    })
})
