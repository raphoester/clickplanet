import {describe, expect, it} from "vitest"
import {placesOf} from "./places.ts"

// A coordinates file as /map writes it: "CPCO", version 1, the count, x y z per tile, then u v per tile.
export function coordinatesOf(points: [number, number][]): ArrayBuffer {
    const buffer = new ArrayBuffer(12 + points.length * 20)
    const view = new DataView(buffer)
    "CPCO".split("").forEach((letter, i) => view.setUint8(i, letter.charCodeAt(0)))
    view.setUint32(4, 1, true)
    view.setUint32(8, points.length, true)
    points.forEach(([latitude, longitude], i) => {
        const lat = (latitude * Math.PI) / 180, lon = (longitude * Math.PI) / 180
        view.setFloat32(12 + i * 12, -Math.cos(lat) * Math.cos(lon), true)
        view.setFloat32(16 + i * 12, Math.sin(lat), true)
        view.setFloat32(20 + i * 12, Math.cos(lat) * Math.sin(lon), true)
    })
    return buffer
}

describe("placesOf", () => {
    it("reads each tile's latitude and longitude as the map's generator does", () => {
        const places = placesOf(coordinatesOf([[45.5, 10.5], [-30, -60]]))

        expect(places.latitudes[0]).toBeCloseTo(45.5, 4)
        expect(places.longitudes[0]).toBeCloseTo(10.5, 4)
        expect(places.latitudes[1]).toBeCloseTo(-30, 4)
        expect(places.longitudes[1]).toBeCloseTo(-60, 4)
    })

    it("refuses a file that is not the map's", () => {
        expect(() => placesOf(new ArrayBuffer(12))).toThrow("not a coordinates file")
        expect(() => placesOf(coordinatesOf([[0, 0]]).slice(0, 20))).toThrow("truncated")
    })
})
