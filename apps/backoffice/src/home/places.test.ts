import {describe, expect, it} from "vitest"
import {placesOf} from "./places.ts"

function positionsOf(points: [number, number][]): Float32Array {
    return Float32Array.from(points.flatMap(([latitude, longitude]) => {
        const lat = (latitude * Math.PI) / 180, lon = (longitude * Math.PI) / 180
        return [-Math.cos(lat) * Math.cos(lon), Math.sin(lat), Math.cos(lat) * Math.sin(lon)]
    }))
}

describe("placesOf", () => {
    it("reads each tile's latitude and longitude as the map's generator does", () => {
        const places = placesOf(positionsOf([[45.5, 10.5], [-30, -60]]))

        expect(places.latitudes[0]).toBeCloseTo(45.5, 4)
        expect(places.longitudes[0]).toBeCloseTo(10.5, 4)
        expect(places.latitudes[1]).toBeCloseTo(-30, 4)
        expect(places.longitudes[1]).toBeCloseTo(-60, 4)
    })
})
