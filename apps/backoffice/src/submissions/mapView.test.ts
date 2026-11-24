import {describe, expect, it} from "vitest"
import {distanceKm} from "../flags/pin.ts"
import {
    boxOf, clampedOf, isWorld, kmAcross, onGlobe, pannedBy, placeAt, pointOf, regionOf, resizedTo, roundedOf, worldOf, zoomedAt,
} from "./mapView.ts"

const BARCELONA = {latitude: 41.3874, longitude: 2.1686}
const OSLO = {latitude: 59.9139, longitude: 10.7522}

describe("a view of the planisphere", () => {
    it("opens on a capital's region, 2,000 km across at its latitude, whatever the latitude", () => {
        for (const capital of [BARCELONA, OSLO]) {
            const view = regionOf(capital, 640, 320)
            expect(pointOf(view, capital)).toEqual([320, 160])
            expect(kmAcross(view)).toBeCloseTo(2000, 0)
            const east = placeAt(view, 640, 160)
            expect(distanceKm(capital, {latitude: capital.latitude, longitude: east.longitude})).toBeGreaterThan(950)
        }
    })

    it("shows the whole world at most, and a degree and a half at least", () => {
        const world = worldOf(640, 427)
        expect(isWorld(world)).toBe(true)
        expect(kmAcross(world)).toBeUndefined()
        expect(world.latitude).toBe(0)
        expect(zoomedAt(world, 0.5, 100, 100)).toEqual(world)
        const closest = zoomedAt(regionOf(BARCELONA, 640, 320), 1000, 320, 160)
        expect(640 / closest.scale).toBeCloseTo(1.5, 6)
    })

    it("keeps the poles at the frame's edges", () => {
        const view = regionOf(BARCELONA, 640, 320)
        const north = pannedBy(view, 0, 100_000)
        expect(north.latitude + 160 / north.scale).toBeCloseTo(90, 6)
        expect(onGlobe(north, 0)).toBe(true)
        expect(onGlobe(worldOf(640, 427), 2)).toBe(false)
    })

    it("follows a drag, and zooms about the pointer", () => {
        const view = regionOf(BARCELONA, 640, 320)
        const under = placeAt(view, 100, 50)
        expect(pointOf(pannedBy(view, 30, -20), under).map(Math.round)).toEqual([130, 30])
        const zoomed = zoomedAt(view, 3, 100, 50)
        expect(placeAt(zoomed, 100, 50).latitude).toBeCloseTo(under.latitude, 9)
        expect(placeAt(zoomed, 100, 50).longitude).toBeCloseTo(under.longitude, 9)
    })

    it("goes on round the world past the antimeridian", () => {
        const view = pannedBy(regionOf({latitude: -17.7134, longitude: 178.065}, 640, 320), -200, 0)
        expect(view.longitude).toBeGreaterThan(180)
        const [x] = pointOf(view, {latitude: -17, longitude: -179})
        expect(x).toBeCloseTo(320 + (181 - view.longitude) * view.scale, 6)
        expect(placeAt(view, 320, 160).longitude).toBeCloseTo(view.longitude - 360, 9)
    })

    it("keeps the same places across a frame resized", () => {
        const view = regionOf(BARCELONA, 640, 320)
        expect(kmAcross(resizedTo(view, 343, 229))).toBeCloseTo(2000, 0)
    })

    it("cuts the land's box at the poles, at the device's pixels", () => {
        const box = boxOf(clampedOf({longitude: 0, latitude: 80, scale: 10, width: 400, height: 200}), 2, 0.25)
        expect(box.north).toBe(90)
        expect(box.scale).toBe(20)
        expect(box.west).toBeCloseTo(-30, 9)
        expect(box.width).toBe(1200)
    })

    it("rounds a place to four decimals, about ten metres", () => {
        expect(roundedOf({latitude: 41.387449, longitude: -2.16865})).toEqual({latitude: 41.3874, longitude: -2.1686})
    })
})
