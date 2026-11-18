import {afterEach, describe, expect, it, vi} from "vitest"
import {loadPlaces, placesOf} from "./places.ts"
import {EARTH_1, FIXTURE_1, runsOf} from "./testBoards.ts"

function positionsOf(points: [number, number][]): Float32Array {
    return Float32Array.from(points.flatMap(([latitude, longitude]) => {
        const lat = (latitude * Math.PI) / 180, lon = (longitude * Math.PI) / 180
        return [-Math.cos(lat) * Math.cos(lon), Math.sin(lat), Math.cos(lat) * Math.sin(lon)]
    }))
}

afterEach(() => {
    vi.unstubAllGlobals()
})

describe("placesOf", () => {
    it("reads each tile's latitude and longitude as the map's generator does, on the board it names", () => {
        const places = placesOf(FIXTURE_1, positionsOf([[45.5, 10.5], [-30, -60]]))

        expect(places.board).toBe(FIXTURE_1)
        expect(places.latitudes[0]).toBeCloseTo(45.5, 4)
        expect(places.longitudes[0]).toBeCloseTo(10.5, 4)
        expect(places.latitudes[1]).toBeCloseTo(-30, 4)
        expect(places.longitudes[1]).toBeCloseTo(-60, 4)
    })
})

describe("loadPlaces", () => {
    it("reads a board's runs once a page load, and keeps them by its name", async () => {
        const fetched = vi.fn((url: string) => {
            void url
            return Promise.resolve(new Response(runsOf(EARTH_1)))
        })
        vi.stubGlobal("fetch", fetched)

        const [first, again] = await Promise.all([loadPlaces("earth-1"), loadPlaces("earth-1")])

        expect(first).toBe(again)
        expect(first.board.name).toBe("earth-1")
        expect(first.latitudes.length).toBe(EARTH_1.tiles)
        expect(await loadPlaces("earth-1")).toBe(first)
        expect(fetched).toHaveBeenCalledTimes(1)
        expect(fetched.mock.calls[0][0]).toMatch(/runs-[0-9a-f]{8}\.bin/)
    })

    it("refuses a board the page does not carry, without a fetch", async () => {
        const fetched = vi.fn()
        vi.stubGlobal("fetch", fetched)

        await expect(loadPlaces("fixture-1")).rejects.toThrow("fixture-1 is not a board this page carries")
        await expect(loadPlaces("")).rejects.toThrow("the answer names no board")
        expect(fetched).not.toHaveBeenCalled()
    })
})
