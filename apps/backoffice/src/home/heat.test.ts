import {describe, expect, it} from "vitest"
import {blur, busiestOf, cellOf, cellsOf, colorAt, COLUMNS, levelsOf, pictureOf, placeOf, rgbOf} from "./heat.ts"
import {Places} from "./places.ts"

function placesAt(points: [number, number][]): Places {
    return {
        latitudes: Float32Array.from(points.map(([latitude]) => latitude)),
        longitudes: Float32Array.from(points.map(([, longitude]) => longitude)),
    }
}

describe("the cells", () => {
    it("count the tiles of each degree and how often they changed hands, and name the busiest", () => {
        const places = placesAt([[45.5, 10.5], [45.2, 10.9], [-30.5, -60.5]])
        const cells = cellsOf(places, Uint32Array.from([0, 3, 1, 2]))

        const north = cellOf(45.5, 10.5)
        expect(north).toBe(44 * COLUMNS + 190)
        expect([cells.tiles[north], cells.changes[north]]).toEqual([2, 4])
        expect(busiestOf(cells)).toBe(north)
        expect(placeOf(north)).toBe("45°N 10°E")
        expect(placeOf(cellOf(-30.5, -60.5))).toBe("30°S 60°W")
    })

    it("have no busiest when nothing changed hands", () => {
        expect(busiestOf(cellsOf(placesAt([[0, 0]]), new Uint32Array(2)))).toBeUndefined()
    })
})

describe("the picture", () => {
    function landOf(places: Places, width: number): number[] {
        const picture = pictureOf(places, new Uint32Array(places.latitudes.length + 1), width)
        return [...picture.land].flatMap((land, pixel) => land === 1 ? [pixel] : [])
    }

    it("draws a tile by the equator on the pixel under it", () => {
        expect(landOf(placesAt([[0.5, 0.5]]), 360)).toEqual([89 * 360 + 180])
    })

    it("draws a tile on the antimeridian on both edges", () => {
        expect(landOf(placesAt([[0.5, 179.9]]), 360)).toEqual([89 * 360, 89 * 360 + 359])
    })

    it("stretches a tile far north over the longitudes it covers", () => {
        expect(landOf(placesAt([[85, 0]]), 360).length).toBeGreaterThan(3)
    })

    it("glows over the land around a change, and nowhere on the sea", () => {
        const picture = pictureOf(placesAt([[0.5, 0.5], [0.5, 2.5]]), Uint32Array.from([0, 5, 0]), 360)
        const levels = levelsOf(picture)

        expect(levels[89 * 360 + 180]).toBe(1)
        expect(levels[89 * 360 + 182]).toBeGreaterThan(0)
        expect(levels[89 * 360 + 181]).toBe(0)
    })
})

describe("blur", () => {
    it("spreads a value round the world without losing any of it", () => {
        const values = new Float32Array(4 * 3)
        values[4] = 9

        const blurred = blur(values, 4, 3, 1)

        expect(blurred.reduce((sum, value) => sum + value, 0)).toBeCloseTo(9, 4)
        expect(blurred[7]).toBeCloseTo(1, 4)
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
