import {describe, expect, it} from "vitest"
import {colorAt, Grid, heatOf, placeOf, rgbOf} from "./heat.ts"

function gridOf(cells: {land: number, changes: number}[]): Grid {
    return {
        columns: cells.length, rows: 1,
        land: Uint8Array.from(cells.map(cell => cell.land)),
        changes: cells.map(cell => cell.changes),
    }
}

describe("heatOf", () => {
    it("reads changes per tile, so a cell of few tiles far north is not read as calmer", () => {
        const heat = heatOf(gridOf([
            {land: 20, changes: 20},
            {land: 1, changes: 3},
            {land: 5, changes: 0},
            {land: 0, changes: 0},
        ]))

        expect(heat.hottest).toBe(1)
        expect(heat.calmest).toBe(1)
        expect(heat.busiest).toBe(3)
        expect([...heat.levels]).toEqual([0, 1, 0, 0])
    })

    it("puts the cells between on a log scale", () => {
        const heat = heatOf(gridOf([{land: 1, changes: 1}, {land: 1, changes: 10}, {land: 1, changes: 100}]))

        expect(heat.levels[1]).toBeCloseTo(0.5)
    })

    it("has nothing to say of a span where no tile changed hands", () => {
        expect(heatOf(gridOf([{land: 3, changes: 0}]))).toEqual({
            levels: new Float32Array(1), calmest: 0, busiest: 0, hottest: undefined,
        })
    })
})

describe("the colors", () => {
    it("reads a hex color", () => {
        expect(rgbOf("#2a78d6")).toEqual([42, 120, 214])
        expect(rgbOf(" #fff")).toEqual([255, 255, 255])
    })

    it("spreads the stops of a ramp evenly from 0 to 1", () => {
        const stops: [number, number, number][] = [[0, 0, 0], [100, 100, 100], [200, 0, 0]]

        expect(colorAt(0, stops)).toEqual([0, 0, 0])
        expect(colorAt(0.25, stops)).toEqual([50, 50, 50])
        expect(colorAt(1, stops)).toEqual([200, 0, 0])
    })
})

describe("placeOf", () => {
    it("names a cell by the degrees it starts from, toward the equator and the prime meridian", () => {
        expect(placeOf(44 * 360 + 190, 360)).toBe("45°N 10°E")
        expect(placeOf(120 * 360 + 119, 360)).toBe("30°S 60°W")
    })
})
