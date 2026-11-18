import {describe, expect, it} from "vitest"
import {Board} from "../boards/boards.ts"
import {Places} from "../boards/places.ts"
import {EARTH_1, FIXTURE_1, placesOn} from "../boards/testBoards.ts"
import {blur, busiestOf, cellOf, cellsOf, colorAt, COLUMNS, levelsOf, Picture, pictureOf, placeOf, rgbOf} from "./heat.ts"

function placesAt(points: [number, number][], board: Board = EARTH_1): Places {
    return {
        board,
        latitudes: Float32Array.from(points.map(([latitude]) => latitude)),
        longitudes: Float32Array.from(points.map(([, longitude]) => longitude)),
    }
}

const landShare = (picture: Picture) => picture.land.reduce((sum, land) => sum + land, 0) / picture.land.length

// The picture as the page drew it before boards were told apart: earth-1's 0.3° tile, on the row under it.
function pictureBefore(places: Places, changes: Uint32Array, width: number): Picture {
    const height = width / 2
    const pixelDegrees = 360 / width
    const land = new Uint8Array(width * height)
    const raw = new Float32Array(width * height)
    for (let i = 0; i < places.latitudes.length; i++) {
        const latitude = places.latitudes[i]
        const y = Math.min(height - 1, Math.max(0, Math.floor(((90 - latitude) / 180) * height)))
        const x = ((places.longitudes[i] + 180) / 360) * width
        const half = Math.min(width / 2, Math.max(0.5,
            0.3 / 2 / Math.max(Math.cos((latitude * Math.PI) / 180), 1e-3) / pixelDegrees))
        const changed = changes[i + 1] ?? 0
        for (let column = Math.floor(x - half); column < Math.ceil(x + half); column++) {
            const pixel = y * width + ((column % width) + width) % width
            land[pixel] = 1
            raw[pixel] += changed
        }
    }
    let heat = raw
    for (let pass = 0; pass < 3; pass++) heat = blur(heat, width, height, 2)
    return {width, height, land, heat}
}

// Every 97th tile changed hands, some many times.
function changesOn(places: Places): Uint32Array {
    const changes = new Uint32Array(places.latitudes.length + 1)
    for (let tile = 1; tile < changes.length; tile += 97) changes[tile] = 1 + (tile % 5)
    return changes
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

describe("the picture on a board", () => {
    it("draws earth-1 pixel for pixel as it did before boards were told apart", () => {
        const places = placesOn(EARTH_1)
        const changes = changesOn(places)

        const picture = pictureOf(places, changes, 1440)
        const before = pictureBefore(places, changes, 1440)

        // Bytes, not elements: a million of each.
        expect(Buffer.from(picture.land.buffer).equals(Buffer.from(before.land.buffer))).toBe(true)
        expect(Buffer.from(picture.heat.buffer).equals(Buffer.from(before.heat.buffer))).toBe(true)
        expect(picture.heat.some(heat => heat > 0)).toBe(true)
    })

    it("draws fixture-1's tiles as wide as its spacing, so its land covers the picture as earth-1's does", () => {
        const earth = placesOn(EARTH_1)
        const fixture = placesOn(FIXTURE_1)
        const earthLand = landShare(pictureOf(earth, new Uint32Array(earth.latitudes.length + 1), 1440))

        const fixtureLand = landShare(pictureOf(fixture, new Uint32Array(fixture.latitudes.length + 1), 1440))
        // Its tiles drawn at earth-1's width instead: specks a tenth of a cell apart.
        const unscaled = landShare(pictureOf({...fixture, board: EARTH_1}, new Uint32Array(fixture.latitudes.length + 1), 1440))

        expect(fixtureLand / earthLand).toBeGreaterThan(0.8)
        expect(fixtureLand / earthLand).toBeLessThan(1.25)
        expect(unscaled / earthLand).toBeLessThan(0.1)
    })

    it("glows where a tile of fixture-1 changed hands, over the rows its tile covers", () => {
        const fixture = placesOn(FIXTURE_1)
        const changes = new Uint32Array(fixture.latitudes.length + 1)
        changes[1] = 3
        const picture = pictureOf(fixture, changes, 1440)
        const levels = levelsOf(picture)
        const y = Math.floor(((90 - fixture.latitudes[0]) / 180) * 720)
        const x = Math.floor(((fixture.longitudes[0] + 180) / 360) * 1440)

        expect(levels[y * 1440 + x]).toBeGreaterThan(0.9)
        expect(levels[(y + 4) * 1440 + x]).toBeGreaterThan(0)
        expect(levels[(y - 4) * 1440 + x]).toBeGreaterThan(0)
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
